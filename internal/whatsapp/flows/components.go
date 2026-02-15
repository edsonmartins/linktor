package flows

// Component represents a WhatsApp Flow component
type Component interface {
	ComponentType() string
	ToMap() map[string]interface{}
}

// BaseComponent contains common component properties
type BaseComponent struct {
	Type     string `json:"type"`
	Name     string `json:"name,omitempty"`
	Visible  *bool  `json:"visible,omitempty"`
	OnClick  string `json:"on-click-action,omitempty"`
}

// =============================================================================
// Layout Components
// =============================================================================

// Form represents a Form layout component
type Form struct {
	Name     string      `json:"name"`
	Children []Component `json:"children"`
	InitValues map[string]interface{} `json:"init-values,omitempty"`
	ErrorMessages map[string]string `json:"error-messages,omitempty"`
}

func (f *Form) ComponentType() string { return "Form" }

func (f *Form) ToMap() map[string]interface{} {
	children := make([]map[string]interface{}, len(f.Children))
	for i, child := range f.Children {
		children[i] = child.ToMap()
	}

	result := map[string]interface{}{
		"type":     "Form",
		"name":     f.Name,
		"children": children,
	}

	if f.InitValues != nil {
		result["init-values"] = f.InitValues
	}
	if f.ErrorMessages != nil {
		result["error-messages"] = f.ErrorMessages
	}

	return result
}

// =============================================================================
// Text Components
// =============================================================================

// TextHeading represents a heading text component
type TextHeading struct {
	Text     string `json:"text"`
	Visible  *bool  `json:"visible,omitempty"`
}

func (t *TextHeading) ComponentType() string { return "TextHeading" }

func (t *TextHeading) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"type": "TextHeading",
		"text": t.Text,
	}
	if t.Visible != nil {
		result["visible"] = *t.Visible
	}
	return result
}

// TextSubheading represents a subheading text component
type TextSubheading struct {
	Text    string `json:"text"`
	Visible *bool  `json:"visible,omitempty"`
}

func (t *TextSubheading) ComponentType() string { return "TextSubheading" }

func (t *TextSubheading) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"type": "TextSubheading",
		"text": t.Text,
	}
	if t.Visible != nil {
		result["visible"] = *t.Visible
	}
	return result
}

// TextBody represents a body text component
type TextBody struct {
	Text    string `json:"text"`
	Visible *bool  `json:"visible,omitempty"`
}

func (t *TextBody) ComponentType() string { return "TextBody" }

func (t *TextBody) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"type": "TextBody",
		"text": t.Text,
	}
	if t.Visible != nil {
		result["visible"] = *t.Visible
	}
	return result
}

// TextCaption represents a caption text component
type TextCaption struct {
	Text    string `json:"text"`
	Visible *bool  `json:"visible,omitempty"`
}

func (t *TextCaption) ComponentType() string { return "TextCaption" }

func (t *TextCaption) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"type": "TextCaption",
		"text": t.Text,
	}
	if t.Visible != nil {
		result["visible"] = *t.Visible
	}
	return result
}

// =============================================================================
// Input Components
// =============================================================================

// TextInput represents a text input component
type TextInput struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	InputType   string `json:"input-type,omitempty"` // text, email, password, passcode, phone, number
	Required    bool   `json:"required,omitempty"`
	MinChars    int    `json:"min-chars,omitempty"`
	MaxChars    int    `json:"max-chars,omitempty"`
	HelperText  string `json:"helper-text,omitempty"`
	Enabled     *bool  `json:"enabled,omitempty"`
	Visible     *bool  `json:"visible,omitempty"`
	Pattern     string `json:"pattern,omitempty"`
}

func (t *TextInput) ComponentType() string { return "TextInput" }

func (t *TextInput) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"type":  "TextInput",
		"name":  t.Name,
		"label": t.Label,
	}
	if t.InputType != "" {
		result["input-type"] = t.InputType
	}
	if t.Required {
		result["required"] = true
	}
	if t.MinChars > 0 {
		result["min-chars"] = t.MinChars
	}
	if t.MaxChars > 0 {
		result["max-chars"] = t.MaxChars
	}
	if t.HelperText != "" {
		result["helper-text"] = t.HelperText
	}
	if t.Enabled != nil {
		result["enabled"] = *t.Enabled
	}
	if t.Visible != nil {
		result["visible"] = *t.Visible
	}
	if t.Pattern != "" {
		result["pattern"] = t.Pattern
	}
	return result
}

