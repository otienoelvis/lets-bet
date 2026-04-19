// Package ng provides Nigeria-specific payment adapters
package ng

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/shopspring/decimal"

	"github.com/betting-platform/internal/core/domain"
)

// FlutterwaveConfig provides configuration for Flutterwave adapter
type FlutterwaveConfig struct {
	SecretKey     string `json:"secret_key"`
	PublicKey     string `json:"public_key"`
	Environment   string `json:"environment"` // "live" or "test"
	BaseURL       string `json:"base_url"`
	WebhookSecret string `json:"webhook_secret"`
	Currency      string `json:"currency"` // NGN for Nigeria
}

// DefaultFlutterwaveConfig returns default configuration
func DefaultFlutterwaveConfig() *FlutterwaveConfig {
	return &FlutterwaveConfig{
		Environment: "test",
		BaseURL:     "https://api.flutterwave.com/v3",
		Currency:    "NGN",
	}
}

// FlutterwaveAdapter provides Flutterwave payment processing for Nigeria
type FlutterwaveAdapter struct {
	config *FlutterwaveConfig
	client *http.Client
}

// NewFlutterwaveAdapter creates a new Flutterwave adapter
func NewFlutterwaveAdapter(config *FlutterwaveConfig) *FlutterwaveAdapter {
	if config == nil {
		config = DefaultFlutterwaveConfig()
	}

	return &FlutterwaveAdapter{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FlutterwavePaymentRequest represents a payment request to Flutterwave
type FlutterwavePaymentRequest struct {
	TxRef          string         `json:"tx_ref"`
	Amount         string         `json:"amount"`
	Currency       string         `json:"currency"`
	CustomerEmail  string         `json:"customer_email"`
	CustomerPhone  string         `json:"customer_phone,omitempty"`
	PaymentOptions string         `json:"payment_options"` // "card, banktransfer, ussd"
	RedirectURL    string         `json:"redirect_url"`
	PaymentPlan    int            `json:"payment_plan,omitempty"`
	SubAccounts    []SubAccount   `json:"subaccounts,omitempty"`
	Meta           map[string]any `json:"meta,omitempty"`
	Customization  Customization  `json:"customization"`
}

// SubAccount represents a Flutterwave subaccount
type SubAccount struct {
	ID              string `json:"id"`
	SplitPercentage int    `json:"split_percentage"`
}

// Customization represents Flutterwave payment customization
type Customization struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Logo        string `json:"logo,omitempty"`
}

// FlutterwavePaymentResponse represents Flutterwave payment response
type FlutterwavePaymentResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    struct {
		ID       int    `json:"id"`
		TxRef    string `json:"tx_ref"`
		FlwRef   string `json:"flw_ref"`
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
		Customer struct {
			Email string `json:"email"`
			Phone string `json:"phone"`
			Name  string `json:"name"`
		} `json:"customer"`
		PaymentLink string `json:"link"`
	} `json:"data"`
}

// FlutterwaveVerifyResponse represents Flutterwave transaction verification
type FlutterwaveVerifyResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    struct {
		ID            int    `json:"id"`
		TxRef         string `json:"tx_ref"`
		FlwRef        string `json:"flw_ref"`
		Amount        string `json:"amount"`
		Currency      string `json:"currency"`
		Status        string `json:"status"`
		PaymentType   string `json:"payment_type"`
		PaymentMethod string `json:"payment_method"`
		Customer      struct {
			Email string `json:"email"`
			Phone string `json:"phone"`
			Name  string `json:"name"`
		} `json:"customer"`
		Meta          map[string]any `json:"meta"`
		CreatedAt     string         `json:"created_at"`
		ChargedAmount string         `json:"charged_amount"`
		AppFee        string         `json:"app_fee"`
		MerchantFee   string         `json:"merchant_fee"`
	} `json:"data"`
}

