# KAP financial reports source record

**Retrieved:** 2026-09-09T11:51:38+03:00 (Europe/Istanbul)<br>
**Scope:** Public filings of listed Turkish companies; intended for a declared
software/technology company panel, not a complete AI-company universe

## Authoritative sources

- Operator: Merkezi Kayıt Kuruluşu A.Ş. (MKK), Public Disclosure Platform (KAP)
- Platform: <https://www.kap.org.tr/en>
- Official description and reporting scope:
  <https://www.kap.org.tr/en/about/general-information>
- Turkish regulations and guides:
  <https://kap.org.tr/tr/about/mevzuat-duyurular-ve-kilavuzlar/tab-content/mevzuatlar_ve_kilavuzlar>

The general-information page returned HTTP 200 on the retrieval date. KAP says it
is the system for electronically signed disclosures required by Turkish capital-
markets and Borsa İstanbul rules, is operated continuously, and serves as a
historical archive. Financial reports include statements, management reports,
responsibility declarations and notes; notifications may be available as HTML,
Word, Excel or XML, with PDF attachments.

## Appropriate use

KAP is an authoritative source for reported revenue, profit, segment information,
currency, consolidation status and explanatory notes for companies required to
file. It can support a reproducible panel of listed software companies if the
company universe and inclusion date are frozen before looking at outcomes.

KAP is **not** a registry of all Turkish software or AI companies. Private firms
are mostly absent, and a company describing AI products does not make all its
revenue “AI revenue.” Use only an explicitly disclosed AI segment as AI-specific
revenue; otherwise label the value total company or reported segment revenue.

## Acquisition contract

1. Define the company universe using documented, outcome-independent rules.
2. Record KAP company code, legal name, fiscal year/period, consolidation status,
   presentation currency, filing type, notification identifier and filing time.
3. Download the filed financial report and notes, not only the convenience summary
   shown on a company page.
4. Prefer the issuer's reported XBRL/XML/Excel representation where KAP officially
   exposes it; retain the original filing alongside parsed data when permitted.
5. Record resolved URL, media type, byte size and SHA-256 for every filing.
6. Link amendments and corrections to the original accession/notification rather
   than overwriting history.

No documented, supported public bulk API contract was verified. Do not treat an
undocumented `/api/` path discovered in browser traffic as stable or approved.
Use the official search/download interface conservatively unless MKK publishes an
automation contract. Cache downloads; do not poll filings aggressively.

## Accounting and revision traps

- Distinguish consolidated from unconsolidated reports and annual from cumulative
  interim periods.
- Use the filing's own presentation currency, scale and accounting standard.
- Inspect comparative-column restatements. KAP warns that convenience tables based
  on the latest current-period column do not necessarily incorporate corrections
  later made to an earlier comparative period; the full filed report controls.
- Treat mergers, disposals, fiscal-year changes, hyperinflation accounting and
  segment reorganisations as possible breaks.
- Do not sum overlapping parent and subsidiary issuers.

## Safety and unresolved items

- Public viewing is verified; a dataset-level license for automated bulk reuse and
  redistribution was not found in the inspected pages. Resolve it before committing
  raw filings or operating a scheduled collector.
- Filings may contain signatures, addresses or other personal information. Extract
  only fields needed for company financial analysis and do not republish personal
  data.
- Never infer a missing AI segment as zero AI revenue.
