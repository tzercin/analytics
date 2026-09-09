# SEC EDGAR financial-data source record

**Retrieved:** 2026-09-09T11:51:38+03:00 (Europe/Istanbul)<br>
**Scope:** Public-company filings and XBRL facts for a declared software/AI-company
panel; no claim that total issuer revenue is AI-specific

## Authoritative sources

- Publisher: U.S. Securities and Exchange Commission (SEC)
- EDGAR API documentation:
  <https://www.sec.gov/search-filings/edgar-application-programming-interfaces>
- API host: <https://data.sec.gov/>
- Developer resources:
  <https://www.sec.gov/about/developer-resources>
- Automated-access rate-control notice:
  <https://www.sec.gov/filergroup/announcements-old/new-rate-control-limits>
- EDGAR overview:
  <https://www.sec.gov/submit-filings/about-edgar>

The public XBRL APIs require no authentication or API key. A read-only
`companyfacts` request returned HTTP 200 JSON on the retrieval date when sent with
an identifying User-Agent. SEC documentation says submissions and XBRL APIs update
throughout the day, while bulk archives are rebuilt nightly.

## Supported interfaces

```text
https://data.sec.gov/submissions/CIK##########.json
https://data.sec.gov/api/xbrl/companyfacts/CIK##########.json
https://data.sec.gov/api/xbrl/companyconcept/CIK##########/{taxonomy}/{tag}.json
https://data.sec.gov/api/xbrl/frames/{taxonomy}/{tag}/{unit}/{period}.json
```

CIKs in these paths are zero-padded to ten digits. Official bulk archives include
`companyfacts.zip` and `submissions.zip`; prefer them to thousands of individual
requests when the task is genuinely cross-company.

## Acquisition contract

- Send a truthful User-Agent identifying the application and a maintainer contact.
- Stay below the SEC's published ceiling of 10 requests/second across all machines;
  this repository should default to no more than 2 requests/second and exponential
  backoff on 429/403/5xx responses.
- Cache immutable filing documents by accession number and honor HTTP caching
  headers for API resources.
- Record CIK, ticker only as a mutable alias, accession number, form, filed date,
  fiscal period/fiscal year, start/end dates, taxonomy, tag, unit and frame.
- Preserve the exact API/file URL, retrieval timestamp, bytes and SHA-256.

## Revenue extraction rules

Do not select whichever tag produces the desired value. Establish a documented
tag policy, inspect the issuer's statement and notes, and reconcile the extracted
fact to the filed 10-K/10-Q/20-F. The APIs aggregate standard taxonomy facts that
apply to the whole filing entity; issuer extensions and segment disclosures can
require reading the original filing.

Deduplicate facts by period, form, accession and filing date. Prefer the latest
amended filing only when the analysis explicitly chooses a latest-known vintage;
otherwise retain both original and amendment. SEC frames align issuers to calendar
periods approximately, so company facts are safer when fiscal calendars differ.

An issuer selling AI products may also sell unrelated products. Total revenue is
not “AI revenue” unless the issuer reports an AI-specific segment or other
auditable disaggregation.

## Safety and unresolved items

- Access to the public database is free, but filing exhibits can contain third-
  party material. Do not assume every embedded work has unrestricted reuse rights.
- Do not collect officer addresses, signatures or other personal data for a
  company-revenue analysis.
- Freeze the AI-company inclusion rule and its evidence before computing growth or
  rankings; SEC industry/ticker metadata is not an AI taxonomy.