// FlutterwavePayoutRequest represents a payout request to Flutterwave
type FlutterwavePayoutRequest struct {
	AccountBank     string `json:"account_bank"`
	AccountNumber   string `json:"account_number"`
	Amount          string `json:"amount"`
	Currency        string `json:"currency"`
	Narration       string `json:"narration"`
	Reference       string `json:"reference"`
	BeneficiaryName string `json:"beneficiary_name"`
}

// FlutterwavePayoutResponse represents Flutterwave payout response
type FlutterwavePayoutResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    struct {
		ID              int    `json:"id"`
		Reference       string `json:"reference"`
		Amount          string `json:"amount"`
		Currency        string `json:"currency"`
		Status          string `json:"status"`
		BeneficiaryName string `json:"beneficiary_name"`
		BankCode        string `json:"bank_code"`
		AccountNumber   string `json:"account_number"`
		CreatedAt       string `json:"created_at"`
	} `json:"data"`
}

// FlutterwaveWebhook represents a Flutterwave webhook payload
type FlutterwaveWebhook struct {
	Event string `json:"event"`
	Data  struct {
		ID            int    `json:"id"`
		TxRef         string `json:"tx_ref"`
		FlwRef        string `json:"flw_ref"`
		Amount        string `json:"amount"`
		Currency      string `json:"currency"`
		Status        string `json:"status"`
		PaymentType   string `json:"payment_type"`
		PaymentMethod string `json:"payment_method"`
		Customer      struct {
			Email string `json:"email"`
			Phone string `json:"phone"`
			Name  string `json:"name"`
		} `json:"customer"`
		Meta          map[string]any `json:"meta"`
		CreatedAt     string         `json:"created_at"`
		ChargedAmount string         `json:"charged_amount"`
		AppFee        string         `json:"app_fee"`
		MerchantFee   string         `json:"merchant_fee"`
	} `json:"data"`
}

