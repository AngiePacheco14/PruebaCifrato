package withholding_test

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"cifrato/internal/domain/withholding"
)

func dec(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return d
}

func TestCalculateWithMinimumBase(t *testing.T) {
	rule := withholding.TaxRule{
		TaxType:          withholding.TaxTypeRetefuente,
		MinBaseUVT:       dec("10"),
		TariffPercentage: dec("2.5"),
		LegalBasis:       "Art. 401 E.T.",
	}
	uvt := dec("52374") // min in pesos = 523740
	minPesos := dec("523740")

	t.Run("base above minimum applies tariff", func(t *testing.T) {
		base := dec("10000000")
		got := withholding.CalculateWithMinimumBase(rule, uvt, base)

		if got.CalculatedValue.String() != "250000" {
			t.Errorf("CalculatedValue = %s, want 250000", got.CalculatedValue)
		}
		if !got.TariffApplied.Equal(rule.TariffPercentage) {
			t.Errorf("TariffApplied = %s, want %s", got.TariffApplied, rule.TariffPercentage)
		}
		if !strings.Contains(got.Justification, "supera") {
			t.Errorf("Justification = %q, want it to mention 'supera'", got.Justification)
		}
	})

	t.Run("base below minimum yields zero", func(t *testing.T) {
		base := dec("100000")
		got := withholding.CalculateWithMinimumBase(rule, uvt, base)

		if !got.CalculatedValue.IsZero() {
			t.Errorf("CalculatedValue = %s, want 0", got.CalculatedValue)
		}
		if !got.TariffApplied.IsZero() {
			t.Errorf("TariffApplied = %s, want 0", got.TariffApplied)
		}
		if !strings.Contains(got.Justification, "no supera la base mínima") {
			t.Errorf("Justification = %q, want it to mention 'no supera la base mínima'", got.Justification)
		}
	})

	t.Run("base exactly equal to minimum applies (edge case, DIAN >= interpretation)", func(t *testing.T) {
		got := withholding.CalculateWithMinimumBase(rule, uvt, minPesos)

		if got.CalculatedValue.IsZero() {
			t.Error("CalculatedValue is zero, want tariff applied when base equals the minimum exactly")
		}
		want := minPesos.Mul(rule.TariffPercentage).Div(dec("100")).Round(2)
		if !got.CalculatedValue.Equal(want) {
			t.Errorf("CalculatedValue = %s, want %s", got.CalculatedValue, want)
		}
	})
}

func TestCalculateReteiva(t *testing.T) {
	rule := withholding.TaxRule{
		TaxType:          withholding.TaxTypeReteiva,
		TariffPercentage: dec("15"),
		LegalBasis:       "Art. 437-2 E.T.",
	}

	t.Run("iva above zero applies tariff over the iva amount, not the subtotal", func(t *testing.T) {
		got := withholding.CalculateReteiva(rule, dec("190000"))

		if got.CalculatedValue.String() != "28500" {
			t.Errorf("CalculatedValue = %s, want 28500 (15%% of 190000)", got.CalculatedValue)
		}
		if !got.BaseAmount.Equal(dec("190000")) {
			t.Errorf("BaseAmount = %s, want 190000 (the IVA value, not subtotal)", got.BaseAmount)
		}
	})

	t.Run("zero iva yields zero with no-iva justification", func(t *testing.T) {
		got := withholding.CalculateReteiva(rule, decimal.Zero)

		if !got.CalculatedValue.IsZero() {
			t.Errorf("CalculatedValue = %s, want 0", got.CalculatedValue)
		}
		if got.Justification != "no aplica: la línea no generó IVA" {
			t.Errorf("Justification = %q, want the no-IVA message", got.Justification)
		}
	})
}

func TestNotApplicable(t *testing.T) {
	got := withholding.NotApplicable(withholding.TaxTypeReteica, "sin tarifa ICA configurada para la ciudad CARTAGENA")

	if got.Justification != "sin tarifa ICA configurada para la ciudad CARTAGENA" {
		t.Errorf("Justification = %q, want the reason propagated verbatim", got.Justification)
	}
	if !got.CalculatedValue.IsZero() || !got.TariffApplied.IsZero() || !got.BaseAmount.IsZero() {
		t.Errorf("expected all amounts zero, got BaseAmount=%s TariffApplied=%s CalculatedValue=%s", got.BaseAmount, got.TariffApplied, got.CalculatedValue)
	}
	if got.TaxType != withholding.TaxTypeReteica {
		t.Errorf("TaxType = %s, want %s", got.TaxType, withholding.TaxTypeReteica)
	}
}
