# Repository agent guidelines

These instructions apply to the entire repository. They govern research, data
acquisition, analysis, generated artifacts and documentation.

## Core rule

Every published number must be traceable to an authoritative source response and
a deterministic calculation. Prefer the originating public institution over news,
search snippets, dashboards copied by third parties or market-estimate blogs.

Before using a source, read its record in `docs/*-source.md`. If the interface,
terms, dataset identifier, schema or revision behavior has changed, update the
source record in the same change as the analysis.

## Source acceptance and discovery

1. Begin at the publisher's official landing page and documentation.
2. Verify the current endpoint or download link with a minimal read-only request.
3. Obtain dataset, dataflow, table, series and dimension identifiers from live
   official metadata. Never guess identifiers or construct them from labels.
4. Record whether the result is a primary official dataset, a company filing, or a
   secondary aggregate derived from a commercial/third-party source.
5. Check license/reuse terms, revision policy, update schedule, authentication and
   rate limits before downloading or committing data.
6. If any of those facts is unknown, mark it as an unresolved gate. Publicly
   viewable does not automatically mean redistributable.
7. Do not automate undocumented private/internal endpoints merely because they are
   visible in browser traffic. Prefer documented APIs, bulk files, or official
   export controls.

Current source records include:

- `docs/tuik-sdmx-source.md`
- `docs/tcmb-sector-balance-sheets-source.md`
- `docs/oecd-sdbs-source.md`
- `docs/eurostat-ai-adoption-source.md`
- `docs/kap-financial-reports-source.md`
- `docs/sec-edgar-source.md`
- `docs/oecd-ai-source.md`
- `docs/companies-house-source.md`
- `docs/invest-turkiye-startup-ecosystem-source.md`

## Network etiquette

- Request only the dimensions and periods needed. Use bulk downloads for genuinely
  large extracts instead of issuing one request per series or company.
- Cache responses and immutable filings. Use conditional requests when supported.
- Default to one concurrent request and no more than one request per second unless
  the source record documents another safe contract.
- Implement bounded retries with exponential backoff and jitter for 429 and
  transient 5xx responses. Honor `Retry-After`.
- Set a truthful identifying User-Agent where the publisher requests one.
- Never bypass authentication, bot protection, pagination limits or disclosure
  controls. A failed availability query is not permission to scrape another path.

Source-specific limits override these defaults. For example, the OECD source
record documents its current hourly download ceiling, and the SEC record documents
both the published ceiling and this repository's lower default.

## Credentials and sensitive data

- Secrets belong only in runtime environment variables or an approved secret
  store. Never commit credentials, tokens, session cookies or authorization
  headers.
- Never print a secret in logs, errors, tests, provenance or command examples.
- Tests must use local test servers or synthetic credentials by default. Live
  integration checks must be explicit and opt-in.
- Collect no personal data unless the approved analytical question requires it.
  Company-market work must exclude officer addresses, signatures, personal
  identifiers and similar fields.

## Raw, derived and presentation layers

Keep these layers separate:

1. **Raw:** exact source bytes, or a checksum plus deterministic retrieval
   instructions when redistribution or repository size prevents storage.
2. **Derived:** tidy tables produced only by version-controlled code.
3. **Presentation:** charts, prose and compact published tables generated from the
   derived layer.

Do not manually edit a derived CSV or chart. Do not commit large archives or bulk
datasets without maintainer approval. Prefer small, reviewable analysis artifacts.

## Required provenance

Each analysis must record, as applicable:

- publisher and dataset title;
- official landing page and documentation/terms URLs;
- exact resolved endpoint or file URL;
- dataset/dataflow/table/series identifiers and version;
- complete query, filters and ordered dimension key;
- retrieval timestamp with timezone;
- source period/vintage, update timestamp and filing/accession identifier;
- HTTP media type, byte size and SHA-256 of raw inputs;
- original codes, labels, units, multipliers, decimals and status flags;
- license/reuse conclusion and unresolved restrictions;
- transformation commands, software version and output checksum;
- revision/reconciliation notes.

Preserve source-language labels exactly, including Turkish characters. Use stable
codes for joins; never join only on a translated display label.

## Statistical integrity

- Missing, suppressed, confidential, provisional and not-applicable values are not
  zero. Preserve and report their flags.
- Do not mix vintages, base years, accounting scopes, currencies, price bases,
  seasonal adjustments or geographic grains without an explicit bridge.
- Freeze any company/peer universe and inclusion rule before examining outcomes.
- “Software-sector turnover” is not automatically software-market expenditure.
  “AI company revenue” requires an auditable AI-specific segment; otherwise call it
  total company revenue. “Investment deals” are not startup counts.
- Separate claims as **source fact**, **calculation**, **inference**, or **opinion**.
  Never turn an analysis hypothesis into a sourced fact.
- Reconcile selected observations and totals to the publisher's own presentation
  before publication.

## Implementation and review

- Make acquisition code deterministic and parameterized; avoid one-off manual
  download steps that cannot be replayed.
- Parsers must fail clearly on schema drift, unexpected units, duplicate keys or
  changed classifications. Do not silently discard unknown fields or flags.
- Unit tests must be offline, deterministic and based on minimal fixtures that
  contain no secrets or unnecessary personal data.
- Before handing off a change, run the relevant tests, `go test ./...` for Go
  changes, `git diff --check`, and inspect `git status --short` for raw data or
  secrets added accidentally.
- No analysis is ready for publication until a maintainer reviews the question,
  source rights, provenance, calculations, caveats and generated artifacts.
