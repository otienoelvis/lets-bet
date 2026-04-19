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
)

// NewFlutterwaveAdapter creates a new Flutterwave adapter for Nigeria
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

// CreatePayment creates a new payment with Flutterwave
func (fwa *FlutterwaveAdapter) CreatePayment(ctx context.Context, req *PaymentRequest) (*PaymentResponse, error) {
	// Convert to Flutterwave request format
	fwReq := map[string]interface{}{
		"tx_ref":          req.TxRef,
		"amount":          req.Amount.StringFixed(2),
		"currency":        req.Currency,
		"customer_email":  req.CustomerEmail,
		"customer_phone":  req.CustomerPhone,
		"customer_name":   req.CustomerName,
		"payment_options": "card,banktransfer,mobilemoney",
		"redirect_url":    req.RedirectURL,
		"customization": map[string]string{
			"title":       "Betting Platform Payment",
			"description": "Payment for betting services",
		},
	}

	// Make API request
	resp, err := fwa.makeRequest(ctx, "POST", "/payments", fwReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	var fwResp map[string]interface{}
	if err := json.Unmarshal(resp, &fwResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to domain response
	paymentResp := &PaymentResponse{
		Status:  fwResp["status"].(string),
		Message: fwResp["message"].(string),
	}

	// Parse data if available
	if data, ok := fwResp["data"].(map[string]interface{}); ok {
		paymentResp.Data = PaymentData{
			ID:            fmt.Sprintf("%.0f", data["id"]),
			TxRef:         data["tx_ref"].(string),
			FlwRef:        data["flw_ref"].(string),
			Amount:        decimal.RequireFromString(data["amount"].(string)),
			Currency:      data["currency"].(string),
			Status:        data["status"].(string),
			PaymentType:   "card",
			CreatedAt:     time.Now(),
			ChargedAmount: decimal.RequireFromString(data["amount"].(string)),
		}
	}

	return paymentResp, nil
}

// VerifyPayment verifies a payment with Flutterwave
func (fwa *FlutterwaveAdapter) VerifyPayment(ctx context.Context, txRef string) (*TransactionVerificationResponse, error) {
	// Make verification request
	resp, err := fwa.makeRequest(ctx, "GET", "/transactions/"+txRef+"/verify", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to verify payment: %w", err)
	}

	var fwResp map[string]interface{}
	if err := json.Unmarshal(resp, &fwResp); err != nil {
		return nil, fmt.Errorf("failed to parse verification response: %w", err)
	}

	// Convert to domain response
	verificationResp := &TransactionVerificationResponse{
		Status:  fwResp["status"].(string),
		Message: fwResp["message"].(string),
	}

	// Parse data if available
	if data, ok := fwResp["data"].(map[string]interface{}); ok {
		verificationResp.Data = TransactionVerificationData{
			ID:            fmt.Sprintf("%.0f", data["id"]),
			TxRef:         data["tx_ref"].(string),
			FlwRef:        data["flw_ref"].(string),
			Amount:        decimal.RequireFromString(data["amount"].(string)),
			Currency:      data["currency"].(string),
			Status:        data["status"].(string),
			PaymentType:   "card",
			CreatedAt:     time.Now(),
			ChargedAmount: decimal.RequireFromString(data["charged_amount"].(string)),
		}
	}

	return verificationResp, nil
}

// ProcessRefund processes a refund
func (fwa *FlutterwaveAdapter) ProcessRefund(ctx context.Context, req *RefundRequest) (*RefundResponse, error) {
	fwReq := map[string]interface{}{
		"id":     req.ID,
		"amount": req.Amount.StringFixed(2),
		"reason": req.Reason,
	}

	resp, err := fwa.makeRequest(ctx, "POST", "/refunds", fwReq)
	if err != nil {
		return nil, fmt.Errorf("failed to process refund: %w", err)
	}

	var fwResp map[string]interface{}
	if err := json.Unmarshal(resp, &fwResp); err != nil {
		return nil, fmt.Errorf("failed to parse refund response: %w", err)
	}

	refundResp := &RefundResponse{
		Status:  fwResp["status"].(string),
		Message: fwResp["message"].(string),
	}

	if data, ok := fwResp["data"].(map[string]interface{}); ok {
		refundResp.Data = RefundData{
			ID:        fmt.Sprintf("%.0f", data["id"]),
			TxRef:     data["tx_ref"].(string),
			FlwRef:    data["flw_ref"].(string),
			Amount:    decimal.RequireFromString(data["amount"].(string)),
			Currency:  data["currency"].(string),
			Status:    data["status"].(string),
			RefundRef: data["refund_ref"].(string),
			CreatedAt: time.Now(),
		}
	}

	return refundResp, nil
}

// makeRequest makes an HTTP request to Flutterwave API
func (fwa *FlutterwaveAdapter) makeRequest(ctx context.Context, method, endpoint string, body interface{}) ([]byte, error) {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	url := fwa.config.BaseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+fwa.config.SecretKey)

	resp, err := fwa.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// verifyWebhookSignature verifies Flutterwave webhook signature
func (fwa *FlutterwaveAdapter) verifyWebhookSignature(payload []byte, signature string) bool {
	if fwa.config.WebhookSecret == "" {
		return true // Skip verification if no secret configured
	}

	h := hmac.New(sha256.New, []byte(fwa.config.WebhookSecret))
	h.Write(payload)
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	return signature == expectedSignature
}

// GetMetrics returns Flutterwave adapter metrics
func (fwa *FlutterwaveAdapter) GetMetrics(ctx context.Context) (*FlutterwaveMetrics, error) {
	// In a real implementation, these would be calculated from actual data
	return &FlutterwaveMetrics{
		TotalTransactions:       1500,
		SuccessfulTransactions: 1425,
		FailedTransactions:     75,
		TotalAmount:            decimal.NewFromInt(75000),
		SuccessRate:            decimal.NewFromFloat(0.95),
		AverageAmount:          decimal.NewFromInt(50),
		LastTransactionTime:    time.Now().Add(-2 * time.Hour),
	}, nil
}
