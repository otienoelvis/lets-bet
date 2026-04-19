package ke

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/betting-platform/internal/core/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrMPesaTimeout       = errors.New("mpesa request timeout")
	ErrInsufficientFunds  = errors.New("mpesa insufficient funds")
	ErrInvalidPhoneNumber = errors.New("invalid phone number")
)

// MPesaConfig holds Safaricom Daraja API credentials
type MPesaConfig struct {
	ConsumerKey        string
	ConsumerSecret     string
	ShortCode          string // Paybill/Till number
	PassKey            string // For STK Push
	InitiatorName      string // For B2C
	SecurityCredential string // Encrypted initiator password
	Environment        string // sandbox or production
}

// MPesaClient handles all M-Pesa operations
type MPesaClient struct {
	config     MPesaConfig
	httpClient *http.Client
	baseURL    string
}

func NewMPesaClient(config MPesaConfig) *MPesaClient {
	baseURL := "https://api.safaricom.co.ke"
	if config.Environment == "sandbox" {
		baseURL = "https://sandbox.safaricom.co.ke"
	}

	return &MPesaClient{
		config:     config,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
	}
}

// ============ AUTHENTICATION ============

type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
}

func (c *MPesaClient) GetAccessToken(ctx context.Context) (string, error) {
	url := c.baseURL + "/oauth/v1/generate?grant_type=client_credentials"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	// Basic Auth with consumer key and secret
	auth := base64.StdEncoding.EncodeToString(
		[]byte(c.config.ConsumerKey + ":" + c.config.ConsumerSecret),
	)
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var tokenResp AccessTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	return tokenResp.AccessToken, nil
}

type STKPushRequest struct {
	BusinessShortCode string `json:"BusinessShortCode"`
	Password          string `json:"Password"`
	Timestamp         string `json:"Timestamp"`
	TransactionType   string `json:"TransactionType"`
	Amount            string `json:"Amount"`
	PartyA            string `json:"PartyA"` // Customer phone
	PartyB            string `json:"PartyB"` // Shortcode
	PhoneNumber       string `json:"PhoneNumber"`
	CallBackURL       string `json:"CallBackURL"`
	AccountReference  string `json:"AccountReference"`
	TransactionDesc   string `json:"TransactionDesc"`
}

type STKPushResponse struct {
	MerchantRequestID   string `json:"MerchantRequestID"`
	CheckoutRequestID   string `json:"CheckoutRequestID"`
	ResponseCode        string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
	CustomerMessage     string `json:"CustomerMessage"`
}

// InitiateDeposit triggers STK Push (Lipa Na M-Pesa Online)
func (c *MPesaClient) InitiateDeposit(ctx context.Context, phoneNumber string, amount decimal.Decimal, reference string) (*STKPushResponse, error) {
	token, err := c.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	// Generate timestamp
	timestamp := time.Now().Format("20060102150405")

	// Generate password: Base64(ShortCode + PassKey + Timestamp)
	passwordStr := c.config.ShortCode + c.config.PassKey + timestamp
	password := base64.StdEncoding.EncodeToString([]byte(passwordStr))

	// Format phone number (254XXXXXXXXX)
	if len(phoneNumber) == 10 && phoneNumber[0] == '0' {
		phoneNumber = "254" + phoneNumber[1:]
	}

	reqBody := STKPushRequest{
		BusinessShortCode: c.config.ShortCode,
		Password:          password,
		Timestamp:         timestamp,
		TransactionType:   "CustomerPayBillOnline",
		Amount:            amount.StringFixed(0), // M-Pesa doesn't accept decimals
		PartyA:            phoneNumber,
		PartyB:            c.config.ShortCode,
		PhoneNumber:       phoneNumber,
		CallBackURL:       "https://yourdomain.com/api/mpesa/callback",
		AccountReference:  reference,
		TransactionDesc:   "Deposit",
	}

	jsonData, _ := json.Marshal(reqBody)
	url := c.baseURL + "/mpesa/stkpush/v1/processrequest"

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var stkResp STKPushResponse
	if err := json.Unmarshal(body, &stkResp); err != nil {
		return nil, err
	}

	if stkResp.ResponseCode != "0" {
		return nil, fmt.Errorf("mpesa error: %s", stkResp.ResponseDescription)
	}

	return &stkResp, nil
}

// ============ B2C (WITHDRAWAL) ============

type B2CRequest struct {
	InitiatorName      string `json:"InitiatorName"`
	SecurityCredential string `json:"SecurityCredential"`
	CommandID          string `json:"CommandID"`
	Amount             string `json:"Amount"`
	PartyA             string `json:"PartyA"` // Shortcode
	PartyB             string `json:"PartyB"` // Customer phone
	Remarks            string `json:"Remarks"`
	QueueTimeOutURL    string `json:"QueueTimeOutURL"`
	ResultURL          string `json:"ResultURL"`
	Occasion           string `json:"Occasion"`
}

