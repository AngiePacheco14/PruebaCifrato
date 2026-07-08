# Cifrato — Motor de retenciones sobre facturas electrónicas

Servicio en Go que recibe facturas electrónicas colombianas (XML UBL DIAN) y calcula
automáticamente las retenciones que aplican — **ReteFuente**, **ReteIVA** y **ReteICA** —
devolviendo para cada una la base gravable, la tarifa aplicada, el valor y la norma legal
que la sustenta.

Prueba técnica para Cifrato. Contexto completo del enunciado en
[`contexto-prueba-cifrato.md`](contexto-prueba-cifrato.md).

## Enfoque

El cálculo se divide en dos responsabilidades bien separadas:

- **Motor de reglas determinístico** — funciones Go puras, sin IO. Busca la tarifa vigente
  para la fecha de la factura y calcula la retención. El monto **nunca** lo decide un LLM:
  en un contexto fiscal el resultado tiene que ser reproducible y auditable.
- **Clasificación por LLM (Claude)** — las facturas traen la descripción del producto/servicio
  en texto libre, sin un código de concepto fiscal. Mapear esa descripción a un concepto de
  retención (compra de bienes, servicio, transporte) es un problema de lenguaje natural, así
  que ahí sí se usa un LLM. Su salida nunca decide un monto, solo indica *a qué regla aplicar*
  — el cálculo lo sigue haciendo el motor determinístico. Para no reclasificar lo mismo dos
  veces, hay una caché por proveedor + descripción.

## Arquitectura

Arquitectura hexagonal (puertos y adaptadores): el dominio y los casos de uso no conocen
Postgres, HTTP ni el SDK de Anthropic — solo interfaces, que los adaptadores implementan.

```
cmd/cifrato/            entrypoint: CLI (serve / migrate)
internal/
  domain/                entidades y motor de cálculo puro (sin IO)
  application/            casos de uso: parsear, clasificar, calcular, orquestar
  infrastructure/
    rest/                  adaptador de entrada: HTTP (handlers, DTOs, rutas)
    adapters/
      xmlparser/             parsea el XML UBL DIAN
      api/anthropic/         clasifica líneas usando Claude
      repository/postgres/   persistencia (GORM + migraciones versionadas)
    dependence/            composición de dependencias
```

## Tecnologías

- **Go** como lenguaje principal.
- **Postgres + GORM** para persistencia.
- **golang-migrate** para versionar el esquema y los datos de referencia (tarifas, UVT,
  ciudades) como SQL, no como `AutoMigrate`.
- **Claude (Anthropic SDK)** para clasificar la descripción de cada línea.
- **dig** para inyección de dependencias.
- **shopspring/decimal** para todos los montos — nunca `float`, para evitar errores de
  redondeo en cifras fiscales.

## Cómo correrlo

Requisitos: Go 1.26+, Docker (para Postgres), y opcionalmente una API key de Anthropic
para clasificación real (sin ella, el pipeline sigue funcionando pero deja las líneas sin
concepto clasificado).

```bash
cp .env.example .env          # ajustar si es necesario
docker compose up -d          # levanta Postgres en localhost:5432

export ANTHROPIC_API_KEY=sk-ant-...   # opcional pero recomendado
go run ./cmd/cifrato serve            # levanta el servidor HTTP en :8080
```

`cifrato serve` migra el esquema automáticamente la primera vez que se conecta a Postgres.
Si preferís forzar la migración por separado (por ejemplo en CI), `go run ./cmd/cifrato
migrate` sigue disponible.

## Endpoints

### `POST /invoices` — procesa una factura

```bash
curl -X POST "http://localhost:8080/invoices?filename=P12206.xml" \
  --data-binary @sample-invoices/sample-1/2025-09-05_P12206_499c1c1fd58b39f6.xml \
  -H "Content-Type: application/xml"
```

Respuesta (`200 OK`):

