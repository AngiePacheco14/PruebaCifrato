package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"

	"cifrato/internal/domain/entity"
	"cifrato/internal/domain/repository"
)

const toolName = "classify_concept"

const systemPrompt = `Eres un clasificador experto en impuestos colombianos (DIAN). Tu única tarea es
leer la descripción de una línea de una factura de compra y clasificarla en uno
de los conceptos fiscales disponibles, usados para determinar la retención en
la fuente aplicable (RETEFUENTE, RETEIVA, RETEICA).

Responde SIEMPRE llamando la herramienta classify_concept con exactamente un
concepto de la lista dada. Si la descripción es ambigua, elige el concepto más
probable y refleja la incertidumbre en el campo "confidence" (no dejes de
responder por ambigüedad — el sistema no tiene una cola de revisión humana,
siempre necesita una clasificación).`

// Classifier implements repository.LineClassifier using the Anthropic
// Messages API with a forced tool call for structured output. The concept
// catalog is captured once at construction time — it does not change during
// the process lifetime, so it is not re-fetched per call.
type Classifier struct {
	client   anthropic.Client
	model    string
	concepts []entity.Concept
	byCode   map[string]entity.Concept
	tool     anthropic.ToolParam
}

// NewClassifier builds the adapter from an already-constructed Anthropic
// client, the model name, and the concept catalog (the caller fetches this
// once via ReferenceDataRepository.ListConcepts in the composition root).
// The tool schema is built once here, not per Classify call, since the
// catalog never changes during the process lifetime.
func NewClassifier(client anthropic.Client, model string, concepts []entity.Concept) (*Classifier, error) {
	if len(concepts) == 0 {
		return nil, fmt.Errorf("anthropic: concepts catalog is empty")
	}
	byCode := make(map[string]entity.Concept, len(concepts))
	for _, c := range concepts {
		byCode[c.Code] = c
	}
	c := &Classifier{client: client, model: model, concepts: concepts, byCode: byCode}
	c.tool = c.buildTool()
	return c, nil
}

var _ repository.LineClassifier = (*Classifier)(nil)

func (c *Classifier) Classify(ctx context.Context, description string) (*entity.LineClassification, error) {
	resp, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     c.model,
		MaxTokens: 1024,
		System:    []anthropic.TextBlockParam{{Text: systemPrompt}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(c.buildUserPrompt(description))),
		},
		Tools:      []anthropic.ToolUnionParam{{OfTool: &c.tool}},
		ToolChoice: anthropic.ToolChoiceParamOfTool(toolName),
	})
	if err != nil {
		// The SDK already retried 429/5xx internally (default max_retries=2);
		// anything reaching here is a real, exhausted failure — propagate.
		return nil, fmt.Errorf("anthropic: calling Anthropic API: %w", err)
	}

	for _, block := range resp.Content {
		if v, ok := block.AsAny().(anthropic.ToolUseBlock); ok {
			return c.parseToolUse(v)
		}
	}
	// tool_choice was forced, so this should not happen; if it does, it's
	// a contract error, not a transient one — propagate rather than degrade.
	return nil, fmt.Errorf("anthropic: response contained no tool_use block")
}

func (c *Classifier) parseToolUse(v anthropic.ToolUseBlock) (*entity.LineClassification, error) {
	var parsed struct {
		ConceptCode string  `json:"concept_code"`
		Confidence  float64 `json:"confidence"`
		Reasoning   string  `json:"reasoning"`
	}
	if err := json.Unmarshal(v.Input, &parsed); err != nil {
		return nil, fmt.Errorf("anthropic: parsing tool_use input: %w", err)
	}
	concept, ok := c.byCode[parsed.ConceptCode]
	if !ok {
		return nil, fmt.Errorf("anthropic: model returned unknown concept_code %q", parsed.ConceptCode)
	}
	return &entity.LineClassification{
		ConceptID:    concept.ID,
		ConceptCode:  concept.Code,
		Confidence:   parsed.Confidence,
		Reasoning:    parsed.Reasoning,
		ModelVersion: c.model,
	}, nil
}

// buildTool constructs the classify_concept tool schema dynamically from
// the concept catalog — the concept_code enum is never hardcoded, so a
// concept added to the database is picked up without a code change.
func (c *Classifier) buildTool() anthropic.ToolParam {
	codes := make([]string, len(c.concepts))
	for i, concept := range c.concepts {
		codes[i] = concept.Code
	}
	return anthropic.ToolParam{
		Name:        toolName,
		Description: anthropic.String("Clasifica la descripción de una línea de factura colombiana en uno de los conceptos fiscales disponibles para retención en la fuente."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{
				"concept_code": map[string]any{
					"type":        "string",
					"enum":        codes,
					"description": "El código del concepto fiscal que mejor clasifica la línea.",
				},
				"confidence": map[string]any{
					"type":        "number",
					"description": "Confianza en la clasificación, de 0.0 a 1.0.",
				},
				"reasoning": map[string]any{
					"type":        "string",
					"description": "Explicación breve (1-2 frases) de por qué se eligió este concepto.",
				},
			},
			Required: []string{"concept_code", "confidence", "reasoning"},
		},
	}
}

func (c *Classifier) buildUserPrompt(description string) string {
	var sb strings.Builder
	sb.WriteString("Conceptos fiscales disponibles:\n")
	for _, concept := range c.concepts {
		detail := ""
		if concept.Description != "" {
			detail = ": " + concept.Description
		}
		fmt.Fprintf(&sb, "- %s (%s)%s\n", concept.Code, concept.Name, detail)
	}
	fmt.Fprintf(&sb, "\nClasifica la siguiente línea de factura:\n%q\n", description)
	return sb.String()
}