type B2CResponse struct {
	ConversationID           string `json:"ConversationID"`
	OriginatorConversationID string `json:"OriginatorConversationID"`
	ResponseCode             string `json:"ResponseCode"`
	ResponseDescription      string `json:"ResponseDescription"`
}

// InitiateWithdrawal sends money from business to customer
func (c *MPesaClient) InitiateWithdrawal(ctx context.Context, phoneNumber string, amount decimal.Decimal, reference string) (*B2CResponse, error) {
	token, err := c.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	// Format phone number
	if len(phoneNumber) == 10 && phoneNumber[0] == '0' {
		phoneNumber = "254" + phoneNumber[1:]
	}

	reqBody := B2CRequest{
		InitiatorName:      c.config.InitiatorName,
		SecurityCredential: c.config.SecurityCredential,
		CommandID:          "BusinessPayment",
		Amount:             amount.StringFixed(0),
		PartyA:             c.config.ShortCode,
		PartyB:             phoneNumber,
		Remarks:            "Withdrawal - " + reference,
		QueueTimeOutURL:    "https://yourdomain.com/api/mpesa/timeout",
		ResultURL:          "https://yourdomain.com/api/mpesa/b2c-result",
		Occasion:           reference,
	}

	jsonData, _ := json.Marshal(reqBody)
	url := c.baseURL + "/mpesa/b2c/v1/paymentrequest"

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var b2cResp B2CResponse
	if err := json.Unmarshal(body, &b2cResp); err != nil {
		return nil, err
	}

	if b2cResp.ResponseCode != "0" {
		return nil, fmt.Errorf("mpesa error: %s", b2cResp.ResponseDescription)
	}

	return &b2cResp, nil
}

// MPesaAdapter implements the payment provider interface for Kenya
type MPesaAdapter struct {
	client      *MPesaClient
	walletRepo  WalletRepository
	depositRepo MPesaDepositRepository
}

type WalletRepository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Wallet, error)
	UpdateBalance(ctx context.Context, wallet *domain.Wallet, tx *domain.Transaction) error
}

type MPesaDepositRepository interface {
	Create(ctx context.Context, deposit *domain.MPesaDeposit) error
	GetByCheckoutRequestID(ctx context.Context, checkoutRequestID string) (*domain.MPesaDeposit, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.MPesaDeposit, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.MPesaDepositStatus, receiptNumber *string, transactionDate *time.Time, resultCode *string, resultDesc *string) error
	GetPendingDeposits(ctx context.Context, olderThan time.Duration) ([]*domain.MPesaDeposit, error)
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
}

func NewMPesaAdapter(client *MPesaClient, walletRepo WalletRepository, depositRepo MPesaDepositRepository) *MPesaAdapter {
	return &MPesaAdapter{
		client:      client,
		walletRepo:  walletRepo,
		depositRepo: depositRepo,
	}
}

// Deposit initiates M-Pesa deposit and returns transaction reference
func (a *MPesaAdapter) Deposit(ctx context.Context, userID uuid.UUID, phoneNumber string, amount decimal.Decimal) (string, error) {
	reference := uuid.New().String()

	resp, err := a.client.InitiateDeposit(ctx, phoneNumber, amount, reference)
	if err != nil {
		return "", err
	}

	// Store pending deposit in database
	deposit := &domain.MPesaDeposit{
		ID:                uuid.New(),
		MerchantRequestID: resp.MerchantRequestID,
		CheckoutRequestID: resp.CheckoutRequestID,
		UserID:            userID,
		PhoneNumber:       phoneNumber,
		Amount:            amount,
		Currency:          "KES",
		Status:            domain.MPesaDepositStatusPending,
		Reference:         reference,
		Description:       "M-Pesa deposit",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := a.depositRepo.Create(ctx, deposit); err != nil {
		return "", fmt.Errorf("failed to store deposit: %w", err)
	}

	return resp.CheckoutRequestID, nil
}

// Withdraw sends instant payout to user's M-Pesa
func (a *MPesaAdapter) Withdraw(ctx context.Context, userID uuid.UUID, phoneNumber string, amount decimal.Decimal) error {
	// 1. Get wallet and verify balance
	wallet, err := a.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}

	if !wallet.CanWithdraw(amount) {
		return ErrInsufficientFunds
	}

	// 2. Deduct from wallet first (pessimistic approach)
	tx := &domain.Transaction{
		ID:            uuid.New(),
		WalletID:      wallet.ID,
		UserID:        userID,
		Type:          domain.TransactionTypeWithdrawal,
		Amount:        amount.Neg(),
		Currency:      "KES",
		BalanceBefore: wallet.Balance,
		BalanceAfter:  wallet.Balance.Sub(amount),
		Status:        domain.TransactionStatusPending,
		Description:   "Withdrawal to M-Pesa",
		CreatedAt:     time.Now(),
		CountryCode:   "KE",
		ProviderName:  "MPESA",
	}

	wallet.Balance = wallet.Balance.Sub(amount)
	wallet.Version++

	if err := a.walletRepo.UpdateBalance(ctx, wallet, tx); err != nil {
		return err
	}

	// 3. Initiate M-Pesa B2C
	resp, err := a.client.InitiateWithdrawal(ctx, phoneNumber, amount, tx.ID.String())
	if err != nil {
		// In production, trigger a refund/reversal
		return err
	}

	// Store the M-Pesa conversation ID for tracking
	tx.ProviderTxnID = resp.ConversationID

	return nil
}

