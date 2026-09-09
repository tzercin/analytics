# Eurostat enterprise AI-adoption source record

**Retrieved:** 2026-09-09T11:51:38+03:00 (Europe/Istanbul)<br>
**Scope:** Enterprise use of artificial-intelligence technologies, including
Türkiye; not counts or revenue of AI vendors

## Authoritative sources

- Publisher: Eurostat
- Dataset code: `isoc_eb_ai`
- Dataset API query used for verification:
  <https://ec.europa.eu/eurostat/api/dissemination/statistics/1.0/data/isoc_eb_ai?geo=TR&sinceTimePeriod=2024&lang=en>
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

The verification query returned HTTP 200 JSON-stat on 2026-09-09. The response was
labelled `Artificial intelligence by size class of enterprise`, reported an update
timestamp of `2026-06-15T11:00:00+0200`, and contained 2024 and 2025 positions for
Türkiye. Those facts describe the retrieved response, not a promised update
schedule.

## Statistical scope

The dimensions observed were `freq`, `size_emp`, `nace_r2`, `indic_is`, `unit`,
`geo`, and `time`. The indicator codelist distinguishes technologies, purposes,
acquisition modes and barriers; the unit codelist includes percentages with
different denominators. Agents must select a single documented indicator/unit
combination and preserve all flags.

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

For each retrieval record the exact URL, response `updated` timestamp, retrieval
time, dimensions, codes, labels, status flags, byte size and SHA-256. Retain the
raw JSON-stat response when reuse and repository-size rules permit. Validate array
indexes carefully: JSON-stat values are positional, and absent cells are not
necessarily zeros.

## Revisions, licensing and interpretation

- Eurostat's standard database generally presents the latest data. Corrections can
  replace prior values, so preserve analysis-time snapshots.
- Follow the Eurostat reuse notice and attribution requirements; inspect any
  third-party material separately.
- Do not compare percentages with different enterprise-size coverage,
  denominators, NACE coverage or survey years.
- Preserve break, estimate, provisional and confidentiality flags.
- Survey self-reporting and rapid changes in the AI questionnaire are material
  interpretation limits. A change in the measured share is not by itself a change
  in the count of AI suppliers.
