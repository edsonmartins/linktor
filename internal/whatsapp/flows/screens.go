package flows

import (
	"encoding/json"
	"fmt"
)

// Screen represents a WhatsApp Flow screen
type Screen struct {
	ID         string                 `json:"id"`
	Title      string                 `json:"title,omitempty"`
	Terminal   bool                   `json:"terminal,omitempty"`
	Success    bool                   `json:"success,omitempty"`
	RefreshOnBack bool                `json:"refresh-on-back,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Layout     *Layout                `json:"layout"`
}

// Layout represents the layout of a screen
type Layout struct {
	Type     string      `json:"type"` // SingleColumnLayout
	Children []Component `json:"children"`
}

// ToMap converts the screen to a map
func (s *Screen) ToMap() map[string]interface{} {
	children := make([]map[string]interface{}, len(s.Layout.Children))
	for i, child := range s.Layout.Children {
		children[i] = child.ToMap()
	}

	layout := map[string]interface{}{
		"type":     s.Layout.Type,
		"children": children,
	}

	result := map[string]interface{}{
		"id":     s.ID,
		"layout": layout,
	}

	if s.Title != "" {
		result["title"] = s.Title
	}
	if s.Terminal {
		result["terminal"] = true
	}
	if s.Success {
		result["success"] = true
	}
	if s.RefreshOnBack {
		result["refresh-on-back"] = true
	}
	if s.Data != nil {
		result["data"] = s.Data
	}

	return result
}

// ScreenBuilder helps build screens
type ScreenBuilder struct {
	screen *Screen
}

// NewScreenBuilder creates a new screen builder
func NewScreenBuilder(id string) *ScreenBuilder {
	return &ScreenBuilder{
		screen: &Screen{
			ID: id,
			Layout: &Layout{
				Type:     "SingleColumnLayout",
				Children: make([]Component, 0),
			},
		},
	}
}

// SetTitle sets the screen title
func (sb *ScreenBuilder) SetTitle(title string) *ScreenBuilder {
	sb.screen.Title = title
	return sb
}

// SetTerminal marks the screen as terminal (end of flow)
func (sb *ScreenBuilder) SetTerminal(terminal bool) *ScreenBuilder {
	sb.screen.Terminal = terminal
	return sb
}

// SetSuccess marks the screen as a success screen
func (sb *ScreenBuilder) SetSuccess(success bool) *ScreenBuilder {
	sb.screen.Success = success
	return sb
}

// SetRefreshOnBack enables refresh when navigating back
func (sb *ScreenBuilder) SetRefreshOnBack(refresh bool) *ScreenBuilder {
	sb.screen.RefreshOnBack = refresh
	return sb
}

// SetData sets the screen data
func (sb *ScreenBuilder) SetData(data map[string]interface{}) *ScreenBuilder {
	sb.screen.Data = data
	return sb
}

// AddComponent adds a component to the screen
func (sb *ScreenBuilder) AddComponent(component Component) *ScreenBuilder {
	sb.screen.Layout.Children = append(sb.screen.Layout.Children, component)
	return sb
}

// AddHeading adds a heading text
func (sb *ScreenBuilder) AddHeading(text string) *ScreenBuilder {
	return sb.AddComponent(&TextHeading{Text: text})
}

// AddSubheading adds a subheading text
func (sb *ScreenBuilder) AddSubheading(text string) *ScreenBuilder {
	return sb.AddComponent(&TextSubheading{Text: text})
}

// AddBody adds body text
func (sb *ScreenBuilder) AddBody(text string) *ScreenBuilder {
	return sb.AddComponent(&TextBody{Text: text})
}

// AddCaption adds caption text
func (sb *ScreenBuilder) AddCaption(text string) *ScreenBuilder {
	return sb.AddComponent(&TextCaption{Text: text})
}

// AddTextInput adds a text input
func (sb *ScreenBuilder) AddTextInput(name, label string, required bool) *ScreenBuilder {
	return sb.AddComponent(&TextInput{
		Name:     name,
		Label:    label,
		Required: required,
	})
}

// AddEmailInput adds an email input
func (sb *ScreenBuilder) AddEmailInput(name, label string, required bool) *ScreenBuilder {
	return sb.AddComponent(&TextInput{
		Name:      name,
		Label:     label,
		InputType: "email",
		Required:  required,
	})
}

// AddPhoneInput adds a phone input
func (sb *ScreenBuilder) AddPhoneInput(name, label string, required bool) *ScreenBuilder {
	return sb.AddComponent(&TextInput{
		Name:      name,
		Label:     label,
		InputType: "phone",
		Required:  required,
	})
}

// AddNumberInput adds a number input
func (sb *ScreenBuilder) AddNumberInput(name, label string, required bool) *ScreenBuilder {
	return sb.AddComponent(&TextInput{
		Name:      name,
		Label:     label,
		InputType: "number",
		Required:  required,
	})
}

// AddPasswordInput adds a password input
func (sb *ScreenBuilder) AddPasswordInput(name, label string, required bool) *ScreenBuilder {
	return sb.AddComponent(&TextInput{
		Name:      name,
		Label:     label,
		InputType: "password",
		Required:  required,
	})
}

// AddTextArea adds a text area
func (sb *ScreenBuilder) AddTextArea(name, label string, required bool) *ScreenBuilder {
	return sb.AddComponent(&TextArea{
		Name:     name,
		Label:    label,
		Required: required,
	})
}

// AddDatePicker adds a date picker
func (sb *ScreenBuilder) AddDatePicker(name, label string, required bool) *ScreenBuilder {
	return sb.AddComponent(&DatePicker{
		Name:     name,
		Label:    label,
		Required: required,
	})
}

// AddDropdown adds a dropdown
func (sb *ScreenBuilder) AddDropdown(name, label string, options []DropdownOption, required bool) *ScreenBuilder {
	return sb.AddComponent(&Dropdown{
		Name:       name,
		Label:      label,
		DataSource: options,
		Required:   required,
	})
}

// AddRadioButtons adds a radio button group
func (sb *ScreenBuilder) AddRadioButtons(name, label string, options []DropdownOption, required bool) *ScreenBuilder {
	return sb.AddComponent(&RadioButtonsGroup{
		Name:       name,
		Label:      label,
		DataSource: options,
		Required:   required,
	})
}

// AddCheckboxGroup adds a checkbox group
func (sb *ScreenBuilder) AddCheckboxGroup(name, label string, options []DropdownOption, required bool) *ScreenBuilder {
	return sb.AddComponent(&CheckboxGroup{
		Name:       name,
		Label:      label,
		DataSource: options,
		Required:   required,
	})
}

// AddOptIn adds an opt-in checkbox
func (sb *ScreenBuilder) AddOptIn(name, label string, required bool) *ScreenBuilder {
	return sb.AddComponent(&OptIn{
		Name:     name,
		Label:    label,
		Required: required,
	})
}

// AddImage adds an image
func (sb *ScreenBuilder) AddImage(src string, width, height int) *ScreenBuilder {
	return sb.AddComponent(&Image{
		Src:    src,
		Width:  width,
		Height: height,
	})
}

// AddForm adds a form with children
func (sb *ScreenBuilder) AddForm(name string, children []Component) *ScreenBuilder {
	return sb.AddComponent(&Form{
		Name:     name,
		Children: children,
	})
}

// AddFooter adds a footer with action
func (sb *ScreenBuilder) AddFooter(label string, action *OnClickAction) *ScreenBuilder {
	return sb.AddComponent(&Footer{
		Label:         label,
		OnClickAction: action,
	})
}

// AddNavigateFooter adds a footer that navigates to another screen
func (sb *ScreenBuilder) AddNavigateFooter(label, nextScreen string, payload map[string]interface{}) *ScreenBuilder {
	return sb.AddFooter(label, NavigateAction(nextScreen, payload))
}

// AddCompleteFooter adds a footer that completes the flow
func (sb *ScreenBuilder) AddCompleteFooter(label string, payload map[string]interface{}) *ScreenBuilder {
	return sb.AddFooter(label, CompleteAction(payload))
}

// AddDataExchangeFooter adds a footer that triggers data exchange
func (sb *ScreenBuilder) AddDataExchangeFooter(label string, payload map[string]interface{}) *ScreenBuilder {
	return sb.AddFooter(label, DataExchangeAction(payload))
}

// Build returns the built screen
func (sb *ScreenBuilder) Build() *Screen {
	return sb.screen
}

// ToJSON converts the screen to JSON
func (sb *ScreenBuilder) ToJSON() (string, error) {
	data, err := json.MarshalIndent(sb.screen.ToMap(), "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// =============================================================================
// Flow Builder
// =============================================================================

// Flow represents a complete WhatsApp Flow
type Flow struct {
	Version       string             `json:"version"`
	DataAPIVersion string            `json:"data_api_version,omitempty"`
	RoutingModel  map[string]interface{} `json:"routing_model,omitempty"`
	Screens       []*Screen          `json:"screens"`
}

// FlowBuilder helps build complete flows
type FlowBuilder struct {
	flow *Flow
}

// NewFlowBuilder creates a new flow builder
func NewFlowBuilder() *FlowBuilder {
	return &FlowBuilder{
		flow: &Flow{
			Version:       "3.1",
			DataAPIVersion: "3.0",
			Screens:       make([]*Screen, 0),
		},
	}
}

// SetVersion sets the flow version
func (fb *FlowBuilder) SetVersion(version string) *FlowBuilder {
	fb.flow.Version = version
	return fb
}

// SetDataAPIVersion sets the data API version
func (fb *FlowBuilder) SetDataAPIVersion(version string) *FlowBuilder {
	fb.flow.DataAPIVersion = version
	return fb
}

// SetRoutingModel sets the routing model
func (fb *FlowBuilder) SetRoutingModel(model map[string]interface{}) *FlowBuilder {
	fb.flow.RoutingModel = model
	return fb
}

// AddScreen adds a screen to the flow
func (fb *FlowBuilder) AddScreen(screen *Screen) *FlowBuilder {
	fb.flow.Screens = append(fb.flow.Screens, screen)
	return fb
}

// AddScreenBuilder adds a screen from a builder
func (fb *FlowBuilder) AddScreenBuilder(sb *ScreenBuilder) *FlowBuilder {
	return fb.AddScreen(sb.Build())
}

// Build returns the built flow
func (fb *FlowBuilder) Build() *Flow {
	return fb.flow
}

// ToMap converts the flow to a map
func (fb *FlowBuilder) ToMap() map[string]interface{} {
	screens := make([]map[string]interface{}, len(fb.flow.Screens))
	for i, screen := range fb.flow.Screens {
		screens[i] = screen.ToMap()
	}

	result := map[string]interface{}{
		"version": fb.flow.Version,
		"screens": screens,
	}

	if fb.flow.DataAPIVersion != "" {
		result["data_api_version"] = fb.flow.DataAPIVersion
	}
	if fb.flow.RoutingModel != nil {
		result["routing_model"] = fb.flow.RoutingModel
	}

	return result
}

// ToJSON converts the flow to JSON
func (fb *FlowBuilder) ToJSON() (string, error) {
	data, err := json.MarshalIndent(fb.ToMap(), "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Validate validates the flow structure
func (fb *FlowBuilder) Validate() error {
	if len(fb.flow.Screens) == 0 {
		return fmt.Errorf("flow must have at least one screen")
	}

	// Check for duplicate screen IDs
	ids := make(map[string]bool)
	for _, screen := range fb.flow.Screens {
		if ids[screen.ID] {
			return fmt.Errorf("duplicate screen ID: %s", screen.ID)
		}
		ids[screen.ID] = true
	}

	// Check that at least one screen is terminal or success
	hasTerminal := false
	for _, screen := range fb.flow.Screens {
		if screen.Terminal || screen.Success {
			hasTerminal = true
			break
		}
	}
	if !hasTerminal {
		return fmt.Errorf("flow must have at least one terminal or success screen")
	}

	return nil
}

// =============================================================================
// Common Flow Templates
// =============================================================================

// CreateSimpleFormFlow creates a simple form flow with one input screen
func CreateSimpleFormFlow(title string, fields []FormField) *FlowBuilder {
	screenBuilder := NewScreenBuilder("FORM_SCREEN").
		SetTitle(title)

	// Add form with fields
	formChildren := make([]Component, 0)
	for _, field := range fields {
		switch field.Type {
		case "text":
			formChildren = append(formChildren, &TextInput{
				Name:     field.Name,
				Label:    field.Label,
				Required: field.Required,
			})
		case "email":
			formChildren = append(formChildren, &TextInput{
				Name:      field.Name,
				Label:     field.Label,
				InputType: "email",
				Required:  field.Required,
			})
		case "phone":
			formChildren = append(formChildren, &TextInput{
				Name:      field.Name,
				Label:     field.Label,
				InputType: "phone",
				Required:  field.Required,
			})
		case "textarea":
			formChildren = append(formChildren, &TextArea{
				Name:     field.Name,
				Label:    field.Label,
				Required: field.Required,
			})
		case "date":
			formChildren = append(formChildren, &DatePicker{
				Name:     field.Name,
				Label:    field.Label,
				Required: field.Required,
			})
		}
	}

	screenBuilder.AddForm("form", formChildren)
	screenBuilder.AddCompleteFooter("Submit", map[string]interface{}{
		"screen": "FORM_SCREEN",
	})

	successScreen := NewScreenBuilder("SUCCESS").
		SetTitle("Success").
		SetSuccess(true).
		SetTerminal(true).
		AddHeading("Thank you!").
		AddBody("Your submission has been received.").
		Build()

	return NewFlowBuilder().
		AddScreenBuilder(screenBuilder).
		AddScreen(successScreen)
}

// FormField represents a field in a form
type FormField struct {
	Name     string
	Label    string
	Type     string // text, email, phone, textarea, date
	Required bool
}

// CreateAppointmentBookingFlow creates an appointment booking flow
func CreateAppointmentBookingFlow(services []DropdownOption, timeSlots []DropdownOption) *FlowBuilder {
	// Service selection screen
	serviceScreen := NewScreenBuilder("SERVICE").
		SetTitle("Select Service").
		AddHeading("Book an Appointment").
		AddBody("Please select the service you would like to book.").
		AddDropdown("service", "Service", services, true).
		AddNavigateFooter("Continue", "DATETIME", map[string]interface{}{}).
		Build()

	// Date and time selection screen
	dateTimeScreen := NewScreenBuilder("DATETIME").
		SetTitle("Select Date & Time").
		AddBody("Choose your preferred date and time.").
		AddDatePicker("date", "Date", true).
		AddDropdown("time", "Time Slot", timeSlots, true).
		AddNavigateFooter("Continue", "CONFIRM", map[string]interface{}{}).
		Build()

	// Confirmation screen
	confirmScreen := NewScreenBuilder("CONFIRM").
		SetTitle("Confirm Booking").
		AddHeading("Confirm Your Appointment").
		AddBody("Please review your booking details.").
		AddTextInput("name", "Your Name", true).
		AddPhoneInput("phone", "Phone Number", true).
		AddOptIn("reminder", "Send me a reminder", false).
		AddCompleteFooter("Book Appointment", map[string]interface{}{}).
		Build()

	// Success screen
	successScreen := NewScreenBuilder("SUCCESS").
		SetTitle("Booking Confirmed").
		SetSuccess(true).
		SetTerminal(true).
		AddHeading("Appointment Booked!").
		AddBody("Your appointment has been confirmed. We'll send you a reminder.").
		Build()

	return NewFlowBuilder().
		AddScreen(serviceScreen).
		AddScreen(dateTimeScreen).
		AddScreen(confirmScreen).
		AddScreen(successScreen)
}

// CreateFeedbackFlow creates a feedback collection flow
func CreateFeedbackFlow(categories []DropdownOption) *FlowBuilder {
	// Category selection
	categoryScreen := NewScreenBuilder("CATEGORY").
		SetTitle("Feedback").
		AddHeading("Share Your Feedback").
		AddBody("We'd love to hear from you!").
		AddDropdown("category", "Category", categories, true).
		AddNavigateFooter("Continue", "DETAILS", map[string]interface{}{}).
		Build()

	// Feedback details
	detailsScreen := NewScreenBuilder("DETAILS").
		SetTitle("Feedback Details").
		AddTextArea("feedback", "Your Feedback", true).
		AddRadioButtons("rating", "How would you rate your experience?", []DropdownOption{
			{ID: "5", Title: "Excellent"},
			{ID: "4", Title: "Good"},
			{ID: "3", Title: "Average"},
			{ID: "2", Title: "Poor"},
			{ID: "1", Title: "Very Poor"},
		}, true).
		AddCompleteFooter("Submit Feedback", map[string]interface{}{}).
		Build()

	// Thank you screen
	thankYouScreen := NewScreenBuilder("THANK_YOU").
		SetTitle("Thank You").
		SetSuccess(true).
		SetTerminal(true).
		AddHeading("Thank You!").
		AddBody("Your feedback has been submitted successfully.").
		Build()

	return NewFlowBuilder().
		AddScreen(categoryScreen).
		AddScreen(detailsScreen).
		AddScreen(thankYouScreen)
}
