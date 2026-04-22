package tax_test

import (
	"errors"
	"testing"

	"github.com/betting-platform/internal/core/usecase/tax"
	"github.com/shopspring/decimal"
)

func TestTaxSecurity_InvalidTaxRates(t *testing.T) {
	t.Parallel()

	// Test invalid stake tax rates (>= 1.0)
	testCases := []struct {
		name         string
		stakeRate    decimal.Decimal
		winningsRate decimal.Decimal
		expectError  bool
	}{
		{
			name:         "Valid rates",
			stakeRate:    decimal.NewFromFloat(0.15),
			winningsRate: decimal.NewFromFloat(0.20),
			expectError:  false,
		},
		{
			name:         "Invalid stake rate - exactly 1.0",
			stakeRate:    decimal.NewFromInt(1),
			winningsRate: decimal.NewFromFloat(0.20),
			expectError:  true,
		},
		{
			name:         "Invalid stake rate - greater than 1.0",
			stakeRate:    decimal.NewFromFloat(1.5),
			winningsRate: decimal.NewFromFloat(0.20),
			expectError:  true,
		},
		{
			name:         "Invalid winnings rate - exactly 1.0",
			stakeRate:    decimal.NewFromFloat(0.15),
			winningsRate: decimal.NewFromInt(1),
			expectError:  true,
		},
		{
			name:         "Invalid winnings rate - greater than 1.0",
			stakeRate:    decimal.NewFromFloat(0.15),
			winningsRate: decimal.NewFromFloat(1.5),
			expectError:  true,
		},
		{
			name:         "Negative stake rate",
			stakeRate:    decimal.NewFromFloat(-0.1),
			winningsRate: decimal.NewFromFloat(0.20),
			expectError:  true,
		},
		{
			name:         "Negative winnings rate",
			stakeRate:    decimal.NewFromFloat(0.15),
			winningsRate: decimal.NewFromFloat(-0.1),
			expectError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := tax.New(tax.Regime{
				CountryCode:       "TEST",
				StakeTaxRate:      tc.stakeRate,
				WinningsTaxRate:   tc.winningsRate,
				WinningsThreshold: decimal.Zero,
				Currency:          "TEST",
			})

			if tc.expectError {
				if err == nil {
					t.Fatal("Expected error for invalid tax rates, got nil")
				}
				if !tc.expectError {
					t.Fatalf("Unexpected error: %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error for valid rates: %v", err)
				}
			}
		})
	}
}

func TestTaxSecurity_NegativeThreshold(t *testing.T) {
	t.Parallel()

	// Test negative winnings threshold
	_, err := tax.New(tax.Regime{
		CountryCode:       "TEST",
		StakeTaxRate:      decimal.NewFromFloat(0.15),
		WinningsTaxRate:   decimal.NewFromFloat(0.20),
		WinningsThreshold: decimal.NewFromFloat(-100), // Invalid negative threshold
		Currency:          "TEST",
	})

	if err == nil {
		t.Fatal("Expected error for negative winnings threshold, got nil")
	}
}

func TestTaxSecurity_NegativeAmounts(t *testing.T) {
	t.Parallel()

	e, err := tax.Default()
	if err != nil {
		t.Fatalf("Failed to create tax engine: %v", err)
	}

	// Test negative stake amount
	_, err = e.ApplyStakeTax("KE", decimal.NewFromFloat(-100))
	if err == nil {
		t.Fatal("Expected error for negative stake amount, got nil")
	}
	if err != tax.ErrNegativeStake {
		t.Fatalf("Expected ErrNegativeStake, got %v", err)
	}

	// Test negative payout amount
	_, err = e.ApplyPayoutTax("KE", decimal.NewFromFloat(-100), decimal.NewFromInt(100))
	if err == nil {
		t.Fatal("Expected error for negative payout amount, got nil")
	}
	if err != tax.ErrNegativePayout {
		t.Fatalf("Expected ErrNegativePayout, got %v", err)
	}

	// Test negative stake in payout calculation
	_, err = e.ApplyPayoutTax("KE", decimal.NewFromInt(200), decimal.NewFromFloat(-100))
	if err == nil {
		t.Fatal("Expected error for negative stake amount in payout, got nil")
	}
	if err != tax.ErrNegativeStake {
		t.Fatalf("Expected ErrNegativeStake, got %v", err)
	}
}

