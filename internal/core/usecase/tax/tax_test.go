package tax_test

import (
	"errors"
	"testing"

	"github.com/betting-platform/internal/core/usecase/tax"
	"github.com/shopspring/decimal"
)

func TestApplyStakeTax_Kenya(t *testing.T) {
	t.Parallel()
	e, err := tax.Default()
	if err != nil {
		t.Fatalf("Failed to create tax engine: %v", err)
	}

	got, err := e.ApplyStakeTax("KE", decimal.NewFromInt(1000))
	if err != nil {
		t.Fatalf("ApplyStakeTax failed: %v", err)
	}

	if !got.StakeTax.Equal(decimal.NewFromInt(150)) {
		t.Fatalf("StakeTax = %s, want 150", got.StakeTax)
	}
	if !got.NetStake.Equal(decimal.NewFromInt(850)) {
		t.Fatalf("NetStake = %s, want 850", got.NetStake)
	}
	if !got.GrossStake.Equal(decimal.NewFromInt(1000)) {
		t.Fatalf("GrossStake = %s, want 1000", got.GrossStake)
	}
}

func TestApplyStakeTax_UnknownCountry_ReturnsError(t *testing.T) {
	t.Parallel()
	e, err := tax.Default()
	if err != nil {
		t.Fatalf("Failed to create tax engine: %v", err)
	}

	_, err = e.ApplyStakeTax("ZZ", decimal.NewFromInt(500))
	if err == nil {
		t.Fatal("Expected error for unknown country, got nil")
	}
	if !errors.Is(err, tax.ErrUnknownCountry) {
		t.Fatalf("Expected ErrUnknownCountry, got %v", err)
	}
}

func TestApplyPayoutTax_Winnings_Kenya(t *testing.T) {
	t.Parallel()
	e, err := tax.Default()
	if err != nil {
		t.Fatalf("Failed to create tax engine: %v", err)
	}

	// Stake 1000, gross payout 3000 → winnings 2000 → WHT = 400, net = 2600
	got, err := e.ApplyPayoutTax("KE", decimal.NewFromInt(3000), decimal.NewFromInt(1000))
	if err != nil {
		t.Fatalf("ApplyPayoutTax failed: %v", err)
	}

	if !got.Winnings.Equal(decimal.NewFromInt(2000)) {
		t.Fatalf("Winnings = %s, want 2000", got.Winnings)
	}
	if !got.WinningsTax.Equal(decimal.NewFromInt(400)) {
		t.Fatalf("WinningsTax = %s, want 400", got.WinningsTax)
	}
	if !got.NetPayout.Equal(decimal.NewFromInt(2600)) {
		t.Fatalf("NetPayout = %s, want 2600", got.NetPayout)
	}
}

func TestApplyPayoutTax_NoWinnings_NoTax(t *testing.T) {
	t.Parallel()
	e, err := tax.Default()
	if err != nil {
		t.Fatalf("Failed to create tax engine: %v", err)
	}

	// Stake 1000, gross payout 500 (loss scenario) → winnings clamped to 0
	got, err := e.ApplyPayoutTax("KE", decimal.NewFromInt(500), decimal.NewFromInt(1000))
	if err != nil {
		t.Fatalf("ApplyPayoutTax failed: %v", err)
	}

	if !got.Winnings.IsZero() {
		t.Fatalf("Winnings = %s, want 0", got.Winnings)
	}
	if !got.WinningsTax.IsZero() {
		t.Fatalf("WinningsTax = %s, want 0", got.WinningsTax)
	}
	if !got.NetPayout.Equal(decimal.NewFromInt(500)) {
		t.Fatalf("NetPayout = %s, want 500", got.NetPayout)
	}
}

func TestApplyPayoutTax_WithThreshold(t *testing.T) {
	t.Parallel()
	// Custom regime with 1000 WHT threshold.
	e, err := tax.New(tax.Regime{
		CountryCode:       "KE",
		StakeTaxRate:      decimal.NewFromFloat(0.15),
		WinningsTaxRate:   decimal.NewFromFloat(0.20),
		WinningsThreshold: decimal.NewFromInt(1000),
		Currency:          "KES",
	})
	if err != nil {
		t.Fatalf("Failed to create tax engine: %v", err)
	}

	// Winnings = 1500 → taxable = 500 → tax = 100 → net = 2400
	got, err := e.ApplyPayoutTax("KE", decimal.NewFromInt(2500), decimal.NewFromInt(1000))
	if err != nil {
		t.Fatalf("ApplyPayoutTax failed: %v", err)
	}

	if !got.TaxableAmount.Equal(decimal.NewFromInt(500)) {
		t.Fatalf("TaxableAmount = %s, want 500", got.TaxableAmount)
	}
	if !got.WinningsTax.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("WinningsTax = %s, want 100", got.WinningsTax)
	}
	if !got.NetPayout.Equal(decimal.NewFromInt(2400)) {
		t.Fatalf("NetPayout = %s, want 2400", got.NetPayout)
	}
}
