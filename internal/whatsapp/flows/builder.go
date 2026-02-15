package flows

import (
	"encoding/json"
	"fmt"
)

// FlowJSON represents the complete JSON structure of a WhatsApp Flow
type FlowJSON struct {
	Version        string                 `json:"version"`
	DataAPIVersion string                 `json:"data_api_version,omitempty"`
	RoutingModel   map[string]interface{} `json:"routing_model,omitempty"`
	Screens        []map[string]interface{} `json:"screens"`
}

// FlowJSONBuilder provides a fluent API for building flow JSON
type FlowJSONBuilder struct {
	json     *FlowJSON
	screens  []*ScreenBuilder
	errors   []error
}

// NewFlowJSONBuilder creates a new flow JSON builder
func NewFlowJSONBuilder() *FlowJSONBuilder {
	return &FlowJSONBuilder{
		json: &FlowJSON{
			Version:        "3.1",
			DataAPIVersion: "3.0",
			Screens:        make([]map[string]interface{}, 0),
		},
		screens: make([]*ScreenBuilder, 0),
		errors:  make([]error, 0),
	}
}

// Version sets the flow version
func (b *FlowJSONBuilder) Version(version string) *FlowJSONBuilder {
	b.json.Version = version
	return b
}

// DataAPIVersion sets the data API version
func (b *FlowJSONBuilder) DataAPIVersion(version string) *FlowJSONBuilder {
	b.json.DataAPIVersion = version
	return b
}

// RoutingModel sets the routing model for dynamic navigation
func (b *FlowJSONBuilder) RoutingModel(model map[string]interface{}) *FlowJSONBuilder {
	b.json.RoutingModel = model
	return b
}

// Screen adds a new screen and returns a screen builder
func (b *FlowJSONBuilder) Screen(id string) *ScreenBuilder {
	sb := NewScreenBuilder(id)
	b.screens = append(b.screens, sb)
	return sb
}

// AddScreen adds an existing screen
func (b *FlowJSONBuilder) AddScreen(screen *Screen) *FlowJSONBuilder {
	b.json.Screens = append(b.json.Screens, screen.ToMap())
	return b
}

// Build builds the final flow JSON
func (b *FlowJSONBuilder) Build() (*FlowJSON, error) {
	// Add screens from builders
	for _, sb := range b.screens {
		b.json.Screens = append(b.json.Screens, sb.Build().ToMap())
	}

	// Validate
	if err := b.Validate(); err != nil {
		return nil, err
	}

	return b.json, nil
}

// BuildJSON builds and returns the JSON string
func (b *FlowJSONBuilder) BuildJSON() (string, error) {
	flow, err := b.Build()
	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(flow, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal flow JSON: %w", err)
	}

	return string(data), nil
}

// BuildJSONCompact builds and returns compact JSON
func (b *FlowJSONBuilder) BuildJSONCompact() (string, error) {
	flow, err := b.Build()
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(flow)
	if err != nil {
		return "", fmt.Errorf("failed to marshal flow JSON: %w", err)
	}

	return string(data), nil
}

// Validate validates the flow structure
func (b *FlowJSONBuilder) Validate() error {
	if len(b.json.Screens) == 0 && len(b.screens) == 0 {
		return fmt.Errorf("flow must have at least one screen")
	}

	// Collect all screen IDs
	screenIDs := make(map[string]bool)

	// From direct screens
	for _, screen := range b.json.Screens {
		id, ok := screen["id"].(string)
		if !ok {
			return fmt.Errorf("screen missing id")
		}
		if screenIDs[id] {
			return fmt.Errorf("duplicate screen ID: %s", id)
		}
		screenIDs[id] = true
	}

	// From screen builders
	for _, sb := range b.screens {
		screen := sb.Build()
		if screenIDs[screen.ID] {
			return fmt.Errorf("duplicate screen ID: %s", screen.ID)
		}
		screenIDs[screen.ID] = true
	}

	return nil
}

// =============================================================================
// Predefined Flow Templates
// =============================================================================