func TestTaxSecurity_UnknownCountryError(t *testing.T) {
	t.Parallel()

	e, err := tax.Default()
	if err != nil {
		t.Fatalf("Failed to create tax engine: %v", err)
	}

	// Test unknown country codes
	unknownCountries := []string{"XX", "ZZ", "INVALID", "", "123"}

	for _, countryCode := range unknownCountries {
		t.Run("UnknownCountry_"+countryCode, func(t *testing.T) {
			t.Parallel()

			// Test stake tax
			_, err := e.ApplyStakeTax(countryCode, decimal.NewFromInt(100))
			if err == nil {
				t.Fatal("Expected error for unknown country, got nil")
			}
			if !errors.Is(err, tax.ErrUnknownCountry) {
				t.Fatalf("Expected ErrUnknownCountry, got %v", err)
			}

			// Test payout tax
			_, err = e.ApplyPayoutTax(countryCode, decimal.NewFromInt(200), decimal.NewFromInt(100))
			if err == nil {
				t.Fatal("Expected error for unknown country, got nil")
			}
			if !errors.Is(err, tax.ErrUnknownCountry) {
				t.Fatalf("Expected ErrUnknownCountry, got %v", err)
			}
		})
	}
}

func TestTaxSecurity_EdgeCaseValidation(t *testing.T) {
	t.Parallel()

	e, err := tax.Default()
	if err != nil {
		t.Fatalf("Failed to create tax engine: %v", err)
	}

	// Test zero amounts (should work)
	_, err = e.ApplyStakeTax("KE", decimal.Zero)
	if err != nil {
		t.Fatalf("ApplyStakeTax failed for zero amount: %v", err)
	}

	_, err = e.ApplyPayoutTax("KE", decimal.Zero, decimal.Zero)
	if err != nil {
		t.Fatalf("ApplyPayoutTax failed for zero amounts: %v", err)
	}

	// Test very small amounts (should work)
	smallAmount := decimal.NewFromFloat(0.01)
	_, err = e.ApplyStakeTax("KE", smallAmount)
	if err != nil {
		t.Fatalf("ApplyStakeTax failed for small amount: %v", err)
	}

	// Test very large amounts (should work)
	largeAmount := decimal.NewFromInt(1000000)
	_, err = e.ApplyStakeTax("KE", largeAmount)
	if err != nil {
		t.Fatalf("ApplyStakeTax failed for large amount: %v", err)
	}
}

func TestTaxSecurity_NetStakeValidation(t *testing.T) {
	t.Parallel()

	// Create a regime with very high tax rate to test net stake validation
	_, err := tax.New(tax.Regime{
		CountryCode:       "TEST",
		StakeTaxRate:      decimal.NewFromFloat(0.99), // 99% tax rate
		WinningsTaxRate:   decimal.NewFromFloat(0.20),
		WinningsThreshold: decimal.Zero,
		Currency:          "TEST",
	})
	if err != nil {
		t.Fatalf("Failed to create tax engine with high rate: %v", err)
	}

	// This should work - net stake would be 1% of gross
	e, err := tax.New(tax.Regime{
		CountryCode:       "TEST",
		StakeTaxRate:      decimal.NewFromFloat(0.99),
		WinningsTaxRate:   decimal.NewFromFloat(0.20),
		WinningsThreshold: decimal.Zero,
		Currency:          "TEST",
	})
	if err != nil {
		t.Fatalf("Failed to create tax engine: %v", err)
	}

	// Test with high tax rate - should still work but net stake is very small
	result, err := e.ApplyStakeTax("TEST", decimal.NewFromInt(100))
	if err != nil {
		t.Fatalf("ApplyStakeTax failed with high tax rate: %v", err)
	}

	// Net stake should be 1 (100 - 99)
	expectedNetStake := decimal.NewFromInt(1)
	if !result.NetStake.Equal(expectedNetStake) {
		t.Fatalf("Expected net stake %s, got %s", expectedNetStake, result.NetStake)
	}

	// Tax should be 99
	expectedTax := decimal.NewFromInt(99)
	if !result.StakeTax.Equal(expectedTax) {
		t.Fatalf("Expected tax %s, got %s", expectedTax, result.StakeTax)
	}
}
