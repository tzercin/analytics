# TÜİK SDMX API source record

**Retrieved:** 2026-09-08T14:33:50+03:00 (Europe/Istanbul)
**Scope:** Client protocol only; no statistical dataset or observation retrieved

## Authoritative sources

- Publisher: Türkiye İstatistik Kurumu (TÜİK)
- Service documentation:
  <https://veriportali.tuik.gov.tr/tr/sdmx-web-service-documentation>
- REST base URL published in that documentation:
  <https://nsiws.tuik.gov.tr/rest>
- Token endpoint published in that documentation:
  <https://giris.tuik.gov.tr/realms/web/protocol/openid-connect/token>
- Referenced standard: SDMX 2.1 structural objects and SDMX REST API 1.5
- SDMX 1.5 specification:
  <https://github.com/sdmx-twg/sdmx-rest/tree/v1.5.0>

The documentation page is JavaScript-rendered. At retrieval, its 3,692-byte
HTML shell had SHA-256
`aba2dabaf0e666f9cca4e2ee715759b4cbd597b7f27bf722373315a5b10db464`
and referenced the 1,132,174-byte asset `assets/main-QWuvKILy.js`, whose
SHA-256 was
`2f315e21f34197c82ca6d81b35e6cd4b52b3060f7a5720eec1dde1a696821649`.
The retrieved source bytes are not committed.

## Implemented contract

The documentation requires a bearer access token for service requests. It
documents two token grants:

1. Data Portal API key: form fields `grant_type=password`,
   `client_id=nsi-ws-consumer`, and `api_key`.
2. TÜİK-provisioned credentials: form fields
   `grant_type=client_credentials`, `client_id`, and `client_secret`.

The package supports documented structural resources for dataflows, data
structures, code lists, concept schemes, category schemes, and categorisations.
It also supports dataflow-based data requests, ordered series keys,
case-sensitive `startPeriod` and `endPeriod`, and the documented `format`
values for SDMX-ML 2.1 Structured Data, SDMX-CSV 1.0, and JSON-stat. Omitting
`format` requests the documented SDMX-ML 2.1 Generic Data default.

## Safety and unresolved items

- No credential, token, dataset identifier, table identifier, series key, or
  statistical value is stored in this repository.
- Automated package tests use local HTTP servers and require no secret. On
  2026-09-08, the client was also exercised against the live authenticated
  service to resolve CPI metadata and retrieve one CPI series; the credential
  remained in the secure runtime environment and was not logged or stored.
- During that live check, a data request carrying the documented
  `format=SDMX-CSV` parameter returned SDMX-ML 2.1 Generic Data with an XML
  content type. The first analysis therefore consumes the default Generic Data
  response. Format negotiation needs additional verification before callers
  rely on CSV or JSON-stat responses.
- The documentation notes that the bulk `codelist/TR/all` request had produced
  an error and requires separate technical verification. The client does not
  claim endpoint availability.
- Rate limits, general service terms, data-specific licenses, redistribution
  permissions, snapshot rules, and revision policies were not stated on the
  inspected documentation page. They must be verified for each analysis before
  retrieval, storage, or publication.
- The documentation is a JavaScript-rendered page and may change without a
  versioned page URL. Reverify it before relying on the client for a production
  or scheduled analysis.
