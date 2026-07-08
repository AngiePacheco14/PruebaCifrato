# Cifrato — Motor de retenciones sobre facturas de compra

Servicio en Go que recibe facturas electrónicas colombianas (UBL DIAN) y calcula
automáticamente las retenciones aplicables — **ReteFuente**, **ReteIVA**, **ReteICA** —
justificando cómo llegó a cada resultado (base gravable, tarifa, norma legal).

Prueba técnica para Cifrato. Contexto completo del enunciado en
[`contexto-prueba-cifrato.md`](contexto-prueba-cifrato.md).

## Enfoque

El cálculo de retenciones es un problema fiscal: el monto debe ser **reproducible y
auditable**. Por eso el proyecto separa tajantemente dos responsabilidades:

- **Motor de reglas determinístico** (`internal/domain/service`): puras funciones Go
  sin IO ni dependencias externas. Nunca delega el cálculo a un LLM — un modelo puede
  alucinar una tarifa o una UVT vieja, y eso no es aceptable en un contexto fiscal. Cada
  retención calculada devuelve `{tipo, base, tarifa, valor, justificación, norma legal}`.
- **Clasificación de conceptos por LLM** (`internal/infrastructure/adapters/api/anthropic`):
  las facturas de muestra no traen código UNSPSC, solo descripción libre en texto — mapear
  esa descripción a un concepto de retención (compra de bienes, servicio, transporte) es un
  problema de lenguaje natural, no de reglas. Aquí sí se usa un LLM (Claude), pero su output
  nunca decide un monto, solo *a cuál regla determinística* aplicar.

Esto se traduce en arquitectura hexagonal (puertos y adaptadores): el dominio y los casos de
uso no conocen Postgres, HTTP ni el SDK de Anthropic — solo interfaces (`ports/in`,
`domain/repository`) que los adaptadores implementan. La convención de carpetas/nombres de
archivo sigue la de otro proyecto Go de la casa (`bia-electronic-bills`), para tener un mismo
"idioma" de arquitectura entre proyectos:

```
cmd/cifrato/                                  entrypoint: CLI (migrate/serve)
internal/
  domain/
    entity/                                   entidades de dominio (Invoice, Calculation, TaxRule, ...)
    enums/                                     XMLType, TaxType
    service/                                   motor de cálculo puro (sin IO)
    repository/                                puertos de salida: interfaces hacia BD/LLM/parser
  application/
    ports/in/                                  casos de uso, vistos desde afuera (interfaces + DTOs)
    usecase/                                   orquestación: ProcessInvoice, CalculateWithholdings, ClassifyInvoiceLines
    config/                                    configuración de negocio (ej. agente retenedor de IVA)
  infrastructure/
    rest/
      handlers/                                adaptador de entrada: POST /invoices
      dto/                                      request/response JSON
      router/                                   registro de rutas
    adapters/
      xmlparser/                                parsea UBL DIAN (Invoice y AttachedDocument/CDATA)
      api/anthropic/                            clasifica líneas con Claude (Anthropic SDK)
      repository/postgres/                      persistencia (GORM)
        model/                                  structs GORM
        mappers/                                dominio ↔ modelo GORM
        migrations/                             esquema + datos de referencia versionados (golang-migrate)
    dependence/                                 composición: dig (grafo de dependencias)
```

Convención de archivos: cada interfaz de puerto vive en su propio archivo, y sus DTOs en un
archivo hermano (`*_dto.go`) — ninguna interfaz mezcla su propio tipo de datos. Las entidades
de dominio siguen el mismo principio en `*_entity.go`, los modelos de persistencia en
`*_model.go`, y las implementaciones de repositorio en `*_repository_impl.go`.

Una diferencia deliberada frente a `bia-electronic-bills`: ahí no existe un `ports/in` (los
handlers llaman directo al struct concreto del caso de uso). Cifrato sí lo mantiene — es una
garantía real de la arquitectura hexagonal (el container de `dig` cablea sobre esa interfaz,
no sobre el struct concreto), y perderla sería un downgrade solo por igualar el nombre de una
carpeta.

## Cómo correrlo