// Deposit initiates a deposit via Flutterwave
func (f *FlutterwaveAdapter) Deposit(ctx context.Context, req *DepositRequest) (*DepositResponse, error) {
	// Generate transaction reference
	txRef := fmt.Sprintf("DEP-%s-%d", req.UserID, time.Now().Unix())

	// Create Flutterwave payment request
	fwReq := FlutterwavePaymentRequest{
		TxRef:          txRef,
		Amount:         req.Amount.StringFixed(2),
		Currency:       f.config.Currency,
		CustomerEmail:  req.Email,
		CustomerPhone:  req.PhoneNumber,
		PaymentOptions: "card, banktransfer, ussd",
		RedirectURL:    req.RedirectURL,
		Customization: Customization{
			Title:       "Betting Platform Deposit",
			Description: fmt.Sprintf("Deposit %s %s to your betting account", req.Amount.StringFixed(2), f.config.Currency),
		},
		Meta: map[string]any{
			"user_id":    req.UserID,
			"platform":   "betting-platform",
			"deposit_id": req.DepositID,
		},
	}

	// Marshal request
	body, err := json.Marshal(fwReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/payments", f.config.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+f.config.SecretKey)

	// Make request
	resp, err := f.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var fwResp FlutterwavePaymentResponse
	if err := json.Unmarshal(respBody, &fwResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.StatusCode != http.StatusOK || fwResp.Status != "success" {
		return nil, fmt.Errorf("Flutterwave API error: %s - %s", fwResp.Status, fwResp.Message)
	}

	// Return deposit response
	return &DepositResponse{
		TransactionID: fwResp.Data.FlwRef,
		Reference:     fwResp.Data.TxRef,
		PaymentLink:   fwResp.Data.PaymentLink,
		Status:        domain.DepositStatusPending,
		Amount:        req.Amount,
		Currency:      f.config.Currency,
		CreatedAt:     time.Now(),
	}, nil
}

// VerifyDeposit verifies a deposit transaction with Flutterwave
func (f *FlutterwaveAdapter) VerifyDeposit(ctx context.Context, transactionID string) (*DepositVerification, error) {
	// Create verification request
	url := fmt.Sprintf("%s/transactions/%s/verify", f.config.BaseURL, transactionID)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+f.config.SecretKey)

	// Make request
	resp, err := f.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var fwResp FlutterwaveVerifyResponse
	if err := json.Unmarshal(respBody, &fwResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.StatusCode != http.StatusOK || fwResp.Status != "success" {
		return nil, fmt.Errorf("Flutterwave API error: %s - %s", fwResp.Status, fwResp.Message)
	}

	// Parse amount
	amount, err := decimal.NewFromString(fwResp.Data.Amount)
	if err != nil {
		return nil, fmt.Errorf("failed to parse amount: %w", err)
	}

	// Determine status
	status := domain.DepositStatusFailed
	if fwResp.Data.Status == "successful" {
		status = domain.DepositStatusCompleted
	}

	// Extract user ID from meta
	var userID string
	if fwResp.Data.Meta != nil {
		if uid, ok := fwResp.Data.Meta["user_id"].(string); ok {
			userID = uid
		}
	}

	return &DepositVerification{
		TransactionID: fwResp.Data.FlwRef,
		Reference:     fwResp.Data.TxRef,
		Status:        status,
		Amount:        amount,
		Currency:      fwResp.Data.Currency,
		PaymentMethod: fwResp.Data.PaymentMethod,
		ProcessedAt:   time.Now(), // Would parse fwResp.Data.CreatedAt in real implementation
		UserID:        userID,
		Fees:          decimal.Zero, // Would calculate from fwResp.Data.AppFee + fwResp.Data.MerchantFee
	}, nil
}

// Payout initiates a payout via Flutterwave
func (f *FlutterwaveAdapter) Payout(ctx context.Context, req *PayoutRequest) (*PayoutResponse, error) {
	// Generate reference
	reference := fmt.Sprintf("PAYOUT-%s-%d", req.UserID, time.Now().Unix())

	// Create Flutterwave payout request
	fwReq := FlutterwavePayoutRequest{
		AccountBank:     req.BankCode,
		AccountNumber:   req.AccountNumber,
		Amount:          req.Amount.StringFixed(2),
		Currency:        f.config.Currency,
		Narration:       req.Narration,
		Reference:       reference,
		BeneficiaryName: req.BeneficiaryName,
	}

	// Marshal request
	body, err := json.Marshal(fwReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/transfers", f.config.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+f.config.SecretKey)

	// Make request
	resp, err := f.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var fwResp FlutterwavePayoutResponse
	if err := json.Unmarshal(respBody, &fwResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.StatusCode != http.StatusOK || fwResp.Status != "success" {
		return nil, fmt.Errorf("Flutterwave API error: %s - %s", fwResp.Status, fwResp.Message)
	}

	// Return payout response
	return &PayoutResponse{
		TransactionID: fmt.Sprintf("%d", fwResp.Data.ID),
		Reference:     fwResp.Data.Reference,
		Status:        domain.PayoutStatusPending,
		Amount:        req.Amount,
		Currency:      f.config.Currency,
		CreatedAt:     time.Now(),
	}, nil
}

// VerifyPayout verifies a payout transaction with Flutterwave
func (f *FlutterwaveAdapter) VerifyPayout(ctx context.Context, transactionID string) (*PayoutVerification, error) {
	// In a real implementation, this would query Flutterwave for transfer status
	// For now, we'll return a placeholder
	return &PayoutVerification{
		TransactionID: transactionID,
		Status:        domain.PayoutStatusPending,
		ProcessedAt:   time.Now(),
	}, nil
}

// ProcessWebhook processes a Flutterwave webhook
func (f *FlutterwaveAdapter) ProcessWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error) {
	// Verify webhook signature
	if !f.verifyWebhookSignature(payload, signature) {
		return nil, fmt.Errorf("invalid webhook signature")
	}

	// Parse webhook
	var webhook FlutterwaveWebhook
	if err := json.Unmarshal(payload, &webhook); err != nil {
		return nil, fmt.Errorf("failed to parse webhook: %w", err)
	}

	// Determine event type
	eventType := domain.WebhookEventTypeUnknown
	switch webhook.Event {
	case "charge.completed":
		eventType = domain.WebhookEventTypePaymentCompleted
	case "transfer.completed":
		eventType = domain.WebhookEventTypePayoutCompleted
	case "charge.failed":
		eventType = domain.WebhookEventTypePaymentFailed
	case "transfer.failed":
		eventType = domain.WebhookEventTypePayoutFailed
	}

	// Extract user ID from meta
	var userID string
	if webhook.Data.Meta != nil {
		if uid, ok := webhook.Data.Meta["user_id"].(string); ok {
			userID = uid
		}
	}

	// Parse amount
	amount, err := decimal.NewFromString(webhook.Data.Amount)
	if err != nil {
		return nil, fmt.Errorf("failed to parse amount: %w", err)
	}

	return &WebhookEvent{
		EventID:       webhook.Data.FlwRef,
		EventType:     eventType,
		TransactionID: webhook.Data.FlwRef,
		Reference:     webhook.Data.TxRef,
		Status:        webhook.Data.Status,
		Amount:        amount,
		Currency:      webhook.Data.Currency,
		UserID:        userID,
		ProcessedAt:   time.Now(), // Would parse webhook.Data.CreatedAt in real implementation
		RawPayload:    payload,
	}, nil
}

// verifyWebhookSignature verifies Flutterwave webhook signature
func (f *FlutterwaveAdapter) verifyWebhookSignature(payload []byte, signature string) bool {
	if f.config.WebhookSecret == "" {
		return true // Skip verification if no secret configured
	}

	// Compute HMAC SHA256
	h := hmac.New(sha256.New, []byte(f.config.WebhookSecret))
	h.Write(payload)
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	return signature == expectedSignature
}

// GetBanks returns list of supported banks
func (f *FlutterwaveAdapter) GetBanks(ctx context.Context, country string) ([]Bank, error) {
	// Create request
	url := fmt.Sprintf("%s/banks/%s", f.config.BaseURL, country)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+f.config.SecretKey)

	// Make request
	resp, err := f.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response (simplified for this example)
	var banksResp struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Data    []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
			Code string `json:"code"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &banksResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.StatusCode != http.StatusOK || banksResp.Status != "success" {
		return nil, fmt.Errorf("Flutterwave API error: %s - %s", banksResp.Status, banksResp.Message)
	}

	// Convert to Bank objects
	var banks []Bank
	for _, bank := range banksResp.Data {
		banks = append(banks, Bank{
			Code: bank.Code,
			Name: bank.Name,
		})
	}

	return banks, nil
}

// Request types
type DepositRequest struct {
	UserID      string
	DepositID   string
	Amount      decimal.Decimal
	Email       string
	PhoneNumber string
	RedirectURL string
}

type DepositResponse struct {
	TransactionID string
	Reference     string
	PaymentLink   string
	Status        domain.DepositStatus
	Amount        decimal.Decimal
	Currency      string
	CreatedAt     time.Time
}

type DepositVerification struct {
	TransactionID string
	Reference     string
	Status        domain.DepositStatus
	Amount        decimal.Decimal
	Currency      string
	PaymentMethod string
	ProcessedAt   time.Time
	UserID        string
	Fees          decimal.Decimal
}

type PayoutRequest struct {
	UserID          string
	Amount          decimal.Decimal
	BankCode        string
	AccountNumber   string
	BeneficiaryName string
	Narration       string
}

type PayoutResponse struct {
	TransactionID string
	Reference     string
	Status        domain.PayoutStatus
	Amount        decimal.Decimal
	Currency      string
	CreatedAt     time.Time
}

type PayoutVerification struct {
	TransactionID string
	Status        domain.PayoutStatus
	ProcessedAt   time.Time
}

type WebhookEvent struct {
	EventID       string
	EventType     domain.WebhookEventType
	TransactionID string
	Reference     string
	Status        string
	Amount        decimal.Decimal
	Currency      string
	UserID        string
	ProcessedAt   time.Time
	RawPayload    []byte
}

type Bank struct {
	Code string
	Name string
}
