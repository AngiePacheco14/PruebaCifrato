# Contexto: Prueba técnica Cifrato — Motor de retenciones sobre facturas de compra

## Objetivo de la prueba
Construir una solución que reciba una o varias facturas electrónicas colombianas y calcule
automáticamente las retenciones aplicables (ReteFuente, ReteIVA, ReteICA), **justificando**
cómo llegó al resultado.

Fecha de entrega objetivo: **10 de julio de 2026** (entregar antes si es posible).

## Datos de entrada: facturas de muestra
10 facturas electrónicas colombianas reales, cada una en dos formatos:
- XML UBL DIAN (fuente de verdad para el cálculo)
- PDF (representación gráfica, no parsear salvo que falte el XML)

Dos variantes de XML detectadas:
- `Invoice` directo (raíz `<Invoice>`)
- `AttachedDocument` que envuelve el `Invoice` real dentro de un bloque `CDATA`
  (contiene además firma digital XAdES, ignorar para el cálculo)

Ninguna factura de muestra trae código UNSPSC — solo `StandardItemIdentification` (SKU del
proveedor) y descripción libre en texto. Esto implica que **clasificar el concepto de
retención requiere un paso de clasificación** (heurística o asistida por LLM), no un
campo estructurado directo.

Algunas facturas ya traen `WithholdingTaxTotal` calculado por el proveedor (ReteFuente,
ReteIVA, ReteICA). Es solo informativo/sugerido por el emisor — la responsabilidad legal de
retener es del comprador (agente retenedor), así que el motor debe calcular de forma
independiente y, como mucho, usar esos valores para cruzar/validar.

Ciudades de emisores en la muestra: Bogotá, Medellín, Girardota — relevante para ReteICA.

## Marco normativo

### ReteFuente y ReteIVA — nacional (DIAN)
- Base legal: Estatuto Tributario (arts. 365-419 renta, 437-2 IVA), Decreto 1625 de 2016.
- UVT 2026: **$52.374** (Resolución DIAN 000238 del 15 de diciembre de 2025).
- **Importante**: las bases de ReteFuente por compras/servicios cambiaron varias veces en
  2026 por litigio sobre el Decreto 572 de 2025 (suspendido el 7 de mayo de 2026, suspensión
  revocada el 2 de junio de 2026, con efectos desde el 1 de julio de 2026). A la fecha de
  este documento rigen las bases del Decreto 572 (10 UVT compras / 2 UVT servicios), pero
  puede volver a cambiar. **Las tarifas no deben vivir hardcodeadas**: necesitan versionado
  por fecha de vigencia (`vigente_desde` / `vigente_hasta`).
- ReteIVA: normalmente 15% sobre el IVA generado (no sobre la base), aplica cuando el
  comprador es agente de retención de IVA.

### ReteICA — territorial/municipal
- Marco nacional (límites): Ley 14 de 1983 / Decreto 1333 de 1986, Ley 1819 de 2016 art. 342
  (2-7 x1000 industrial, 2-10 x1000 comercial/servicios).
- Cada Concejo Municipal fija su propia tarifa por Acuerdo, según actividad económica (CIIU).
  No hay tabla ICA nacional única — es un catálogo por municipio.
- Ejemplo Bogotá: tarifas de 4.14 a 13.8 por mil según actividad (Secretaría Distrital de
  Hacienda), 14 por mil para financieras.
- Base gravable y bases mínimas también varían por municipio.

## Decisiones de arquitectura acordadas

1. **Parser XML en Go**: detectar `Invoice` vs `AttachedDocument` (extraer CDATA), usar
   `encoding/xml` con structs por nombre local de tag (sin lidiar con namespaces
   explícitamente). Extraer: emisor (NIT, ciudad, responsabilidad fiscal/TaxLevelCode),
   comprador, líneas (concepto, cantidad, valor, IVA), totales, WithholdingTaxTotal existente.

2. **Motor de reglas determinístico** (fuente de verdad para el cálculo, NO delegar a LLM):
   - Tablas versionadas por fecha de vigencia, no hardcodeadas.
   - Tabla ReteFuente/ReteIVA nacional + tabla ReteICA por municipio (Bogotá, Medellín,
     Girardota para esta prueba).
   - Cada retención calculada devuelve: `{tipo, base, tarifa, valor, justificación,
     norma/artículo}`.

3. **Clasificación de concepto** (heurística por palabras clave para el MVP; LLM como
   extensión mencionada en el README, no implementada como núcleo): mapear descripción
   libre → concepto normalizado (compra de bienes, servicio, transporte, etc.), con opción
   de marcar "revisión humana" si la confianza es baja.

4. **Por qué no usar LLM para el cálculo mismo**: el monto/tarifa/base deben ser
   reproducibles y auditables (la prueba exige justificar el resultado). Un LLM puede
   alucinar un porcentaje o una UVT vieja — no es aceptable en un contexto fiscal.

5. **Salida**: JSON estructurado con el detalle y justificación por retención, más resumen
   por factura. Tests table-driven usando las 10 facturas de muestra reales como fixtures.

## Plan de tiempo estimado
- Parser XML en Go: 2-3 h
- Motor de reglas (tablas ReteFuente/ReteIVA nacional + ReteICA Bogotá/Medellín/Girardota): 3-4 h
- Clasificador de concepto (heurístico): 1-2 h
- Tests con las 10 facturas reales + output con justificación: 2-3 h
- README (enfoque, supuestos, limitaciones, cómo se manejaría el versionado de tarifas): 1 h
- Total MVP: 9-13 h

## Próximo paso
Implementar el parser XML en Go primero, usando las 10 facturas de `/mnt/user-data/uploads/sample-invoices.zip`
como fixtures de prueba.
