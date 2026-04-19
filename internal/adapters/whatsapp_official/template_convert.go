package whatsapp_official

import (
	"fmt"
	"sort"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// SendValues carries the runtime data needed to turn a stored entity
// template into an outgoing template message. The structure mirrors the
// shape Meta expects at send time and is deliberately narrower than the
// free-form builder API — callers who need arbitrary components should
// reach for TemplateBuilder directly.
type SendValues struct {
	// BodyParams are positional values for the template's body placeholders.
	// For POSITIONAL templates the slice order matches {{1}}, {{2}}, …;
	// for NAMED templates the keys match the {{named}} identifiers.
	BodyParams []string
	NamedBody  map[string]string

	// Header media. Only one of these should be set, matching the
	// template's declared header format.
	HeaderText             string
	HeaderImageID          string
	HeaderImageURL         string
	HeaderVideoID          string
	HeaderVideoURL         string
	HeaderDocumentID       string
	HeaderDocumentURL      string
	HeaderDocumentFilename string
	HeaderLocation         *LocationObject

	// ButtonValues holds the runtime payload for dynamic buttons keyed by
	// button index (the order they appear in the template definition).
	// For URL buttons the value is the URL suffix; for COPY_CODE it's the
	// coupon; for OTP it's the code; for QUICK_REPLY it's the callback
	// payload; for FLOW it's the flow_token.
	ButtonValues map[int]string

	// FlowExtras lets a caller attach a flow_action_data blob to a flow
	// button at the given index. Ignored for any other button type.
	FlowExtras map[int]map[string]interface{}

	// CardValues carries per-card runtime values for a CAROUSEL component.
	// Keyed by card_index (matching the position in the approved template),
	// each entry is a scoped SendValues whose own BodyParams / HeaderImage*
	// / ButtonValues feed the card's sub-components. Cards without a
	// matching entry are sent with their static definition only.
	CardValues map[int]CardSendValues
}

// CardSendValues carries the runtime substitutions for a single carousel
// card. Mirrors the top-level SendValues but scoped to one card — each
// card has its own header (image/video), body params, and button values.
type CardSendValues struct {
	BodyParams             []string
	NamedBody              map[string]string
	HeaderImageID          string
	HeaderImageURL         string
	HeaderVideoID          string
	HeaderVideoURL         string
	HeaderDocumentID       string
	HeaderDocumentURL      string
	HeaderDocumentFilename string
	ButtonValues           map[int]string
	FlowExtras             map[int]map[string]interface{}
}

// BuildSendPayload converts a stored entity.Template plus its runtime
// values into an adapter TemplateObject ready to feed SendMessage. This is
// the canonical bridge between the "what the template looks like" world
// (entity) and the "what we send per message" world (adapter).
//
// The function does not validate the structure — it assumes the entity
// template has already passed TemplateService's validation. It focuses on
// shape translation: entity components (HEADER/BODY/BUTTONS/…) become the
// lowercase send components with sub_type + index that Meta expects on
// the /messages endpoint.
func BuildSendPayload(t *entity.Template, values SendValues) (*TemplateObject, error) {
	if t == nil {
		return nil, fmt.Errorf("template is required")
	}

	obj := &TemplateObject{
		Name: t.Name,
		Language: &TemplateLanguage{
			Policy: "deterministic",
			Code:   t.Language,
		},
		Components: []TemplateComponent{},
	}

	for _, comp := range t.Components {
		switch comp.Type {
		case "HEADER":
			h, err := buildHeaderSendComponent(comp, values)
			if err != nil {
				return nil, err
			}
			if h != nil {
				obj.Components = append(obj.Components, *h)
			}

		case "BODY":
			if params := buildBodyParams(comp, values); len(params) > 0 {
				obj.Components = append(obj.Components, TemplateComponent{
					Type:       "body",
					Parameters: params,
				})
			}

		case "BUTTONS":
			obj.Components = append(obj.Components, buildButtonSendComponents(comp, values)...)

		case "CAROUSEL":
			if card := buildCarouselSendComponent(comp, values); card != nil {
				obj.Components = append(obj.Components, *card)
			}

		case "FOOTER", "LIMITED_TIME_OFFER":
			// Static content — Meta uses the approved template definition and
			// does not expect runtime parameters for these on /messages.

		default:
			// Unknown component types are skipped silently so newer template
			// shapes don't break send; validation already ran at create time.
		}
	}

	return obj, nil
}

// buildCarouselSendComponent emits the carousel wrapper that Meta expects:
//
//	{"type":"carousel","cards":[{"card_index":0,"components":[...]}]}
//
// Each card's runtime values come from `values.CardValues[cardIndex]`.
// Cards without runtime values are skipped entirely — Meta then uses the
// approved card definition with no substitutions.
func buildCarouselSendComponent(comp entity.TemplateComponent, values SendValues) *TemplateComponent {
	if len(values.CardValues) == 0 {
		return nil
	}

	cards := make([]TemplateCard, 0, len(comp.Cards))
	for cardIndex := range comp.Cards {
		cv, ok := values.CardValues[cardIndex]
		if !ok {
			continue
		}
		// Re-use the top-level helpers by projecting CardSendValues into a
		// SendValues. Each helper already handles "missing value → skip",
		// so empty fields on the card become absent components.
		scoped := SendValues{
			BodyParams:             cv.BodyParams,
			NamedBody:              cv.NamedBody,
			HeaderImageID:          cv.HeaderImageID,
			HeaderImageURL:         cv.HeaderImageURL,
			HeaderVideoID:          cv.HeaderVideoID,
			HeaderVideoURL:         cv.HeaderVideoURL,
			HeaderDocumentID:       cv.HeaderDocumentID,
			HeaderDocumentURL:      cv.HeaderDocumentURL,
			HeaderDocumentFilename: cv.HeaderDocumentFilename,
			ButtonValues:           cv.ButtonValues,
			FlowExtras:             cv.FlowExtras,
		}

		var cardComponents []TemplateComponent
		for _, sub := range comp.Cards[cardIndex].Components {
			switch sub.Type {
			case "HEADER":
				if h, err := buildHeaderSendComponent(sub, scoped); err == nil && h != nil {
					cardComponents = append(cardComponents, *h)
				}
			case "BODY":
				if p := buildBodyParams(sub, scoped); len(p) > 0 {
					cardComponents = append(cardComponents, TemplateComponent{Type: "body", Parameters: p})
				}
			case "BUTTONS":
				cardComponents = append(cardComponents, buildButtonSendComponents(sub, scoped)...)
			}
		}

		cards = append(cards, TemplateCard{
			CardIndex:  cardIndex,
			Components: cardComponents,
		})
	}

	if len(cards) == 0 {
		return nil
	}
	return &TemplateComponent{Type: "carousel", Cards: cards}
}

func buildHeaderSendComponent(comp entity.TemplateComponent, values SendValues) (*TemplateComponent, error) {
	format := comp.Format
	if format == "" {
		format = "TEXT"
	}
	var param *TemplateParameter

	switch format {
	case "TEXT":
		if values.HeaderText == "" {
			return nil, nil // nothing to substitute
		}
		param = &TemplateParameter{Type: "text", Text: values.HeaderText}

	case "IMAGE":
		img := &MediaObject{}
		switch {
		case values.HeaderImageID != "":
			img.ID = values.HeaderImageID
		case values.HeaderImageURL != "":
			img.Link = values.HeaderImageURL
		default:
			return nil, nil
		}
		param = &TemplateParameter{Type: "image", Image: img}

	case "VIDEO":
		vid := &MediaObject{}
		switch {
		case values.HeaderVideoID != "":
			vid.ID = values.HeaderVideoID
		case values.HeaderVideoURL != "":
			vid.Link = values.HeaderVideoURL
		default:
			return nil, nil
		}
		param = &TemplateParameter{Type: "video", Video: vid}

	case "DOCUMENT":
		doc := &DocumentObject{Filename: values.HeaderDocumentFilename}
		switch {
		case values.HeaderDocumentID != "":
			doc.ID = values.HeaderDocumentID
		case values.HeaderDocumentURL != "":
			doc.Link = values.HeaderDocumentURL
		default:
			return nil, nil
		}
		param = &TemplateParameter{Type: "document", Document: doc}

	case "LOCATION":
		if values.HeaderLocation == nil {
			return nil, nil
		}
		param = &TemplateParameter{Type: "location", Location: values.HeaderLocation}
	}

	if param == nil {
		return nil, nil
	}
	return &TemplateComponent{Type: "header", Parameters: []TemplateParameter{*param}}, nil
}

func buildBodyParams(comp entity.TemplateComponent, values SendValues) []TemplateParameter {
	// Named params: Meta needs the parameter_name on each parameter to
	// match it against the template's {{name}} placeholders. Iterating
	// over the map is order-unstable in Go, so we sort by key to keep the
	// outbound JSON deterministic (useful for cache headers, test
	// assertions, and reproducible request logs).
	if len(values.NamedBody) > 0 {
		keys := make([]string, 0, len(values.NamedBody))
		for k := range values.NamedBody {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		params := make([]TemplateParameter, 0, len(keys))
		for _, k := range keys {
			params = append(params, TemplateParameter{
				Type:          "text",
				ParameterName: k,
				Text:          values.NamedBody[k],
			})
		}
		return params
	}
	params := make([]TemplateParameter, len(values.BodyParams))
	for i, v := range values.BodyParams {
		params[i] = TemplateParameter{Type: "text", Text: v}
	}
	return params
}

func buildButtonSendComponents(comp entity.TemplateComponent, values SendValues) []TemplateComponent {
	out := make([]TemplateComponent, 0, len(comp.Buttons))
	for i, btn := range comp.Buttons {
		idx := i
		switch btn.Type {
		case "QUICK_REPLY":
			payload, ok := values.ButtonValues[i]
			if !ok {
				continue
			}
			out = append(out, TemplateComponent{
				Type: "button", SubType: "quick_reply", Index: &idx,
				Parameters: []TemplateParameter{{Type: "payload", Text: payload}},
			})

		case "URL":
			suffix, ok := values.ButtonValues[i]
			if !ok {
				continue
			}
			out = append(out, TemplateComponent{
				Type: "button", SubType: "url", Index: &idx,
				Parameters: []TemplateParameter{{Type: "text", Text: suffix}},
			})

		case "COPY_CODE":
			coupon, ok := values.ButtonValues[i]
			if !ok {
				continue
			}
			out = append(out, TemplateComponent{
				Type: "button", SubType: "copy_code", Index: &idx,
				Parameters: []TemplateParameter{{Type: "coupon_code", Text: coupon}},
			})

		case "OTP":
			otp, ok := values.ButtonValues[i]
			if !ok {
				continue
			}
			out = append(out, TemplateComponent{
				Type: "button", SubType: "url", Index: &idx,
				Parameters: []TemplateParameter{{Type: "text", Text: otp}},
			})

		case "FLOW":
			token, ok := values.ButtonValues[i]
			if !ok {
				continue
			}
			action := map[string]interface{}{"flow_token": token}
			if extras, hasExtras := values.FlowExtras[i]; hasExtras {
				action["flow_action_data"] = extras
			}
			out = append(out, TemplateComponent{
				Type: "button", SubType: "flow", Index: &idx,
				Parameters: []TemplateParameter{{Type: "action", Action: action}},
			})

		case "PHONE_NUMBER":
			// Phone buttons have no runtime data — the number is baked in.
			// Emit the component so Meta's template structure matches.
			out = append(out, TemplateComponent{
				Type: "button", SubType: "phone_number", Index: &idx,
			})
		}
	}
	return out
}
