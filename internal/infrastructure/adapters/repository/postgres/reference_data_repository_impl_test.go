package postgres

import "testing"

// TestNormalizeCityName_MatchesRealSampleInvoiceIssuerCity reproduces the
// exact mismatch found between the seeded reference cities ("BOGOTA D.C",
// plain form) and the free-text CityName issuers write into real invoice
// XML ("Bogotá, D.C.", accented, comma-punctuated) — before this fix,
// FindCityByName's exact string match silently never found Bogotá, so
// RETEICA never applied to any Bogotá-issued invoice in the sample set.
func TestNormalizeCityName_MatchesRealSampleInvoiceIssuerCity(t *testing.T) {
	cases := []struct {
		seeded string
		issuer string
	}{
		{"BOGOTA D.C", "Bogotá, D.C."},
		{"MEDELLIN", "Medellín"},
		{"GIRARDOTA", "Girardota"},
	}

	for _, c := range cases {
		got, want := normalizeCityName(c.issuer), normalizeCityName(c.seeded)
		if got != want {
			t.Errorf("normalizeCityName(%q) = %q, normalizeCityName(%q) = %q — want equal", c.issuer, got, c.seeded, want)
		}
	}
}

func TestNormalizeCityName_DistinctCitiesStayDistinct(t *testing.T) {
	if normalizeCityName("BOGOTA D.C") == normalizeCityName("MEDELLIN") {
		t.Error("normalizeCityName must not collapse distinct cities into the same key")
	}
}
