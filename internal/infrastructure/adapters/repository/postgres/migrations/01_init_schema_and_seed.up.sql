-- Esquema completo de Cifrato + datos de referencia mínimos para poder
-- calcular retenciones de punta a punta (ciudades, conceptos, UVT y
-- tarifas RETEFUENTE/RETEIVA/RETEICA). Sin estos datos el motor de cálculo
-- no tiene ninguna regla que aplicar.
--
-- Vigencias en 2025-01-01: las 10 facturas de muestra van de 2025-08 a
-- 2026-04; usar una sola vigencia (con el valor UVT 2026, $52.374) para
-- todo ese rango es una simplificación deliberada de esta prueba técnica —
-- ver README.md ("Limitaciones conocidas") antes de usar en producción.
--
-- Las tarifas RETEICA (las 3 ciudades) y RETEFUENTE de transporte_carga son
-- representativas, NO verificadas contra el Acuerdo Municipal vigente ni
-- contra un contador — ver el texto de cada legal_basis abajo.

-- ── Tablas de referencia (sin dependencias) ────────────────────────────────

CREATE TABLE cities (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    department  VARCHAR(100) NOT NULL,
    created_at  TIMESTAMPTZ,
    updated_at  TIMESTAMPTZ,
    CONSTRAINT idx_city_name_dept UNIQUE (name, department)
);

CREATE TABLE withholding_concepts (
    id          BIGSERIAL PRIMARY KEY,
    code        VARCHAR(50) NOT NULL UNIQUE,
    name        VARCHAR(150) NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ,
    updated_at  TIMESTAMPTZ
);

CREATE TABLE uvt_values (
    id                    BIGSERIAL PRIMARY KEY,
    year                  BIGINT NOT NULL UNIQUE,
    value                 NUMERIC(12,2) NOT NULL,
    effective_from        TIMESTAMPTZ NOT NULL,
    effective_to          TIMESTAMPTZ,
    resolution_reference  TEXT,
    created_at            TIMESTAMPTZ,
    updated_at            TIMESTAMPTZ
);
CREATE INDEX idx_uvt_values_effective_from ON uvt_values (effective_from);
CREATE INDEX idx_uvt_values_effective_to ON uvt_values (effective_to);

-- ── Facturas ────────────────────────────────────────────────────────────────

CREATE TABLE invoices (
    id                          BIGSERIAL PRIMARY KEY,
    cufe                        VARCHAR(100) NOT NULL UNIQUE,
    invoice_number              VARCHAR(50) NOT NULL,
    issue_date                  TIMESTAMPTZ NOT NULL,
    xml_type                    VARCHAR(20) NOT NULL,
    issuer_nit                  VARCHAR(20) NOT NULL,
    issuer_name                 VARCHAR(255) NOT NULL,
    issuer_city                 VARCHAR(100),
    issuer_tax_responsibility   VARCHAR(255),
    buyer_nit                   VARCHAR(20) NOT NULL,
    buyer_name                  VARCHAR(255) NOT NULL,
    subtotal                    NUMERIC(18,2) NOT NULL,
    iva_total                   NUMERIC(18,2) NOT NULL,
    invoice_total               NUMERIC(18,2) NOT NULL,
    source_xml_path             VARCHAR(500),
    source_pdf_path             VARCHAR(500),
    reported_retefuente         NUMERIC(18,2),
    reported_reteiva            NUMERIC(18,2),
    reported_reteica            NUMERIC(18,2),
    created_at                  TIMESTAMPTZ,
    updated_at                  TIMESTAMPTZ
);
CREATE INDEX idx_invoices_invoice_number ON invoices (invoice_number);
CREATE INDEX idx_invoices_issue_date ON invoices (issue_date);
CREATE INDEX idx_invoices_issuer_nit ON invoices (issuer_nit);
CREATE INDEX idx_invoices_buyer_nit ON invoices (buyer_nit);

