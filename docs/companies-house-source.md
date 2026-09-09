# UK Companies House source record

**Retrieved:** 2026-09-09T11:51:38+03:00 (Europe/Istanbul)<br>
**Scope:** UK company register and electronically filed accounts; optional
cross-country company evidence, not a Türkiye source

## Authoritative sources

- Publisher: Companies House, UK Department for Business and Trade
- Data products:
  <https://www.gov.uk/guidance/companies-house-data-products>
- Developer hub:
  <https://developer.company-information.service.gov.uk/>
- Public Data API specification:
  <https://developer-specs.company-information.service.gov.uk/companies-house-public-data-api/reference>
- Guidance for searching the register:
  <https://www.gov.uk/guidance/searching-the-companies-house-register>

The data-products page returned HTTP 200 on the retrieval date. Companies House
documents a free monthly CSV snapshot of live companies, daily/monthly bulk files
of electronically filed accounts in XBRL/iXBRL, and read-only public-data APIs.
The official page states that electronic accounts represent about 60% of the
roughly 2.2 million accounts filed each year; this is a coverage statement, not a
fixed guarantee.

## Access contract

The REST API returns live public register data but requires an API key. Store the
key only in a runtime environment variable and never in source, logs, fixtures or
provenance files. The free bulk company and accounts products are preferable for
large analyses; do not enumerate the register through search requests.

For each company record preserve company number as the stable identifier, legal
name, status, incorporation/dissolution dates, SIC codes and snapshot month. For
accounts preserve filing/document identifiers, accounting period, filing date,
taxonomy, scale, currency, consolidation status, original XBRL/iXBRL bytes and
checksum.

Follow current developer rate limits, cache responses and implement backoff. The
official API is read-only for this work; no agent may use a filing or data-change
endpoint.

## Revenue and AI-classification limits

- UK SIC codes do not identify “AI companies.” Classification needs frozen,
  reviewable evidence independent of the financial outcome.
- Many small-company filing regimes can omit a turnover line. Missing turnover is
  not zero revenue.
- Filed accounts may be abbreviated, revised, dormant, consolidated or cover
  non-standard periods. Reconcile extracted XBRL to the rendered filing.
- Registered office, incorporation and SIC describe a legal entity, not necessarily
  its operating headquarters or product market.
- Use this source for named-company evidence or a carefully bounded UK comparison,
  not a global AI-startup count.

## Reuse and privacy

Companies House says it imposes no additional rules on use of public-register
information but users remain responsible for data-protection, copyright and other
law. The register includes personal data. For market analysis, do not download or
publish officers, residential addresses, signatures or people-with-significant-
control data unless a separately approved question requires them.