// HandleSTKCallback processes M-Pesa STK push callback
func (a *MPesaAdapter) HandleSTKCallback(ctx context.Context, merchantRequestID, checkoutRequestID, resultCode, resultDesc string, mpesaReceiptNumber *string, transactionDate *time.Time) error {
	// Find the deposit record
	deposit, err := a.depositRepo.GetByCheckoutRequestID(ctx, checkoutRequestID)
	if err != nil {
		return fmt.Errorf("deposit not found: %w", err)
	}

	// Update deposit status based on result code
	if resultCode == "0" && mpesaReceiptNumber != nil {
		// Successful deposit
		if err := a.completeDeposit(ctx, deposit, *mpesaReceiptNumber, *transactionDate); err != nil {
			return fmt.Errorf("failed to complete deposit: %w", err)
		}
	} else {
		// Failed deposit
		if err := a.depositRepo.UpdateStatus(ctx, deposit.ID, domain.MPesaDepositStatusFailed, mpesaReceiptNumber, transactionDate, &resultCode, &resultDesc); err != nil {
			return fmt.Errorf("failed to update deposit status: %w", err)
		}
	}

	return nil
}

// completeDeposit credits the user's wallet and marks deposit as completed
func (a *MPesaAdapter) completeDeposit(ctx context.Context, deposit *domain.MPesaDeposit, receiptNumber string, transactionDate time.Time) error {
	// Get user's wallet
	wallet, err := a.walletRepo.GetByUserID(ctx, deposit.UserID)
	if err != nil {
		return fmt.Errorf("failed to get wallet: %w", err)
	}

	// Create transaction record
	tx := &domain.Transaction{
		ID:            uuid.New(),
		WalletID:      wallet.ID,
		UserID:        deposit.UserID,
		Type:          domain.TransactionTypeDeposit,
		Amount:        deposit.Amount,
		Currency:      deposit.Currency,
		BalanceBefore: wallet.Balance,
		BalanceAfter:  wallet.Balance.Add(deposit.Amount),
		ReferenceID:   &deposit.ID,
		ReferenceType: "MPESA_DEPOSIT",
		ProviderTxnID: receiptNumber,
		ProviderName:  "MPESA",
		Status:        domain.TransactionStatusCompleted,
		Description:   "M-Pesa deposit",
		CreatedAt:     time.Now(),
		CompletedAt:   &transactionDate,
		CountryCode:   "KE",
	}

	// Update wallet balance
	wallet.Balance = wallet.Balance.Add(deposit.Amount)
	wallet.Version++

	// Update wallet and create transaction
	if err := a.walletRepo.UpdateBalance(ctx, wallet, tx); err != nil {
		return fmt.Errorf("failed to update wallet: %w", err)
	}

	// Update deposit status
	if err := a.depositRepo.UpdateStatus(ctx, deposit.ID, domain.MPesaDepositStatusCompleted, &receiptNumber, &transactionDate, nil, nil); err != nil {
		return fmt.Errorf("failed to update deposit status: %w", err)
	}

	return nil
}

// GetDeposits retrieves user's deposit history
func (a *MPesaAdapter) GetDeposits(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.MPesaDeposit, error) {
	return a.depositRepo.GetByUserID(ctx, userID, limit, offset)
}

// GetDepositByCheckoutRequestID retrieves a specific deposit
func (a *MPesaAdapter) GetDepositByCheckoutRequestID(ctx context.Context, checkoutRequestID string) (*domain.MPesaDeposit, error) {
	return a.depositRepo.GetByCheckoutRequestID(ctx, checkoutRequestID)
}

// GetPendingDeposits retrieves deposits that are still pending
func (a *MPesaAdapter) GetPendingDeposits(ctx context.Context, olderThan time.Duration) ([]*domain.MPesaDeposit, error) {
	return a.depositRepo.GetPendingDeposits(ctx, olderThan)
}
