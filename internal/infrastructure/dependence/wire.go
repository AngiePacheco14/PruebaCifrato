package dependence

import (
	"context"

	anthropicsdk "github.com/anthropics/anthropic-sdk-go"
	"go.uber.org/dig"
	"gorm.io/gorm"

	appconfig "cifrato/internal/application/config"
	"cifrato/internal/application/ports/in"
	"cifrato/internal/application/usecase"
	"cifrato/internal/domain/entity"
	"cifrato/internal/domain/repository"
	"cifrato/internal/infrastructure/adapters/api/anthropic"
	"cifrato/internal/infrastructure/adapters/repository/postgres"
	"cifrato/internal/infrastructure/adapters/xmlparser"
	"cifrato/internal/infrastructure/rest/handlers"
)

// NewWire builds the dependency graph (Postgres, the LLM classifier, the XML
// parser, and both use cases) as a dig.Container. Wiring mistakes surface at
// container.Invoke time rather than at build time.
func NewWire() *dig.Container {
	container := dig.New()

	container.Provide(postgres.ConfigFromEnv)
	container.Provide(postgres.OpenAndMigrate)

	container.Provide(func(db *gorm.DB) repository.InvoiceRepository {
		return postgres.NewInvoiceRepository(db)
	})
	container.Provide(func(db *gorm.DB) repository.ClassificationCacheRepository {
		return postgres.NewClassificationCacheRepository(db)
	})
	container.Provide(func(db *gorm.DB) repository.TaxRuleRepository {
		return postgres.NewTaxRuleRepository(db)
	})
	container.Provide(func(db *gorm.DB) repository.CalculationRepository {
		return postgres.NewCalculationRepository(db)
	})
	container.Provide(func(db *gorm.DB) repository.ReferenceDataRepository {
		return postgres.NewReferenceDataRepository(db)
	})

	// Reads ANTHROPIC_API_KEY from the environment.
	container.Provide(func() anthropicsdk.Client { return anthropicsdk.NewClient() })
	container.Provide(anthropic.ModelFromEnv)
	// Concept catalog is fetched once at startup.
	container.Provide(func(referenceData repository.ReferenceDataRepository) ([]entity.Concept, error) {
		return referenceData.ListConcepts(context.Background())
	})
	container.Provide(func(client anthropicsdk.Client, model string, concepts []entity.Concept) (repository.LineClassifier, error) {
		return anthropic.NewClassifier(client, model, concepts)
	})

	container.Provide(func() repository.InvoiceParser { return xmlparser.NewParser() })
	container.Provide(appconfig.FromEnv)

	container.Provide(func(cache repository.ClassificationCacheRepository, classifier repository.LineClassifier) in.ClassifyInvoiceLines {
		return usecase.NewClassifyInvoiceLines(cache, classifier)
	})
	container.Provide(func(taxRules repository.TaxRuleRepository, calculations repository.CalculationRepository, referenceData repository.ReferenceDataRepository, cfg appconfig.Config) in.CalculateWithholdings {
		return usecase.NewCalculateWithholdings(taxRules, calculations, referenceData, cfg)
	})
	container.Provide(func(parser repository.InvoiceParser, invoices repository.InvoiceRepository, classifyLines in.ClassifyInvoiceLines, calculateWithholdings in.CalculateWithholdings) in.ProcessInvoice {
		return usecase.NewProcessInvoice(parser, invoices, classifyLines, calculateWithholdings)
	})

	container.Provide(func(processInvoice in.ProcessInvoice) *handlers.InvoiceHandler {
		return handlers.NewInvoiceHandler(processInvoice)
	})

	return container
}