CREATE TABLE invoice_lines (
    id                          BIGSERIAL PRIMARY KEY,
    invoice_id                  BIGINT NOT NULL REFERENCES invoices (id) ON DELETE CASCADE,
    line_number                 BIGINT NOT NULL,
    sku                         VARCHAR(100),
    description                 TEXT NOT NULL,
    quantity                    NUMERIC(18,4) NOT NULL,
    unit_price                  NUMERIC(18,2) NOT NULL,
    line_total                  NUMERIC(18,2) NOT NULL,
    iva_rate                    NUMERIC(5,2) NOT NULL DEFAULT 0,
    iva_value                   NUMERIC(18,2) NOT NULL DEFAULT 0,
    concept_id                  BIGINT REFERENCES withholding_concepts (id),
    classification_confidence   DOUBLE PRECISION,
    created_at                  TIMESTAMPTZ,
    updated_at                  TIMESTAMPTZ,
    CONSTRAINT idx_invoice_lines_invoice_line_number UNIQUE (invoice_id, line_number)
);
CREATE INDEX idx_invoice_lines_invoice_id ON invoice_lines (invoice_id);
CREATE INDEX idx_invoice_lines_sku ON invoice_lines (sku);
CREATE INDEX idx_invoice_lines_concept_id ON invoice_lines (concept_id);

-- ── Tarifas y clasificación ─────────────────────────────────────────────────

-- Unifica RETEFUENTE/RETEIVA (nacional, city_id NULL) y RETEICA (territorial,
-- city_id apunta al municipio) en una sola tabla versionada por vigencia.
CREATE TABLE additional_taxes_rules (
    id                  BIGSERIAL PRIMARY KEY,
    tax_type            VARCHAR(20) NOT NULL,
    concept_id          BIGINT NOT NULL REFERENCES withholding_concepts (id),
    city_id             BIGINT REFERENCES cities (id),
    min_base_uvt        NUMERIC(10,2) NOT NULL DEFAULT 0,
    tariff_percentage   NUMERIC(7,4) NOT NULL,
    legal_basis         TEXT,
    effective_from      TIMESTAMPTZ NOT NULL,
    effective_to        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ,
    updated_at          TIMESTAMPTZ
);
CREATE INDEX idx_rule_lookup ON additional_taxes_rules (tax_type, concept_id, city_id, effective_from);
CREATE INDEX idx_additional_taxes_rules_effective_to ON additional_taxes_rules (effective_to);

-- Caché de clasificación por LLM autoalimentada (no un catálogo de keywords
-- curado a mano). El uniqueIndex (issuer_nit, sku) excluye de forma natural
-- las filas donde cualquiera de las dos sea NULL (semántica estándar de
-- Postgres) — por eso no hace falta un índice parcial.
CREATE TABLE line_classifications (
    id                       BIGSERIAL PRIMARY KEY,
    issuer_nit               VARCHAR(20),
    sku                      VARCHAR(100),
    description_normalized   VARCHAR(500),
    concept_id               BIGINT NOT NULL REFERENCES withholding_concepts (id),
    confidence               DOUBLE PRECISION NOT NULL,
    model_version            VARCHAR(50) NOT NULL,
    reasoning                TEXT,
    created_at               TIMESTAMPTZ,
    CONSTRAINT idx_issuer_sku UNIQUE (issuer_nit, sku)
);
CREATE INDEX idx_line_classifications_description_normalized ON line_classifications (description_normalized);
CREATE INDEX idx_line_classifications_concept_id ON line_classifications (concept_id);