// ContactFormTemplate creates a contact form flow
func ContactFormTemplate() *FlowJSONBuilder {
	builder := NewFlowJSONBuilder()

	// Contact form screen
	builder.Screen("CONTACT_FORM").
		SetTitle("Contact Us").
		AddHeading("Get in Touch").
		AddBody("Fill out the form below and we'll get back to you.").
		AddForm("contact_form", []Component{
			&TextInput{Name: "name", Label: "Full Name", Required: true},
			&TextInput{Name: "email", Label: "Email", InputType: "email", Required: true},
			&TextInput{Name: "phone", Label: "Phone", InputType: "phone", Required: false},
			&Dropdown{
				Name:  "subject",
				Label: "Subject",
				DataSource: []DropdownOption{
					{ID: "general", Title: "General Inquiry"},
					{ID: "support", Title: "Technical Support"},
					{ID: "sales", Title: "Sales Question"},
					{ID: "feedback", Title: "Feedback"},
				},
				Required: true,
			},
			&TextArea{Name: "message", Label: "Message", Required: true, MaxLength: 500},
		}).
		AddCompleteFooter("Send Message", map[string]interface{}{
			"screen": "CONTACT_FORM",
		})

	// Success screen
	builder.Screen("SUCCESS").
		SetTitle("Message Sent").
		SetSuccess(true).
		SetTerminal(true).
		AddHeading("Thank You!").
		AddBody("We've received your message and will respond within 24 hours.")

	return builder
}

// SurveyTemplate creates a survey flow
func SurveyTemplate(questions []SurveyQuestion) *FlowJSONBuilder {
	builder := NewFlowJSONBuilder()

	// Introduction screen
	builder.Screen("INTRO").
		SetTitle("Survey").
		AddHeading("Quick Survey").
		AddBody("Help us improve by answering a few questions. It only takes a minute!").
		AddNavigateFooter("Start Survey", "Q1", nil)

	// Add question screens
	for i, q := range questions {
		screenID := fmt.Sprintf("Q%d", i+1)
		nextScreen := "THANKS"
		if i < len(questions)-1 {
			nextScreen = fmt.Sprintf("Q%d", i+2)
		}

		sb := builder.Screen(screenID).
			SetTitle(fmt.Sprintf("Question %d of %d", i+1, len(questions))).
			AddBody(q.Question)

		switch q.Type {
		case "radio":
			sb.AddRadioButtons(q.Name, "", q.Options, q.Required)
		case "checkbox":
			sb.AddCheckboxGroup(q.Name, "", q.Options, q.Required)
		case "dropdown":
			sb.AddDropdown(q.Name, "", q.Options, q.Required)
		case "text":
			sb.AddTextArea(q.Name, "Your answer", q.Required)
		case "rating":
			sb.AddRadioButtons(q.Name, "", []DropdownOption{
				{ID: "5", Title: "⭐⭐⭐⭐⭐ Excellent"},
				{ID: "4", Title: "⭐⭐⭐⭐ Good"},
				{ID: "3", Title: "⭐⭐⭐ Average"},
				{ID: "2", Title: "⭐⭐ Poor"},
				{ID: "1", Title: "⭐ Very Poor"},
			}, q.Required)
		}

		if i == len(questions)-1 {
			sb.AddCompleteFooter("Submit", map[string]interface{}{
				"screen": screenID,
			})
		} else {
			sb.AddNavigateFooter("Next", nextScreen, nil)
		}
	}

	// Thank you screen
	builder.Screen("THANKS").
		SetTitle("Thank You").
		SetSuccess(true).
		SetTerminal(true).
		AddHeading("Thank You!").
		AddBody("Your responses have been recorded.")

	return builder
}

// SurveyQuestion represents a survey question
type SurveyQuestion struct {
	Name     string
	Question string
	Type     string           // radio, checkbox, dropdown, text, rating
	Options  []DropdownOption // For radio, checkbox, dropdown
	Required bool
}

// LeadCaptureTemplate creates a lead capture flow
func LeadCaptureTemplate(offerTitle, offerDescription string) *FlowJSONBuilder {
	builder := NewFlowJSONBuilder()

	// Offer screen
	builder.Screen("OFFER").
		SetTitle(offerTitle).
		AddHeading(offerTitle).
		AddBody(offerDescription).
		AddNavigateFooter("Get Started", "FORM", nil)

	// Form screen
	builder.Screen("FORM").
		SetTitle("Your Information").
		AddForm("lead_form", []Component{
			&TextInput{Name: "first_name", Label: "First Name", Required: true},
			&TextInput{Name: "last_name", Label: "Last Name", Required: true},
			&TextInput{Name: "email", Label: "Email", InputType: "email", Required: true},
			&TextInput{Name: "phone", Label: "Phone", InputType: "phone", Required: true},
			&TextInput{Name: "company", Label: "Company", Required: false},
			&OptIn{Name: "consent", Label: "I agree to receive marketing communications", Required: true},
		}).
		AddCompleteFooter("Submit", map[string]interface{}{
			"screen": "FORM",
		})

	// Confirmation screen
	builder.Screen("CONFIRMATION").
		SetTitle("Thank You").
		SetSuccess(true).
		SetTerminal(true).
		AddHeading("You're All Set!").
		AddBody("We'll be in touch shortly with your exclusive offer.")

	return builder
}