// TextArea represents a multi-line text input component
type TextArea struct {
	Name       string `json:"name"`
	Label      string `json:"label"`
	Required   bool   `json:"required,omitempty"`
	MaxLength  int    `json:"max-length,omitempty"`
	HelperText string `json:"helper-text,omitempty"`
	Enabled    *bool  `json:"enabled,omitempty"`
	Visible    *bool  `json:"visible,omitempty"`
}

func (t *TextArea) ComponentType() string { return "TextArea" }

func (t *TextArea) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"type":  "TextArea",
		"name":  t.Name,
		"label": t.Label,
	}
	if t.Required {
		result["required"] = true
	}
	if t.MaxLength > 0 {
		result["max-length"] = t.MaxLength
	}
	if t.HelperText != "" {
		result["helper-text"] = t.HelperText
	}
	if t.Enabled != nil {
		result["enabled"] = *t.Enabled
	}
	if t.Visible != nil {
		result["visible"] = *t.Visible
	}
	return result
}

// DatePicker represents a date picker component
type DatePicker struct {
	Name       string `json:"name"`
	Label      string `json:"label"`
	Required   bool   `json:"required,omitempty"`
	MinDate    string `json:"min-date,omitempty"` // YYYY-MM-DD or timestamp
	MaxDate    string `json:"max-date,omitempty"`
	HelperText string `json:"helper-text,omitempty"`
	Enabled    *bool  `json:"enabled,omitempty"`
	Visible    *bool  `json:"visible,omitempty"`
}

func (d *DatePicker) ComponentType() string { return "DatePicker" }

func (d *DatePicker) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"type":  "DatePicker",
		"name":  d.Name,
		"label": d.Label,
	}
	if d.Required {
		result["required"] = true
	}
	if d.MinDate != "" {
		result["min-date"] = d.MinDate
	}
	if d.MaxDate != "" {
		result["max-date"] = d.MaxDate
	}
	if d.HelperText != "" {
		result["helper-text"] = d.HelperText
	}
	if d.Enabled != nil {
		result["enabled"] = *d.Enabled
	}
	if d.Visible != nil {
		result["visible"] = *d.Visible
	}
	return result
}

// DropdownOption represents an option in a dropdown
type DropdownOption struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Enabled     *bool  `json:"enabled,omitempty"`
}

// Dropdown represents a dropdown selection component
type Dropdown struct {
	Name       string            `json:"name"`
	Label      string            `json:"label"`
	DataSource []DropdownOption  `json:"data-source"`
	Required   bool              `json:"required,omitempty"`
	HelperText string            `json:"helper-text,omitempty"`
	Enabled    *bool             `json:"enabled,omitempty"`
	Visible    *bool             `json:"visible,omitempty"`
}

func (d *Dropdown) ComponentType() string { return "Dropdown" }

func (d *Dropdown) ToMap() map[string]interface{} {
	dataSource := make([]map[string]interface{}, len(d.DataSource))
	for i, opt := range d.DataSource {
		item := map[string]interface{}{
			"id":    opt.ID,
			"title": opt.Title,
		}
		if opt.Description != "" {
			item["description"] = opt.Description
		}
		if opt.Enabled != nil {
			item["enabled"] = *opt.Enabled
		}
		dataSource[i] = item
	}

	result := map[string]interface{}{
		"type":        "Dropdown",
		"name":        d.Name,
		"label":       d.Label,
		"data-source": dataSource,
	}
	if d.Required {
		result["required"] = true
	}
	if d.HelperText != "" {
		result["helper-text"] = d.HelperText
	}
	if d.Enabled != nil {
		result["enabled"] = *d.Enabled
	}
	if d.Visible != nil {
		result["visible"] = *d.Visible
	}
	return result
}

// RadioButtonsGroup represents a radio button group component
type RadioButtonsGroup struct {
	Name       string           `json:"name"`
	Label      string           `json:"label"`
	DataSource []DropdownOption `json:"data-source"`
	Required   bool             `json:"required,omitempty"`
	HelperText string           `json:"helper-text,omitempty"`
	Enabled    *bool            `json:"enabled,omitempty"`
	Visible    *bool            `json:"visible,omitempty"`
}

func (r *RadioButtonsGroup) ComponentType() string { return "RadioButtonsGroup" }