Requisitos: Go 1.26+, Docker (Postgres), y opcionalmente una API key de Anthropic para
clasificación real (sin ella, el pipeline sigue funcionando pero deja las líneas sin
concepto clasificado — ver [Clasificación sin cola de revisión humana](#clasificación-sin-cola-de-revisión-humana)).

```bash
cp .env.example .env          # ajustar si es necesario
docker compose up -d          # levanta Postgres en localhost:5432

go run ./cmd/cifrato migrate  # crea el esquema Y carga los datos de referencia
                               # (ciudades, conceptos, UVT, tarifas) — ver Migraciones abajo

export ANTHROPIC_API_KEY=sk-ant-...   # opcional pero recomendado
go run ./cmd/cifrato serve            # levanta el servidor HTTP en :8080
```

Procesar una factura de muestra:

```bash
curl -X POST "http://localhost:8080/invoices?filename=P12206.xml" \
  --data-binary @sample-invoices/sample-1/2025-09-05_P12206_499c1c1fd58b39f6.xml \
  -H "Content-Type: application/xml"
```

Respuesta (`200 OK`, ejemplo real con esta factura de muestra, línea clasificada como
`compra_bienes`):

```json
{
  "cufe": "499c1c1fd58b39f63d5e8f7820426441757a612f95a2e20dc68e61f8dc19b3d95fc71d853d2acad7ab71f05109a7eb78",
  "invoice_number": "P12206",
  "issuer_nit": "900290912",
  "issuer_name": "INSTINCTS HUMAINS SAS",
  "invoice_total": "52049172.00",
  "calculations": [
    {
      "invoice_line_id": 3, "tax_type": "RETEFUENTE", "concept_id": 1,
      "base_amount": "43738800.00", "tariff_applied": "2.5", "calculated_value": "1093470.00",
      "legal_basis": "Art. 401 E.T.; Decreto 572 de 2025",
      "justification": "base gravable $43738800.00 supera/iguala el mínimo de 10 UVT ($523740.00); se aplica tarifa 2.5%"
    },
    {
      "invoice_line_id": 3, "tax_type": "RETEIVA", "concept_id": 1,
      "base_amount": "8310372.00", "tariff_applied": "15", "calculated_value": "1246555.80",
      "legal_basis": "Art. 437-2 E.T.",
      "justification": "tarifa 15% sobre IVA generado $8310372.00"
    },
    {
      "invoice_line_id": 3, "tax_type": "RETEICA", "concept_id": 1,
      "base_amount": "43738800.00", "tariff_applied": "0.5", "calculated_value": "218694.00",
      "legal_basis": "Ley 14 de 1983; Decreto 1333 de 1986; Ley 1819 de 2016 art. 342 — TARIFA DE EJEMPLO PARA PRUEBAS, NO VERIFICADA CONTRA EL ACUERDO MUNICIPAL VIGENTE",
      "justification": "base gravable $43738800.00 supera/iguala el mínimo de 0 UVT ($0.00); se aplica tarifa 0.5%"
    }
  ]
}
```

Nótese que el RETEFUENTE (`$1.093.470`) y el RETEIVA (`$1.246.555,80`) calculados de forma
independiente **coinciden exactamente** con los `WithholdingTaxTotal` que el propio emisor
reportó en el XML de esta factura — una validación cruzada útil, aunque el motor nunca lee ni
depende de esos valores para calcular.

Cada línea de la factura genera hasta 3 registros de cálculo (uno por tipo de retención),
persistidos en `withholding_calculations` y devueltos en la respuesta. Errores de
parseo/negocio responden `4xx` con `{"error": "..."}`.

### Variables de entorno

| Variable | Default | Uso |
|---|---|---|
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE` | ver `.env.example` | conexión a Postgres |
| `VAT_WITHHOLDING_AGENT` | `false` | si la empresa compradora es agente de retención de IVA (a nivel de compañía, no por factura) |
| `ANTHROPIC_API_KEY` | — | credencial del SDK de Anthropic; sin ella, la clasificación LLM falla por línea (no bloquea la factura) |
| `CLASSIFIER_MODEL` | `claude-haiku-4-5` | modelo usado para clasificar líneas — tarea simple, no justifica un modelo más caro |
| `HTTP_ADDR` | `:8080` | dirección de escucha del servidor HTTP |

## Decisiones de diseño y supuestos

### Versionado de tarifas
Las tarifas de ReteFuente/ReteIVA/ReteICA **no viven hardcodeadas**: la tabla
`additional_taxes_rules` versiona cada tarifa por `effective_from`/`effective_to`, y el motor
busca la regla vigente a la fecha de emisión de cada factura (`TaxRuleRepository.FindApplicable`).
Esto es exactamente lo que exige el contexto de la prueba: las bases de ReteFuente por
compras/servicios cambiaron varias veces en 2026 por el litigio del Decreto 572 de 2025, y una
tabla versionada permite cargar una nueva vigencia sin tocar código. Lo mismo aplica a la UVT
(`uvt_values`), también versionada por año/fecha de vigencia.

### `additional_taxes_rules` unifica nacional y territorial
ReteFuente/ReteIVA son de alcance nacional; ReteICA es territorial (cada municipio fija su
propia tarifa). En vez de dos tablas separadas, `additional_taxes_rules` tiene una columna
`city_id` **nullable**: `NULL` significa regla nacional, un valor significa tarifa municipal.
Evita una tabla paralela casi idéntica y una columna booleana redundante.

### Clasificación sin cola de revisión humana
El contexto de la prueba sugiere "marcar revisión humana si la confianza es baja" para el
clasificador. Se decidió **no implementar esa cola**: el caso de uso de Cifrato es procesar
muchas facturas de forma directa, y una cola de revisión humana no escala con ese volumen. En
su lugar:

- Caché híbrida (`line_classifications`): busca primero por `(issuer_nit, sku)`, luego por
  descripción normalizada. Si hay acierto, no se llama al LLM — así el costo y la latencia
  bajan con cada factura nueva del mismo proveedor.
- Si no hay acierto en caché, se llama a Claude con tool-use forzado (siempre debe responder
  con una clasificación, reflejando la incertidumbre en `confidence`, nunca absteniéndose).
- Si la llamada al LLM falla (red, rate limit, credenciales), esa línea queda sin concepto
  clasificado — no bloquea el resto de la factura. El motor de retenciones ya maneja ese caso
  con gracia (`"no aplica: línea sin concepto clasificado"`), el mismo camino que una línea
  nunca clasificada. El fallo se loggea para observabilidad, pero el pipeline sigue.

### Migraciones SQL versionadas (golang-migrate), no AutoMigrate
El esquema vive en `internal/infrastructure/adapters/repository/postgres/migrations/` como SQL
real (`.up.sql`/`.down.sql`), no como `gorm.AutoMigrate` sobre los structs de `model/`. GORM
sigue siendo el ORM para leer/escribir en runtime, pero deja de ser la fuente de verdad del
esquema — eso ahora es código versionado y revisable como cualquier otro cambio. La primera
migración (`000001_init_schema_and_seed`) trae, además del esquema, los datos base (ciudades,
conceptos, UVT, tarifas) — sin ellos el motor no tiene ninguna regla que aplicar, así que
separar "crear tablas" de "cargar datos mínimos" en dos pasos no aportaba nada. El runner
(`postgres.Migrate`) embebe las migraciones en el binario (`//go:embed migrations/*.sql`) y
reutiliza la conexión que ya administra GORM, sin abrir una segunda. El `.down.sql` ya existe
y está listo para revertir el esquema, pero **`cifrato` todavía no expone un comando `migrate
down`** — hoy solo se corre `Up()`; un rollback real requeriría correrlo manualmente contra la
base (`psql` o el CLI de `golang-migrate` instalado aparte) hasta que se cablee un subcomando.

### `dig` para inyección de dependencias
`internal/infrastructure/dependence/wire.go` define `NewWire() *dig.Container`: un único
archivo con `container.Provide(...)` por cada dependencia (Postgres, clasificador LLM, parser
XML, casos de uso), exactamente el patrón de composition root que usa `bia-electronic-bills`
(mismo nombre de archivo — `wire.go` — y misma función `NewWire()`, aunque el nombre venga de
cuando bia migró desde Wire y se quedó así). Es un contenedor en runtime basado en reflexión,
no código generado: `container.Provide(fn)` registra un constructor por su tipo de retorno, y
`container.Invoke(fn)` resuelve el grafo completo la primera vez que alguien pide algo — en
Cifrato eso pasa en `router.NewRouter(container)`, que hace `container.Invoke(func(h
*handlers.InvoiceHandler) {...})` al registrar las rutas, igual que
`router.NewRouter(container)` en bia. Cuando el constructor real devuelve un tipo concreto pero
el consumidor necesita una interfaz (ej. `postgres.NewInvoiceRepository` devuelve
`*postgres.InvoiceRepository`, pero `usecase.NewProcessInvoice` pide `repository.InvoiceRepository`),
se registra con una closure que envuelve el constructor y declara la interfaz como tipo de
retorno — el mismo patrón que bia usa para bindear `repository.BillsRepository` a su adaptador
API.

**Trade-off frente a Wire** (usado antes en esta misma sesión): con Wire, un error de cableado
(falta un provider, dos providers para el mismo tipo) lo detecta la herramienta `wire` **antes**
de que exista un binario — el comando falla en generación. Con `dig`, ese mismo error solo
aparece en **runtime**, la primera vez que se llama `container.Invoke(...)` (en Cifrato, al
arrancar `cifrato serve`). Se adoptó `dig` específicamente para igualar la convención de
`bia-electronic-bills`, no por preferencia técnica sobre Wire.

## Limitaciones conocidas

- **UVT y tarifas usan una sola vigencia (`2025-01-01`) para todo el rango de prueba.** Las 10
  facturas de muestra van de 2025-08 a 2026-04; usar el valor UVT 2026 ($52.374) como
  aproximación para todo ese rango es una simplificación deliberada para que las 10 facturas
  sean procesables de punta a punta en esta prueba. El valor UVT 2025 real es distinto — antes
  de producción, sembrar ambas vigencias por separado.
- **Las tarifas de ReteICA (las 3 ciudades) y de ReteFuente para `transporte_carga` son
  representativas, no verificadas contra el Acuerdo Municipal vigente ni contra un contador.**
  El contexto de la prueba solo da un rango legal general (Ley 1819/2016 art. 342) y un rango
  específico de Bogotá sin desglose por actividad económica — no hay cifras para Medellín ni
  Girardota en la fuente. Cada fila semilla con estas tarifas lo dice explícitamente en su
  `legal_basis` (`"TARIFA DE EJEMPLO PARA PRUEBAS, NO VERIFICADA..."`).
- **El catálogo de conceptos fiscales es mínimo** (`compra_bienes`, `servicios_generales`,
  `transporte_carga`) — suficiente para clasificar las 10 facturas de muestra, no exhaustivo
  frente a todos los conceptos de retención que existen en la norma.
- **RETEICA siempre usa la ciudad del emisor/vendedor**, nunca la del comprador — decisión de
  negocio explícita (ReteICA grava dónde opera quien vende, no quien compra).
- **La búsqueda de ciudad para RETEICA compara nombres normalizados** (mayúsculas, sin
  tildes, sin puntuación) en vez de un `name = ?` exacto — necesario porque el XML trae el
  `CityName` tal cual lo escribió el emisor (ej. `"Bogotá, D.C."`) y las ciudades sembradas
  usan un formato distinto (`"BOGOTA D.C"`); con comparación exacta nunca coincidían y RETEICA
  no se aplicaba a ninguna factura de Bogotá de la muestra. La normalización vive en Go
  (`ReferenceDataRepository.normalizeCityName`), no en SQL — la tabla de ciudades es pequeña y
  se carga una vez por factura, así que no hace falta un índice funcional ni la extensión
  `unaccent` de Postgres. Sigue siendo una comparación por texto, no por código DIAN de
  ciudad (`Address/ID`, ej. `11001`); dos nombres distintos que normalicen igual (poco
  probable con nombres de municipios reales) colisionarían.
- **Un solo endpoint HTTP** (`POST /invoices`, una factura por request) — no hay carga por
  lote ni autenticación; ambos quedan fuera del alcance de esta prueba.
- Los `WithholdingTaxTotal` que algunas facturas traen del proveedor son solo informativos —
  el motor calcula de forma independiente y no los usa como fuente de verdad (la
  responsabilidad legal de retener es del comprador).

## Tests

```bash
go test ./...
```

- `internal/domain/service/withholding_engine_test.go` — motor de cálculo puro, sin IO.
- `internal/infrastructure/adapters/xmlparser/xmlparser_adapter_test.go` — las 10 facturas
  reales de `sample-invoices/`, ambas variantes (`Invoice` directo y `AttachedDocument`/CDATA).
- `internal/application/usecase/calculate_withholdings_use_case_impl_test.go`,
  `classify_invoice_lines_use_case_impl_test.go` — casos de uso con fakes en memoria (sin red
  ni Postgres).
- `internal/infrastructure/adapters/repository/postgres/invoice_repository_impl_test.go` —
  requiere Postgres (`DB_HOST`), se salta si no está disponible.
- `internal/infrastructure/adapters/repository/postgres/reference_data_repository_impl_test.go`
  — normalización de nombres de ciudad para RETEICA (puro, sin Postgres), reproduce
  directamente el caso real `"Bogotá, D.C."` (XML) vs `"BOGOTA D.C"` (semilla).
- `internal/infrastructure/adapters/api/anthropic/classifier_adapter_test.go` — integración
  real contra la API de Anthropic, se salta si `ANTHROPIC_API_KEY` no está seteada.

`sample-invoices/` no se versiona en git (son datos reales de facturación de empresas
colombianas) — se agregan localmente para correr las pruebas contra facturas reales.
