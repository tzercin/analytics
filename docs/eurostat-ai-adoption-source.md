# Eurostat enterprise AI-adoption source record

**Retrieved:** 2026-09-09T14:33:25+03:00 (Europe/Istanbul)<br>
**Scope:** Enterprise use of artificial-intelligence technologies, including
Türkiye; not counts or revenue of AI vendors

## Authoritative sources

- Publisher: Eurostat
- Dataset code: `isoc_eb_ai`
- Data Browser landing page:
  <https://ec.europa.eu/eurostat/databrowser/view/isoc_eb_ai/default/table?lang=en>
- Dataset API query selected for the first analysis:
  <https://ec.europa.eu/eurostat/api/dissemination/statistics/1.0/data/isoc_eb_ai?freq=A&indic_is=E_AI_TTM&lang=en&nace_r2=C10-S951_X_K&sinceTimePeriod=2024&unit=PC_ENT&untilTimePeriod=2025>
- Statistics API guide:
  <https://ec.europa.eu/eurostat/web/user-guides/data-browser/api-data-access/api-getting-started/api>
- Digital-economy data documentation:
  <https://ec.europa.eu/eurostat/web/digital-economy-and-society/information-data>
- 2026 methodological/results report:
  <https://ec.europa.eu/eurostat/web/products-statistical-reports/w/ks-01-26-009>
- Revision policy:
  <https://ec.europa.eu/eurostat/data/data-revision-policy>
- Copyright and reuse notice:
  <https://ec.europa.eu/eurostat/help/copyright-notice>

The selected query returned HTTP 200 JSON-stat on 2026-09-09. The response was
labelled `Artificial intelligence by size class of enterprise`, identified
dataset `ISOC_EB_AI` version `1.0`, data structure `ISOC_EB_AI` version `45.0`,
and DOI `10.2908/ISOC_EB_AI`. It reported data update
`2026-06-15T11:00:00+0200` and structure update
`2025-12-11T11:00:00+0100`. Those facts describe the stored response, not a
promised update schedule.

## Statistical scope

The dimensions observed, in positional order, were `freq`, `size_emp`,
`nace_r2`, `indic_is`, `unit`, `geo`, and `time`. The first analysis selects
`E_AI_TTM`, “Enterprises using AI technologies performing analysis of written
language (text mining)”, with `PC_ENT`, “Percentage of enterprises”. It uses
annual frequency `A`, NACE aggregate `C10-S951_X_K`, years 2024–2025, and size
codes `GE10`, `10-49`, `50-249`, and `GE250`. These identifiers and labels came
from the live response, not from label-to-code guessing.

This narrower technology indicator was chosen instead of headline code
`E_AI_TANY` because the 2025 questionnaire added picture/video/sound generation
to the technologies included in that composite. Eurostat publishes direct
2024–2025 comparisons for text mining in its 2026 report. Annual questionnaire
and national-method changes still remain a comparability limitation.

The source is the enterprise ICT-usage survey. Published headline results concern
enterprises with at least 10 employees or self-employed persons in covered
activities. They measure enterprises **using** AI technology. They do not measure
AI startups, AI-company revenue, model usage volume, or the value of an AI market.

## Acquisition contract

The statistics endpoint is:

```text
https://ec.europa.eu/eurostat/api/dissemination/statistics/1.0/data/{datasetCode}
```

It is public and no credential was required in the verification request. Use
query parameters for every required dimension and bound time explicitly. Do not
download the full multidimensional cube when a small slice is sufficient.

No numeric request-per-second quota was found in the official Statistics API
guides checked on 2026-09-09. This remains an unresolved service-contract gate
for any scheduled acquisition. The first analysis makes one synchronous request,
honours `Retry-After`, and uses bounded backoff for HTTP 429 and transient 5xx
responses; offline reproduction makes no request.

For each retrieval record the exact URL, response `updated` timestamp, retrieval
time, dimensions, codes, labels, status flags, byte size and SHA-256. Eurostat's
reuse notice authorises commercial and non-commercial reuse of statistical data
and metadata with source acknowledgement; modifications must be identified and
carry a non-responsibility disclaimer. The selected 8.7 kB response is therefore
stored exactly with an acquisition receipt. Validate array indexes carefully:
JSON-stat values are positional, and absent cells are not necessarily zeros.

## Revisions, licensing and interpretation

- Eurostat's standard database generally presents the latest data. Corrections can
  replace prior values, so preserve analysis-time snapshots. Eurostat's digital
  economy information page says reference-year enterprise data are released in
  December; subsequent validated national revisions can still update datasets.
- Statistical data and metadata may be reused with attribution under the notice
  checked above. Do not reuse third-party images, logos or maps under that
  conclusion; inspect their rights separately.
- Do not compare percentages with different enterprise-size coverage,
  denominators, NACE coverage or survey years.
- Preserve break, estimate, provisional and confidentiality flags.
- Survey self-reporting and rapid changes in the AI questionnaire are material
  interpretation limits. A change in the measured share is not by itself a change
  in the count of AI suppliers.