-- One row per (invoice, concept, tax_type) — the tariff/minimum-base check
-- is evaluated on the base amount aggregated across every line of that
-- concept in the invoice, not per line (RETEFUENTE/RETEICA's minimum base
-- is a per-payment threshold, not a per-item one — DIAN evaluates it over
-- what's owed to a provider for the transaction, not each line separately).
-- concept_id has no FK (matches withholding_concepts' own choice here): 0 is
-- the "lines with no classified concept" bucket, never a real concept row,
-- so it can't collide with an actual concept ID. Recalculating overwrites,
-- no history is kept.
CREATE TABLE withholding_calculations (
    id                  BIGSERIAL PRIMARY KEY,
    invoice_id          BIGINT NOT NULL,
    tax_type            VARCHAR(20) NOT NULL,
    concept_id          BIGINT NOT NULL,
    base_amount         NUMERIC(18,2) NOT NULL,
    tariff_applied      NUMERIC(7,4) NOT NULL,
    calculated_value    NUMERIC(18,2) NOT NULL,
    legal_basis         TEXT,
    justification       TEXT,
    created_at          TIMESTAMPTZ,
    updated_at          TIMESTAMPTZ,
    CONSTRAINT idx_invoice_concept_taxtype UNIQUE (invoice_id, concept_id, tax_type)
);
-- No separate invoice_id index: the unique constraint above already leads
-- with invoice_id, so Postgres can use it for invoice_id-only lookups too.
CREATE INDEX idx_withholding_calculations_concept_id ON withholding_calculations (concept_id);

-- ── Datos de referencia (mínimos para poder correr las retenciones) ────────

INSERT INTO cities (name, department) VALUES
    ('BOGOTA D.C', 'Bogotá D.C.'),
    ('MEDELLIN', 'Antioquia'),
    ('GIRARDOTA', 'Antioquia');

INSERT INTO withholding_concepts (code, name) VALUES
    ('compra_bienes', 'Compra de bienes'),
    ('servicios_generales', 'Servicios generales'),
    ('transporte_carga', 'Transporte de carga');

INSERT INTO uvt_values (year, value, effective_from, resolution_reference) VALUES
    (2026, 52374.00, '2025-01-01', 'Resolución DIAN 000238 del 15 de diciembre de 2025');

-- RETEFUENTE nacional — confirmadas en el contexto de la prueba salvo
-- transporte_carga (aplicación análoga, no confirmada en la fuente).
INSERT INTO additional_taxes_rules (tax_type, concept_id, city_id, min_base_uvt, tariff_percentage, legal_basis, effective_from)
SELECT 'RETEFUENTE', id, NULL, 10, 2.5, 'Art. 401 E.T.; Decreto 572 de 2025', '2025-01-01'
FROM withholding_concepts WHERE code = 'compra_bienes';

INSERT INTO additional_taxes_rules (tax_type, concept_id, city_id, min_base_uvt, tariff_percentage, legal_basis, effective_from)
SELECT 'RETEFUENTE', id, NULL, 2, 4, 'Art. 392 E.T.; Decreto 572 de 2025', '2025-01-01'
FROM withholding_concepts WHERE code = 'servicios_generales';

INSERT INTO additional_taxes_rules (tax_type, concept_id, city_id, min_base_uvt, tariff_percentage, legal_basis, effective_from)
SELECT 'RETEFUENTE', id, NULL, 4, 1, 'Art. 401 E.T. (aplicación análoga) — NO CONFIRMADO EN LA FUENTE, verificar con contador antes de producción', '2025-01-01'
FROM withholding_concepts WHERE code = 'transporte_carga';

-- RETEIVA nacional — sin mínimo por decisión de negocio, 15% confirmado.
INSERT INTO additional_taxes_rules (tax_type, concept_id, city_id, min_base_uvt, tariff_percentage, legal_basis, effective_from)
SELECT 'RETEIVA', id, NULL, 0, 15, 'Art. 437-2 E.T.', '2025-01-01'
FROM withholding_concepts WHERE code IN ('compra_bienes', 'servicios_generales', 'transporte_carga');

-- RETEICA — mismas tarifas de ejemplo en las 3 ciudades sembradas: no hay
-- dato real en la fuente que las diferencie por municipio.
INSERT INTO additional_taxes_rules (tax_type, concept_id, city_id, min_base_uvt, tariff_percentage, legal_basis, effective_from)
SELECT 'RETEICA', c.id, ci.id, 0, v.tariff,
       'Ley 14 de 1983; Decreto 1333 de 1986; Ley 1819 de 2016 art. 342 — TARIFA DE EJEMPLO PARA PRUEBAS, NO VERIFICADA CONTRA EL ACUERDO MUNICIPAL VIGENTE',
       '2025-01-01'
FROM withholding_concepts c
JOIN (VALUES ('compra_bienes', 0.5), ('servicios_generales', 0.7), ('transporte_carga', 0.6)) AS v(code, tariff)
    ON v.code = c.code
CROSS JOIN cities ci
WHERE ci.name IN ('BOGOTA D.C', 'MEDELLIN', 'GIRARDOTA');
