// Package tax implements the Kenyan gambling tax regime for our betting platform.
//
//   - Excise duty / GGR: 15% of every stake (collected at bet placement).
//   - Withholding tax (WHT): 20% of winnings above a per-bet threshold,
//     deducted before payout.
//
// All math uses shopspring/decimal to avoid float drift on monetary values.
// The engine is country-aware so the same code path can serve NG/GH later.
package tax

import "github.com/shopspring/decimal"

// Regime is the tax configuration for a single country.
type Regime struct {
	// CountryCode is the ISO-3166 alpha-2 code this regime applies to.
	CountryCode string
	// StakeTaxRate is the fraction of each stake collected as excise / GGR (e.g. 0.15).
	StakeTaxRate decimal.Decimal
	// WinningsTaxRate is the fraction of winnings over the threshold withheld (e.g. 0.20).
	WinningsTaxRate decimal.Decimal
	// WinningsThreshold is the amount above which WHT applies (0 = all winnings).
	WinningsThreshold decimal.Decimal
	// Currency is the ISO-4217 code these amounts are denominated in.
	Currency string
}

// StakeBreakdown is what the player actually wagers after stake tax is withheld.
type StakeBreakdown struct {
	GrossStake decimal.Decimal `json:"gross_stake"`
	StakeTax   decimal.Decimal `json:"stake_tax"`
	NetStake   decimal.Decimal `json:"net_stake"`
}

// PayoutBreakdown is what the player receives after winnings tax is withheld.
type PayoutBreakdown struct {
	GrossWinnings decimal.Decimal `json:"gross_winnings"`
	Winnings      decimal.Decimal `json:"winnings"` // payout - original stake
	TaxableAmount decimal.Decimal `json:"taxable_amount"`
	WinningsTax   decimal.Decimal `json:"winnings_tax"`
	NetPayout     decimal.Decimal `json:"net_payout"`
}

// Engine applies tax regimes looked up by country code.
type Engine struct {
	regimes map[string]Regime
}

// New creates an Engine preloaded with the given regimes.
func New(regimes ...Regime) *Engine {
	m := make(map[string]Regime, len(regimes))
	for _, r := range regimes {
		m[r.CountryCode] = r
	}
	return &Engine{regimes: m}
}

// Default returns the Kenya-tuned engine used at launch.
//
// Rates are per Kenya Finance Act 2023 and subsequent amendments:
// 15% excise on stake, 20% WHT on winnings with no de-minimis threshold.
func Default() *Engine {
	return New(Regime{
		CountryCode:       "KE",
		StakeTaxRate:      decimal.NewFromFloat(0.15),
		WinningsTaxRate:   decimal.NewFromFloat(0.20),
		WinningsThreshold: decimal.Zero,
		Currency:          "KES",
	})
}

// Regime looks up the regime for a country. The second return is false when
// the country is not configured.
func (e *Engine) Regime(countryCode string) (Regime, bool) {
	r, ok := e.regimes[countryCode]
	return r, ok
}

// ApplyStakeTax withholds the stake tax and returns how much actually funds the bet.
func (e *Engine) ApplyStakeTax(countryCode string, gross decimal.Decimal) StakeBreakdown {
	r, ok := e.regimes[countryCode]
	if !ok {
		return StakeBreakdown{GrossStake: gross, NetStake: gross}
	}
	tax := gross.Mul(r.StakeTaxRate).Round(2)
	return StakeBreakdown{
		GrossStake: gross,
		StakeTax:   tax,
		NetStake:   gross.Sub(tax),
	}
}

// ApplyPayoutTax withholds WHT on the winnings portion of a payout. The input
// `grossPayout` is the total return (stake + winnings); `stake` is the
// original (gross) stake that produced it. Only the winnings delta above the
// configured threshold is taxed.
func (e *Engine) ApplyPayoutTax(countryCode string, grossPayout, stake decimal.Decimal) PayoutBreakdown {
	r, ok := e.regimes[countryCode]
	winnings := grossPayout.Sub(stake)
	if winnings.IsNegative() {
		winnings = decimal.Zero
	}

	b := PayoutBreakdown{
		GrossWinnings: grossPayout,
		Winnings:      winnings,
		NetPayout:     grossPayout,
	}
	if !ok || winnings.IsZero() {
		return b
	}

	taxable := winnings.Sub(r.WinningsThreshold)
	if taxable.IsNegative() {
		taxable = decimal.Zero
	}
	tax := taxable.Mul(r.WinningsTaxRate).Round(2)

	b.TaxableAmount = taxable
	b.WinningsTax = tax
	b.NetPayout = grossPayout.Sub(tax)
	return b
}
