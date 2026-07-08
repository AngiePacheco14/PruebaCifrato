package postgres

import (
	"context"
	"os"
	"testing"
)

// TestNormalizeCityName_MatchesRealSampleInvoiceIssuerCity checks that a
// seeded city name ("BOGOTA D.C") and a free-text issuer city ("Bogotá, D.C.")
// normalize to the same key.
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

// TestFindCityByName_MatchesBareNameWithoutSuffix checks that a bare
// "BOGOTÁ" (no "D.C." suffix) still matches via the substring match.
func TestFindCityByName_MatchesBareNameWithoutSuffix(t *testing.T) {
	if os.Getenv("DB_HOST") == "" {
		t.Skip("DB_HOST not set, skipping integration test (requires docker-compose postgres)")
	}

	db, err := Open(ConfigFromEnv())
	if err != nil {
		t.Fatalf("opening connection: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("migrating: %v", err)
	}

	repo := NewReferenceDataRepository(db)
	ctx := context.Background()

	got, err := repo.FindCityByName(ctx, "BOGOTÁ")
	if err != nil {
		t.Fatalf("FindCityByName() error = %v", err)
	}
	if got == nil {
		t.Fatal("FindCityByName(\"BOGOTÁ\") = nil, want the seeded Bogotá D.C. row")
	}
	if got.Name != "BOGOTA D.C" {
		t.Errorf("matched city = %q, want \"BOGOTA D.C\"", got.Name)
	}
}

// TestFindCityByName_EmptyNameNeverMatches checks that a blank name doesn't
// match every row (strings.Contains(x, "") is always true).
func TestFindCityByName_EmptyNameNeverMatches(t *testing.T) {
	if os.Getenv("DB_HOST") == "" {
		t.Skip("DB_HOST not set, skipping integration test (requires docker-compose postgres)")
	}

	db, err := Open(ConfigFromEnv())
	if err != nil {
		t.Fatalf("opening connection: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("migrating: %v", err)
	}

	repo := NewReferenceDataRepository(db)
	ctx := context.Background()

	got, err := repo.FindCityByName(ctx, "   ")
	if err != nil {
		t.Fatalf("FindCityByName() error = %v", err)
	}
	if got != nil {
		t.Errorf("FindCityByName of a blank name = %+v, want nil", got)
	}
}
