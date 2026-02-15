package payments

import (
	"time"
)

// =============================================================================
// Payment Status Types
// =============================================================================

// PaymentStatus represents the status of a payment
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusSuccess   PaymentStatus = "success"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusCanceled  PaymentStatus = "canceled"
	PaymentStatusRefunded  PaymentStatus = "refunded"
	PaymentStatusExpired   PaymentStatus = "expired"
)

// PaymentType represents the type of payment
type PaymentType string

const (
	PaymentTypeOrder      PaymentType = "order"
	PaymentTypeInvoice    PaymentType = "invoice"
	PaymentTypeSubscription PaymentType = "subscription"
)

// PaymentMethod represents supported payment methods
type PaymentMethod string

const (
	PaymentMethodUPI        PaymentMethod = "upi"         // India
	PaymentMethodNetBanking PaymentMethod = "netbanking"  // India
	PaymentMethodCard       PaymentMethod = "card"
	PaymentMethodPix        PaymentMethod = "pix"         // Brazil
	PaymentMethodBoleto     PaymentMethod = "boleto"      // Brazil
	PaymentMethodWallet     PaymentMethod = "wallet"
)

// =============================================================================
// Payment Types
// =============================================================================

// Payment represents a payment transaction
type Payment struct {
	ID                string        `json:"id"`
	OrganizationID    string        `json:"organization_id"`
	ChannelID         string        `json:"channel_id"`
	OrderID           string        `json:"order_id,omitempty"`
	ReferenceID       string        `json:"reference_id"`
	CustomerPhone     string        `json:"customer_phone"`
	Amount            int64         `json:"amount"` // In cents
	Currency          string        `json:"currency"`
	Status            PaymentStatus `json:"status"`
	Type              PaymentType   `json:"type"`
	Method            PaymentMethod `json:"method,omitempty"`
	GatewayPaymentID  string        `json:"gateway_payment_id,omitempty"`
	GatewayOrderID    string        `json:"gateway_order_id,omitempty"`
	Description       string        `json:"description,omitempty"`
	ExpiresAt         *time.Time    `json:"expires_at,omitempty"`
	PaidAt            *time.Time    `json:"paid_at,omitempty"`
	FailedAt          *time.Time    `json:"failed_at,omitempty"`
	RefundedAt        *time.Time    `json:"refunded_at,omitempty"`
	FailureReason     string        `json:"failure_reason,omitempty"`
	MessageID         string        `json:"message_id,omitempty"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

// PaymentItem represents an item in a payment
type PaymentItem struct {
	Name        string `json:"name"`
	Quantity    int    `json:"quantity"`
	UnitPrice   int64  `json:"unit_price"`
	TotalPrice  int64  `json:"total_price"`
	Currency    string `json:"currency"`
	Description string `json:"description,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
}

// =============================================================================
// Payment Request Types
// =============================================================================

// PaymentRequest represents a request to create a payment
type PaymentRequest struct {
	To              string         `json:"to"`
	ReferenceID     string         `json:"reference_id"`
	Type            PaymentType    `json:"type"`
	Amount          int64          `json:"amount"`
	Currency        string         `json:"currency"`
	Description     string         `json:"description,omitempty"`
	Items           []PaymentItem  `json:"items,omitempty"`
	ExpiresIn       time.Duration  `json:"expires_in,omitempty"` // Duration until expiry
	CallbackURL     string         `json:"callback_url,omitempty"`
	PaymentSettings *PaymentSettings `json:"payment_settings,omitempty"`
}

// PaymentSettings represents payment configuration
type PaymentSettings struct {
	AllowedMethods    []PaymentMethod `json:"allowed_methods,omitempty"`
	UPIDetails        *UPIDetails     `json:"upi_details,omitempty"`
	PixDetails        *PixDetails     `json:"pix_details,omitempty"`
}

// UPIDetails represents UPI-specific payment details (India)
type UPIDetails struct {
	VPA           string `json:"vpa"` // Virtual Payment Address
	MerchantName  string `json:"merchant_name"`
	TransactionNote string `json:"transaction_note,omitempty"`
}

