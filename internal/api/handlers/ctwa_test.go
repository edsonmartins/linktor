package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCTWAHandler(t *testing.T) {
	h := NewCTWAHandler()
	require.NotNil(t, h)
	assert.NotNil(t, h.clients)
}

func TestCTWAHandler_GetReferral_NoClient(t *testing.T) {
	h := NewCTWAHandler()
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/ctwa/referrals/ref-1", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}, {Key: "referralId", Value: "ref-1"}}

	h.GetReferral(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "not found")
}

func TestCTWAHandler_GetReferralByPhone_NoClient(t *testing.T) {
	h := NewCTWAHandler()
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/ctwa/referrals/phone/+5511999999999", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}, {Key: "phone", Value: "+5511999999999"}}

	h.GetReferralByPhone(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCTWAHandler_GetReferralsByCampaign_NoClient(t *testing.T) {
	h := NewCTWAHandler()
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/ctwa/campaigns/camp-1/referrals", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}, {Key: "campaignId", Value: "camp-1"}}

	h.GetReferralsByCampaign(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCTWAHandler_TrackConversion_NoClient(t *testing.T) {
	h := NewCTWAHandler()
	w, c := newTestContext(http.MethodPost, "/channels/channel-1/ctwa/conversions", map[string]interface{}{
		"referral_id":     "ref-1",
		"conversion_type": "purchase",
		"value":           100.0,
		"currency":        "BRL",
	})
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}}

	h.TrackConversion(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCTWAHandler_GetConversion_NoClient(t *testing.T) {
	h := NewCTWAHandler()
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/ctwa/conversions/conv-1", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}, {Key: "conversionId", Value: "conv-1"}}

	h.GetConversion(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCTWAHandler_GetConversionsByReferral_NoClient(t *testing.T) {
	h := NewCTWAHandler()
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/ctwa/referrals/ref-1/conversions", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}, {Key: "referralId", Value: "ref-1"}}

	h.GetConversionsByReferral(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCTWAHandler_GetFreeWindow_NoClient(t *testing.T) {
	h := NewCTWAHandler()
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/ctwa/free-window/+5511999999999", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}, {Key: "phone", Value: "+5511999999999"}}

	h.GetFreeWindow(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCTWAHandler_GetStats_NoClient(t *testing.T) {
	h := NewCTWAHandler()
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/ctwa/stats", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}}

	h.GetStats(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCTWAHandler_GetTopAds_NoClient(t *testing.T) {
	h := NewCTWAHandler()
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/ctwa/top-ads", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}}

	h.GetTopAds(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCTWAHandler_GenerateReport_NoClient(t *testing.T) {
	h := NewCTWAHandler()
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/ctwa/report", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}}

	h.GenerateReport(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCTWAHandler_GetDashboard_NoClient(t *testing.T) {
	h := NewCTWAHandler()
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/ctwa/dashboard", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}}

	h.GetDashboard(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCTWAHandler_ProcessReferralWebhook_NoClient(t *testing.T) {
	h := NewCTWAHandler()
	w, c := newTestContext(http.MethodPost, "/webhooks/ctwa/channel-1", map[string]string{"from": "+5511999999999"})
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}}

	h.ProcessReferralWebhook(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