func (r *RadioButtonsGroup) ToMap() map[string]interface{} {
	dataSource := make([]map[string]interface{}, len(r.DataSource))
	for i, opt := range r.DataSource {
		item := map[string]interface{}{
			"id":    opt.ID,
			"title": opt.Title,
		}
		if opt.Description != "" {
			item["description"] = opt.Description
		}
		dataSource[i] = item
	}

	result := map[string]interface{}{
		"type":        "RadioButtonsGroup",
		"name":        r.Name,
		"label":       r.Label,
		"data-source": dataSource,
	}
	if r.Required {
		result["required"] = true
	}
	if r.HelperText != "" {
		result["helper-text"] = r.HelperText
	}
	if r.Enabled != nil {
		result["enabled"] = *r.Enabled
	}
	if r.Visible != nil {
		result["visible"] = *r.Visible
	}
	return result
}

// CheckboxGroup represents a checkbox group component
type CheckboxGroup struct {
	Name       string           `json:"name"`
	Label      string           `json:"label"`
	DataSource []DropdownOption `json:"data-source"`
	Required   bool             `json:"required,omitempty"`
	MinSelected int             `json:"min-selected-items,omitempty"`
	MaxSelected int             `json:"max-selected-items,omitempty"`
	HelperText string           `json:"helper-text,omitempty"`
	Enabled    *bool            `json:"enabled,omitempty"`
	Visible    *bool            `json:"visible,omitempty"`
}

func (c *CheckboxGroup) ComponentType() string { return "CheckboxGroup" }

func (c *CheckboxGroup) ToMap() map[string]interface{} {
	dataSource := make([]map[string]interface{}, len(c.DataSource))
	for i, opt := range c.DataSource {
		item := map[string]interface{}{
			"id":    opt.ID,
			"title": opt.Title,
		}
		if opt.Description != "" {
			item["description"] = opt.Description
		}
		dataSource[i] = item
	}

	result := map[string]interface{}{
		"type":        "CheckboxGroup",
		"name":        c.Name,
		"label":       c.Label,
		"data-source": dataSource,
	}
	if c.Required {
		result["required"] = true
	}
	if c.MinSelected > 0 {
		result["min-selected-items"] = c.MinSelected
	}
	if c.MaxSelected > 0 {
		result["max-selected-items"] = c.MaxSelected
	}
	if c.HelperText != "" {
		result["helper-text"] = c.HelperText
	}
	if c.Enabled != nil {
		result["enabled"] = *c.Enabled
	}
	if c.Visible != nil {
		result["visible"] = *c.Visible
	}
	return result
}

// OptIn represents an opt-in checkbox component
type OptIn struct {
	Name       string `json:"name"`
	Label      string `json:"label"`
	Required   bool   `json:"required,omitempty"`
	OnClickURL string `json:"on-click-action,omitempty"`
	Visible    *bool  `json:"visible,omitempty"`
}

func (o *OptIn) ComponentType() string { return "OptIn" }

func (o *OptIn) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"type":  "OptIn",
		"name":  o.Name,
		"label": o.Label,
	}
	if o.Required {
		result["required"] = true
	}
	if o.OnClickURL != "" {
		result["on-click-action"] = map[string]interface{}{
			"name":    "navigate",
			"next":    map[string]string{"type": "screen", "name": "_url"},
			"payload": map[string]string{"url": o.OnClickURL},
		}
	}
	if o.Visible != nil {
		result["visible"] = *o.Visible
	}
	return result
}

// =============================================================================
// Media Components
// =============================================================================