```json
{
  "cufe": "499c1c1fd58b39f6...",
  "invoice_number": "P12206",
  "issuer_nit": "900290912",
  "issuer_name": "INSTINCTS HUMAINS SAS",
  "invoice_total": "52049172.00",
  "summary": {
    "total_retefuente": "1093470.00",
    "total_reteiva": "1246555.80",
    "total_reteica": "218694.00"
  },
  "calculations": [
    {
      "tax_type": "RETEFUENTE", "concept_id": 1,
      "base_amount": "43738800.00", "tariff_applied": "2.5", "calculated_value": "1093470.00",
      "legal_basis": "Art. 401 E.T.; Decreto 572 de 2025",
      "justification": "base gravable $43738800.00 supera/iguala el mínimo de 10 UVT ($523740.00); se aplica tarifa 2.5%"
    }
  ]
}
```

`calculations` trae un resultado por cada tipo de retención y concepto fiscal presente en
la factura (la base ya está sumada entre todas las líneas que comparten ese concepto, no es
el valor de una sola línea — el mínimo de UVT de ReteFuente/ReteICA se evalúa por factura, no
por ítem). `summary` es ese mismo detalle sumado por tipo de retención, para no tener que
sumarlo a mano.

### `POST /invoices/batch` — procesa varias facturas a la vez

```bash
curl -X POST "http://localhost:8080/invoices/batch" \
  -F "files=@factura1.xml" \
  -F "files=@factura2.xml"
```

`multipart/form-data` con un campo `files` repetido por cada XML. Se procesan en paralelo
(hasta 5 a la vez); una factura mal formada no aborta el resto del lote — cada archivo
reporta su propio éxito o error en la respuesta.

```json
{
  "results": [
    { "filename": "factura1.xml", "success": true, "invoice": { "...": "..." } },
    { "filename": "factura2.xml", "success": false, "error": "..." }
  ]
}
```

### Variables de entorno

| Variable | Default | Uso |
|---|---|---|
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE` | ver `.env.example` | conexión a Postgres |
| `RUN_MIGRATIONS` | `true` | si `false`, `cifrato serve` no migra automáticamente al arrancar |
| `VAT_WITHHOLDING_AGENT` | `false` | si la empresa compradora es agente de retención de IVA |
| `ANTHROPIC_API_KEY` | — | credencial para clasificar líneas con Claude; sin ella, la clasificación falla por línea sin bloquear la factura |
| `CLASSIFIER_MODEL` | `claude-haiku-4-5` | modelo usado para clasificar líneas |
| `HTTP_ADDR` | `:8080` | dirección de escucha del servidor HTTP |

## Limitaciones conocidas

- El valor de UVT y algunas tarifas (ReteICA por ciudad, transporte de carga) son
  representativos para poder procesar las facturas de muestra de punta a punta — no están
  verificados contra la norma vigente ni contra un contador. Cada regla lo indica en su
  `legal_basis` cuando aplica.
- El catálogo de conceptos fiscales es mínimo (compra de bienes, servicios generales,
  transporte de carga) — suficiente para las facturas de muestra, no exhaustivo.
- ReteICA siempre usa la ciudad del emisor/vendedor, nunca la del comprador.
- La búsqueda de ciudad hace comparación por nombre normalizado con contención (no exacta),
  porque el XML no siempre trae el nombre igual al de la tabla de ciudades sembrada. Puede
  dar falsos positivos entre municipios con nombres parecidos; no ocurre con las facturas de
  muestra.
- No hay autenticación en los endpoints — fuera de alcance de esta prueba.
- Los `WithholdingTaxTotal` que algunas facturas traen del proveedor son solo informativos;
  el motor calcula de forma independiente y no depende de ellos.

## Tests

```bash
go test ./...
```

Cubren el motor de cálculo puro, el parseo de las facturas de muestra reales, los casos de
uso con fakes en memoria, y (si hay Postgres/API key disponibles) la persistencia y la
clasificación real contra Claude.

`sample-invoices/` no se versiona en git (son datos reales de facturación de empresas
colombianas) — se agrega localmente para correr las pruebas contra facturas reales.