// OrderTrackingTemplate creates an order tracking flow
func OrderTrackingTemplate() *FlowJSONBuilder {
	builder := NewFlowJSONBuilder()

	// Order input screen
	builder.Screen("ORDER_INPUT").
		SetTitle("Track Order").
		AddHeading("Track Your Order").
		AddBody("Enter your order number to check the status.").
		AddTextInput("order_number", "Order Number", true).
		AddDataExchangeFooter("Track Order", map[string]interface{}{
			"screen": "ORDER_INPUT",
		})

	// Order status screen (populated by data exchange)
	builder.Screen("ORDER_STATUS").
		SetTitle("Order Status").
		AddHeading("Order Details").
		// This would be populated dynamically
		AddBody("{{order_status}}").
		SetTerminal(true)

	// Not found screen
	builder.Screen("NOT_FOUND").
		SetTitle("Order Not Found").
		AddHeading("Order Not Found").
		AddBody("We couldn't find an order with that number. Please check and try again.").
		AddNavigateFooter("Try Again", "ORDER_INPUT", nil)

	return builder
}

// ProductCatalogTemplate creates a product browsing flow
func ProductCatalogTemplate(categories []DropdownOption) *FlowJSONBuilder {
	builder := NewFlowJSONBuilder()

	// Category selection
	builder.Screen("CATEGORIES").
		SetTitle("Shop").
		AddHeading("Browse Products").
		AddBody("Select a category to explore.").
		AddDropdown("category", "Category", categories, true).
		AddDataExchangeFooter("View Products", map[string]interface{}{
			"screen": "CATEGORIES",
		})

	// Product list (populated dynamically)
	builder.Screen("PRODUCTS").
		SetTitle("Products").
		// Products would be populated by data exchange
		AddBody("Select a product to view details.").
		AddNavigateFooter("Back to Categories", "CATEGORIES", nil)

	// Product details
	builder.Screen("PRODUCT_DETAILS").
		SetTitle("Product Details").
		// Details would be populated by data exchange
		AddBody("{{product_description}}").
		AddCompleteFooter("Add to Cart", map[string]interface{}{
			"screen": "PRODUCT_DETAILS",
		})

	// Cart confirmation
	builder.Screen("CART_CONFIRMATION").
		SetTitle("Added to Cart").
		SetSuccess(true).
		SetTerminal(true).
		AddHeading("Added to Cart!").
		AddBody("The item has been added to your cart.")

	return builder
}

// =============================================================================
// Flow JSON Utilities
// =============================================================================

// ParseFlowJSON parses a flow JSON string
func ParseFlowJSON(jsonStr string) (*FlowJSON, error) {
	var flow FlowJSON
	if err := json.Unmarshal([]byte(jsonStr), &flow); err != nil {
		return nil, fmt.Errorf("failed to parse flow JSON: %w", err)
	}
	return &flow, nil
}

// ValidateFlowJSON validates a flow JSON string
func ValidateFlowJSON(jsonStr string) error {
	flow, err := ParseFlowJSON(jsonStr)
	if err != nil {
		return err
	}

	if flow.Version == "" {
		return fmt.Errorf("flow version is required")
	}

	if len(flow.Screens) == 0 {
		return fmt.Errorf("flow must have at least one screen")
	}

	// Check for duplicate screen IDs
	ids := make(map[string]bool)
	for _, screen := range flow.Screens {
		id, ok := screen["id"].(string)
		if !ok || id == "" {
			return fmt.Errorf("all screens must have an id")
		}
		if ids[id] {
			return fmt.Errorf("duplicate screen ID: %s", id)
		}
		ids[id] = true
	}

	return nil
}

// MinifyFlowJSON removes whitespace from flow JSON
func MinifyFlowJSON(jsonStr string) (string, error) {
	flow, err := ParseFlowJSON(jsonStr)
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(flow)
	if err != nil {
		return "", fmt.Errorf("failed to marshal flow: %w", err)
	}

	return string(data), nil
}

// PrettifyFlowJSON formats flow JSON with indentation
func PrettifyFlowJSON(jsonStr string) (string, error) {
	flow, err := ParseFlowJSON(jsonStr)
	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(flow, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal flow: %w", err)
	}

	return string(data), nil
}