// PixDetails represents Pix-specific payment details (Brazil)
type PixDetails struct {
	Key          string `json:"key"` // CPF, CNPJ, email, phone, or random key
	KeyType      string `json:"key_type"` // CPF, CNPJ, EMAIL, PHONE, RANDOM
	MerchantName string `json:"merchant_name"`
	MerchantCity string `json:"merchant_city"`
	Description  string `json:"description,omitempty"`
}

// =============================================================================
// Payment Response Types
// =============================================================================

// PaymentResponse represents the response from a payment request
type PaymentResponse struct {
	PaymentID     string        `json:"payment_id"`
	Status        PaymentStatus `json:"status"`
	PaymentURL    string        `json:"payment_url,omitempty"`
	QRCode        string        `json:"qr_code,omitempty"`
	QRCodeBase64  string        `json:"qr_code_base64,omitempty"`
	ExpiresAt     *time.Time    `json:"expires_at,omitempty"`
	MessageID     string        `json:"message_id,omitempty"`
}

// =============================================================================
// Payment Webhook Types
// =============================================================================

// PaymentWebhookPayload represents a payment webhook event
type PaymentWebhookPayload struct {
	Type          string                 `json:"type"`
	PaymentID     string                 `json:"payment_id"`
	ReferenceID   string                 `json:"reference_id"`
	Status        PaymentStatus          `json:"status"`
	Amount        int64                  `json:"amount"`
	Currency      string                 `json:"currency"`
	Method        PaymentMethod          `json:"method,omitempty"`
	PaidAt        *time.Time             `json:"paid_at,omitempty"`
	FailureReason string                 `json:"failure_reason,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// PaymentWebhookType constants
const (
	PaymentWebhookTypeCreated   = "payment.created"
	PaymentWebhookTypeCompleted = "payment.completed"
	PaymentWebhookTypeFailed    = "payment.failed"
	PaymentWebhookTypeCanceled  = "payment.canceled"
	PaymentWebhookTypeRefunded  = "payment.refunded"
	PaymentWebhookTypeExpired   = "payment.expired"
)

// =============================================================================
// Refund Types
// =============================================================================

// RefundStatus represents the status of a refund
type RefundStatus string

const (
	RefundStatusPending    RefundStatus = "pending"
	RefundStatusProcessing RefundStatus = "processing"
	RefundStatusSuccess    RefundStatus = "success"
	RefundStatusFailed     RefundStatus = "failed"
)

// Refund represents a payment refund
type Refund struct {
	ID              string       `json:"id"`
	PaymentID       string       `json:"payment_id"`
	Amount          int64        `json:"amount"`
	Currency        string       `json:"currency"`
	Status          RefundStatus `json:"status"`
	Reason          string       `json:"reason,omitempty"`
	GatewayRefundID string       `json:"gateway_refund_id,omitempty"`
	FailureReason   string       `json:"failure_reason,omitempty"`
	CreatedAt       time.Time    `json:"created_at"`
	ProcessedAt     *time.Time   `json:"processed_at,omitempty"`
}

// RefundRequest represents a request to refund a payment
type RefundRequest struct {
	PaymentID string `json:"payment_id"`
	Amount    int64  `json:"amount,omitempty"` // If 0, full refund
	Reason    string `json:"reason,omitempty"`
}

// =============================================================================
// Gateway Configuration Types
// =============================================================================

// GatewayType represents supported payment gateways
type GatewayType string

const (
	GatewayRazorpay GatewayType = "razorpay" // India
	GatewayPayU     GatewayType = "payu"     // India
	GatewayPagSeguro GatewayType = "pagseguro" // Brazil
	GatewayMercadoPago GatewayType = "mercadopago" // Brazil
	GatewayStripe   GatewayType = "stripe"    // Global
)

// GatewayConfig represents payment gateway configuration
type GatewayConfig struct {
	Type          GatewayType `json:"type"`
	APIKey        string      `json:"api_key"`
	APISecret     string      `json:"api_secret"`
	MerchantID    string      `json:"merchant_id,omitempty"`
	WebhookSecret string      `json:"webhook_secret,omitempty"`
	SandboxMode   bool        `json:"sandbox_mode"`
	WebhookURL    string      `json:"webhook_url,omitempty"`
	// Gateway-specific settings
	RazorpayConfig  *RazorpayConfig  `json:"razorpay_config,omitempty"`
	PagSeguroConfig *PagSeguroConfig `json:"pagseguro_config,omitempty"`
}

// RazorpayConfig represents Razorpay-specific configuration
type RazorpayConfig struct {
	AccountID      string `json:"account_id"`
	EnabledMethods []PaymentMethod `json:"enabled_methods"`
}

// PagSeguroConfig represents PagSeguro-specific configuration
type PagSeguroConfig struct {
	Email          string `json:"email"`
	Token          string `json:"token"`
	EnablePix      bool   `json:"enable_pix"`
	EnableBoleto   bool   `json:"enable_boleto"`
}

// =============================================================================
// Payment Statistics Types
// =============================================================================

// PaymentStats represents payment statistics
type PaymentStats struct {
	TotalPayments     int                      `json:"total_payments"`
	SuccessfulPayments int                     `json:"successful_payments"`
	FailedPayments    int                      `json:"failed_payments"`
	TotalAmount       int64                    `json:"total_amount"`
	RefundedAmount    int64                    `json:"refunded_amount"`
	Currency          string                   `json:"currency"`
	SuccessRate       float64                  `json:"success_rate"`
	ByStatus          map[PaymentStatus]int    `json:"by_status"`
	ByMethod          map[PaymentMethod]int    `json:"by_method"`
	DailyStats        []DailyPaymentStats      `json:"daily_stats"`
}

// DailyPaymentStats represents daily payment statistics
type DailyPaymentStats struct {
	Date         string  `json:"date"`
	Count        int     `json:"count"`
	Amount       int64   `json:"amount"`
	SuccessCount int     `json:"success_count"`
	FailedCount  int     `json:"failed_count"`
}

// =============================================================================
// WhatsApp Message Types for Payments
// =============================================================================

// PaymentMessage represents a payment request message
type PaymentMessage struct {
	To              string        `json:"to"`
	Type            string        `json:"type"` // "order_details"
	ReferenceID     string        `json:"reference_id"`
	PaymentStatus   string        `json:"payment_status"`
	OrderStatus     string        `json:"order_status,omitempty"`
	Description     string        `json:"description,omitempty"`
	TotalAmount     *Amount       `json:"total_amount,omitempty"`
	Items           []PaymentItem `json:"items,omitempty"`
	PaymentLink     *PaymentLink  `json:"payment_link,omitempty"`
}

// Amount represents an amount with currency
type Amount struct {
	Value    int64  `json:"value"` // In cents
	Currency string `json:"currency"`
	Offset   int    `json:"offset"` // Decimal places (e.g., 100 for cents)
}

// PaymentLink represents a payment link in a message
type PaymentLink struct {
	URL     string `json:"url"`
	Title   string `json:"title,omitempty"`
	Caption string `json:"caption,omitempty"`
}

// =============================================================================
// Payment Event Types
// =============================================================================

// PaymentEvent represents a payment event for pub/sub
type PaymentEvent struct {
	Type      string    `json:"type"`
	PaymentID string    `json:"payment_id"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Data      *Payment  `json:"data,omitempty"`
}

// Payment event type constants
const (
	PaymentEventCreated   = "payment.created"
	PaymentEventPending   = "payment.pending"
	PaymentEventSuccess   = "payment.success"
	PaymentEventFailed    = "payment.failed"
	PaymentEventCanceled  = "payment.canceled"
	PaymentEventRefunded  = "payment.refunded"
	PaymentEventExpired   = "payment.expired"
)

// NewPaymentEvent creates a new payment event
func NewPaymentEvent(eventType string, payment *Payment) *PaymentEvent {
	return &PaymentEvent{
		Type:      eventType,
		PaymentID: payment.ID,
		Status:    string(payment.Status),
		Timestamp: time.Now(),
		Data:      payment,
	}
}