// Image represents an image component
type Image struct {
	Src         string `json:"src"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
	ScaleType   string `json:"scale-type,omitempty"` // contain, cover
	AspectRatio string `json:"aspect-ratio,omitempty"`
	AltText     string `json:"alt-text,omitempty"`
	Visible     *bool  `json:"visible,omitempty"`
}

func (i *Image) ComponentType() string { return "Image" }

func (i *Image) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"type": "Image",
		"src":  i.Src,
	}
	if i.Width > 0 {
		result["width"] = i.Width
	}
	if i.Height > 0 {
		result["height"] = i.Height
	}
	if i.ScaleType != "" {
		result["scale-type"] = i.ScaleType
	}
	if i.AspectRatio != "" {
		result["aspect-ratio"] = i.AspectRatio
	}
	if i.AltText != "" {
		result["alt-text"] = i.AltText
	}
	if i.Visible != nil {
		result["visible"] = *i.Visible
	}
	return result
}

// EmbeddedLink represents an embedded link component
type EmbeddedLink struct {
	Text    string `json:"text"`
	OnClick *OnClickAction `json:"on-click-action,omitempty"`
	Visible *bool  `json:"visible,omitempty"`
}

func (e *EmbeddedLink) ComponentType() string { return "EmbeddedLink" }

func (e *EmbeddedLink) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"type": "EmbeddedLink",
		"text": e.Text,
	}
	if e.OnClick != nil {
		result["on-click-action"] = e.OnClick.ToMap()
	}
	if e.Visible != nil {
		result["visible"] = *e.Visible
	}
	return result
}

// =============================================================================
// Action Components
// =============================================================================

// OnClickAction represents an action triggered by clicking
type OnClickAction struct {
	Name    string                 `json:"name"` // navigate, complete, data_exchange
	Next    *NavigationTarget      `json:"next,omitempty"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

func (a *OnClickAction) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"name": a.Name,
	}
	if a.Next != nil {
		result["next"] = a.Next.ToMap()
	}
	if a.Payload != nil {
		result["payload"] = a.Payload
	}
	return result
}

// NavigationTarget represents a navigation target
type NavigationTarget struct {
	Type string `json:"type"` // screen, plugin
	Name string `json:"name"`
}

func (n *NavigationTarget) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"type": n.Type,
		"name": n.Name,
	}
}

// Footer represents a footer with action button
type Footer struct {
	Label          string         `json:"label"`
	OnClickAction  *OnClickAction `json:"on-click-action"`
	LeftCaption    string         `json:"left-caption,omitempty"`
	CenterCaption  string         `json:"center-caption,omitempty"`
	RightCaption   string         `json:"right-caption,omitempty"`
	Enabled        *bool          `json:"enabled,omitempty"`
	Visible        *bool          `json:"visible,omitempty"`
}

func (f *Footer) ComponentType() string { return "Footer" }

func (f *Footer) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"type":  "Footer",
		"label": f.Label,
	}
	if f.OnClickAction != nil {
		result["on-click-action"] = f.OnClickAction.ToMap()
	}
	if f.LeftCaption != "" {
		result["left-caption"] = f.LeftCaption
	}
	if f.CenterCaption != "" {
		result["center-caption"] = f.CenterCaption
	}
	if f.RightCaption != "" {
		result["right-caption"] = f.RightCaption
	}
	if f.Enabled != nil {
		result["enabled"] = *f.Enabled
	}
	if f.Visible != nil {
		result["visible"] = *f.Visible
	}
	return result
}

// =============================================================================
// Conditional Components
// =============================================================================

// If represents a conditional component
type If struct {
	Condition string      `json:"condition"`
	Then      []Component `json:"then"`
	Else      []Component `json:"else,omitempty"`
}

func (i *If) ComponentType() string { return "If" }

func (i *If) ToMap() map[string]interface{} {
	thenComponents := make([]map[string]interface{}, len(i.Then))
	for j, comp := range i.Then {
		thenComponents[j] = comp.ToMap()
	}

	result := map[string]interface{}{
		"type":      "If",
		"condition": i.Condition,
		"then":      thenComponents,
	}

	if len(i.Else) > 0 {
		elseComponents := make([]map[string]interface{}, len(i.Else))
		for j, comp := range i.Else {
			elseComponents[j] = comp.ToMap()
		}
		result["else"] = elseComponents
	}

	return result
}

// =============================================================================
// Helper Functions
// =============================================================================

// BoolPtr returns a pointer to a bool
func BoolPtr(b bool) *bool {
	return &b
}

// NavigateAction creates a navigate action
func NavigateAction(screenName string, payload map[string]interface{}) *OnClickAction {
	return &OnClickAction{
		Name: "navigate",
		Next: &NavigationTarget{
			Type: "screen",
			Name: screenName,
		},
		Payload: payload,
	}
}

// CompleteAction creates a complete action (closes the flow)
func CompleteAction(payload map[string]interface{}) *OnClickAction {
	return &OnClickAction{
		Name:    "complete",
		Payload: payload,
	}
}

// DataExchangeAction creates a data exchange action
func DataExchangeAction(payload map[string]interface{}) *OnClickAction {
	return &OnClickAction{
		Name:    "data_exchange",
		Payload: payload,
	}
}
