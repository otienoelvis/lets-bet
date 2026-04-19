package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// PaymentStatus represents the status of payment transactions
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "PENDING"
	PaymentStatusCompleted PaymentStatus = "COMPLETED"
	PaymentStatusFailed    PaymentStatus = "FAILED"
	PaymentStatusCancelled PaymentStatus = "CANCELLED"
)

// DepositStatus represents the status of deposit transactions
type DepositStatus string

const (
	DepositStatusPending   DepositStatus = "PENDING"
	DepositStatusCompleted DepositStatus = "COMPLETED"
	DepositStatusFailed    DepositStatus = "FAILED"
	DepositStatusCancelled DepositStatus = "CANCELLED"
)

// PayoutStatus represents the status of payout transactions
type PayoutStatus string

const (
	PayoutStatusPending   PayoutStatus = "PENDING"
	PayoutStatusCompleted PayoutStatus = "COMPLETED"
	PayoutStatusFailed    PayoutStatus = "FAILED"
	PayoutStatusCancelled PayoutStatus = "CANCELLED"
)

// WebhookEventType represents different types of webhook events
type WebhookEventType string

const (
	WebhookEventTypeUnknown          WebhookEventType = "UNKNOWN"
	WebhookEventTypePaymentCompleted WebhookEventType = "PAYMENT_COMPLETED"
	WebhookEventTypePaymentFailed    WebhookEventType = "PAYMENT_FAILED"
	WebhookEventTypePayoutCompleted  WebhookEventType = "PAYOUT_COMPLETED"
	WebhookEventTypePayoutFailed     WebhookEventType = "PAYOUT_FAILED"
)

// PaymentMethod represents different payment methods
type PaymentMethod string

const (
	PaymentMethodCard         PaymentMethod = "CARD"
	PaymentMethodBankTransfer PaymentMethod = "BANK_TRANSFER"
	PaymentMethodUSSD         PaymentMethod = "USSD"
	PaymentMethodMobileMoney  PaymentMethod = "MOBILE_MONEY"
)

// Payment represents a generic payment transaction
type Payment struct {
	ID            string          `json:"id" db:"id"`
	UserID        string          `json:"user_id" db:"user_id"`
	Type          TransactionType `json:"type" db:"type"`
	Amount        decimal.Decimal `json:"amount" db:"amount"`
	Currency      string          `json:"currency" db:"currency"`
	Status        PaymentStatus   `json:"status" db:"status"`
	PaymentMethod PaymentMethod   `json:"payment_method" db:"payment_method"`
	ProviderName  string          `json:"provider_name" db:"provider_name"`
	ProviderTxnID string          `json:"provider_txn_id" db:"provider_txn_id"`
	ReferenceID   *string         `json:"reference_id,omitempty" db:"reference_id"`
	ReferenceType *string         `json:"reference_type,omitempty" db:"reference_type"`
	Description   string          `json:"description" db:"description"`
	Meta          map[string]any  `json:"meta,omitempty" db:"meta"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	CompletedAt   *time.Time      `json:"completed_at,omitempty" db:"completed_at"`
	ProcessedAt   *time.Time      `json:"processed_at,omitempty" db:"processed_at"`
	FailureReason *string         `json:"failure_reason,omitempty" db:"failure_reason"`
}

// IsCompleted returns true if the payment is in a final state
func (p *Payment) IsCompleted() bool {
	return p.Status == PaymentStatusCompleted ||
		p.Status == PaymentStatusFailed ||
		p.Status == PaymentStatusCancelled
}

// IsSuccessful returns true if the payment was completed successfully
func (p *Payment) IsSuccessful() bool {
	return p.Status == PaymentStatusCompleted
}

// MarkCompleted updates the payment to completed status
func (p *Payment) MarkCompleted() {
	p.Status = PaymentStatusCompleted
	now := time.Now()
	p.CompletedAt = &now
	p.ProcessedAt = &now
}

// MarkFailed updates the payment to failed status
func (p *Payment) MarkFailed(reason string) {
	p.Status = PaymentStatusFailed
	now := time.Now()
	p.ProcessedAt = &now
	p.FailureReason = &reason
}

// MarkCancelled updates the payment to cancelled status
func (p *Payment) MarkCancelled(reason string) {
	p.Status = PaymentStatusCancelled
	now := time.Now()
	p.ProcessedAt = &now
	p.FailureReason = &reason
}
