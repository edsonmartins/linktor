package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupLibraryTestService() (*TemplateService, *mockTemplateRepository) {
	svc, templateRepo := setupTemplateService()
	channelRepo := svc.channelRepo.(*testutil.MockChannelRepository)
	channelRepo.Channels["ch-1"] = &entity.Channel{
		ID: "ch-1", TenantID: "tenant-1", Type: entity.ChannelTypeWhatsAppOfficial,
		Credentials: map[string]string{"access_token": "test-token", "waba_id": "waba-1"},
	}
	return svc, templateRepo
}

func TestListTemplateLibrary_HitsCorrectEndpoint(t *testing.T) {
	svc, _ := setupLibraryTestService()

	var capturedURL string
	svc.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		capturedURL = r.URL.String()
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(
				`{"data":[{"name":"payment_reminder_1","category":"UTILITY","body_text":"Pay by {{1}}"}]}`,
			)),
		}, nil
	})

	items, err := svc.ListTemplateLibrary(context.Background(), "ch-1", LibraryQuery{
		Search:   "payment",
		Language: "pt_BR",
	})
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "payment_reminder_1", items[0].Name)
	assert.Contains(t, capturedURL, "/message_template_library")
	assert.Contains(t, capturedURL, "search=payment")
	assert.Contains(t, capturedURL, "language=pt_BR")
}

func TestListTemplateLibrary_MissingChannel(t *testing.T) {
	svc, _ := setupTemplateService()
	_, err := svc.ListTemplateLibrary(context.Background(), "nonexistent", LibraryQuery{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "credentials")
}

func TestListTemplateLibrary_EncodesSpecialCharacters(t *testing.T) {
	// A search term with '&', '=', or spaces must be percent-encoded,
	// otherwise it would either inject new query params or produce an
	// invalid URL that Meta rejects.
	svc, _ := setupLibraryTestService()

	var capturedRawQuery string
	svc.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		capturedRawQuery = r.URL.RawQuery
		return &http.Response{
			StatusCode: http.StatusOK, Header: make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{"data":[]}`)),
		}, nil
	})

	_, err := svc.ListTemplateLibrary(context.Background(), "ch-1", LibraryQuery{
		Search: "hello world & co",
		Topic:  "plan=basic",
	})
	require.NoError(t, err)

	// After decoding, the values must match exactly what we sent.
	decoded, err := url.ParseQuery(capturedRawQuery)
	require.NoError(t, err)
	assert.Equal(t, "hello world & co", decoded.Get("search"))
	assert.Equal(t, "plan=basic", decoded.Get("topic"))
	// And the raw wire format must not contain the literal special chars.
	assert.NotContains(t, capturedRawQuery, "hello world")
	assert.Contains(t, capturedRawQuery, "%26") // '&' encoded
}

func TestCreateFromLibrary_SendsCorrectPayload(t *testing.T) {
	svc, templateRepo := setupLibraryTestService()

	var captured map[string]interface{}
	var capturedURL string
	svc.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		capturedURL = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"id":"hsm-lib-1","status":"APPROVED","category":"UTILITY"}`)),
		}, nil
	})

	tmpl, err := svc.CreateFromLibrary(context.Background(), &CreateFromLibraryInput{
		TenantID:            "tenant-1",
		ChannelID:           "ch-1",
		Name:                "my_delivery",
		Language:            "pt_BR",
		Category:            entity.TemplateCategoryUtility,
		LibraryTemplateName: "delivery_update_1",
		LibraryTemplateButtonInputs: []map[string]interface{}{
			{"type": "URL", "url": map[string]interface{}{"base_url": "https://linktor.dev/{{1}}"}},
		},
		LibraryTemplateBodyInputs: map[string]interface{}{
			"add_contact_number": true,
		},
	})
	require.NoError(t, err)

	assert.Equal(t, "hsm-lib-1", tmpl.ExternalID)
	// Library templates often land APPROVED immediately since they're pre-vetted
	assert.Equal(t, entity.TemplateStatusApproved, tmpl.Status)
	assert.Contains(t, capturedURL, "/waba-1/message_templates")

	assert.Equal(t, "my_delivery", captured["name"])
	assert.Equal(t, "delivery_update_1", captured["library_template_name"])
	assert.NotNil(t, captured["library_template_body_inputs"])
	assert.NotNil(t, captured["library_template_button_inputs"])
	assert.Len(t, templateRepo.Templates, 1)
}

func TestCreateFromLibrary_RequiresName(t *testing.T) {
	svc, _ := setupLibraryTestService()
	_, err := svc.CreateFromLibrary(context.Background(), &CreateFromLibraryInput{
		ChannelID: "ch-1",
		Name:      "foo",
		Language:  "pt_BR",
		Category:  entity.TemplateCategoryUtility,
		// LibraryTemplateName omitted
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "library_template_name")
}

func TestCreateFromLibrary_DuplicateRejected(t *testing.T) {
	svc, templateRepo := setupLibraryTestService()
	templateRepo.Templates["existing"] = &entity.Template{
		TenantID: "tenant-1", ChannelID: "ch-1",
		Name: "my_delivery", Language: "pt_BR",
	}

	_, err := svc.CreateFromLibrary(context.Background(), &CreateFromLibraryInput{
		TenantID:            "tenant-1",
		ChannelID:           "ch-1",
		Name:                "my_delivery",
		Language:            "pt_BR",
		Category:            entity.TemplateCategoryUtility,
		LibraryTemplateName: "delivery_update_1",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}
