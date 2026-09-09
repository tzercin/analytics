# TCMB Sektör Bilançoları source record

**Retrieved:** 2026-09-09T11:51:38+03:00 (Europe/Istanbul)<br>
**Scope:** Aggregate company balance sheets and income statements by economic
activity; no company-level record or observation is committed

## Authoritative sources

- Publisher: Türkiye Cumhuriyet Merkez Bankası (TCMB)
- Dataset and download page:
  <https://www.tcmb.gov.tr/wps/wcm/connect/TR/TCMB%2BTR/Main%2BMenu/Istatistikler/Reel%2BSektor%2BIstatistikleri/Sektor%2BBilancolari/Sektor%2BBilanco%2BVerileri/>
- Dataset overview:
  <https://www.tcmb.gov.tr/wps/wcm/connect/TR/TCMB%2BTR/Main%2BMenu/Istatistikler/Reel%2BSektor%2BIstatistikleri/Sektor%2BBilancolari/>
- Current Turkish metadata:
  <https://www3.tcmb.gov.tr/sektor/dosyalar/menu/metadata_tr.pdf>
- Historical releases:
  <https://www.tcmb.gov.tr/wps/wcm/connect/TR/TCMB%2BTR/Main%2BMenu/Istatistikler/Reel%2BSektor%2BIstatistikleri/Sektor%2BBilancolari/Arsiv/>
- EVDS terms, relevant when a series is acquired through EVDS:
  <https://evds3.tcmb.gov.tr/igmevdsms-dis/documents/showDocument?docId=18>

On the retrieval date the dataset page and metadata PDF returned HTTP 200. The
current page offered the 2025 release covering 2023-2024, a four-digit NACE panel,
column definitions, and an archive. Download URLs are CMS-generated and may
change, so consumers must discover them from the landing page rather than copy an
old opaque URL into code.

## Statistical scope

The current metadata says the aggregates combine company tax declarations and
financial statements supplied by Gelir İdaresi Başkanlığı, activity information
from TÜİK, and credit information from Türkiye Bankalar Birliği Risk Merkezi. The
published outputs are aggregates by sector and scale, not company microdata.

For software-sector work, inspect the live four-digit NACE catalogue before each
analysis. Candidate activities include software publishing and computer
programming, but the exact codes, labels, coverage, confidentiality treatment and
company counts must come from the downloaded release, not from memory.

This source can measure the finances of firms classified to an activity. It does
**not** directly measure domestic software expenditure or market size: reported
net sales can include exports and non-software products, imports can be absent,
and software produced inside firms classified to other sectors is outside the
software activity aggregate.

## Acquisition contract

1. Start at the official dataset page and record the visible release title and
   covered accounting years.
2. Download the smallest appropriate official ZIP/workbook and its column/formula
   dictionary. Prefer the four-digit NACE panel only when that grain is necessary.
3. Record final resolved URL, retrieval time, byte size, media type and SHA-256.
4. Preserve original NACE codes, Turkish labels, units, scale class, observation
   count and any confidentiality/status markers.
5. Store a raw snapshot only after dataset-specific redistribution terms have been
   checked; otherwise store the checksum and deterministic retrieval instructions.

No authentication requirement was observed for the published files. Do not route
these downloads through a TCMB EVDS credential unless the selected series is in
fact served by EVDS.

## Revisions and interpretation

- Treat every annual package as a named vintage. Do not silently replace an older
  package with the newest file.
- The 2025 metadata describes special handling of 2023 balance sheets under
  inflation accounting. Comparisons across that boundary require an explicit
  accounting-method review.
- Reconcile totals, firm counts and units against the release's own tables before
  deriving ratios.
- Do not add sector aggregates across overlapping NACE levels.
- Nominal net sales are not real growth; any deflator and base period must be
  declared separately.

## Safety and unresolved items

- Dataset-page reuse and raw-file redistribution terms were not independently
  verified. EVDS terms must not be assumed to govern non-EVDS downloads.
- Publication timing is annual in practice, but a stable machine-readable release
  calendar and file API were not verified.
- Never attempt to reconstruct company records or bypass statistical disclosure
  controls from aggregates.
