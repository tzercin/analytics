# OECD Structural Business Statistics source record

**Retrieved:** 2026-09-09T11:51:38+03:00 (Europe/Istanbul)<br>
**Scope:** Structural business statistics by economic activity and enterprise size

## Authoritative sources

- Publisher: Organisation for Economic Co-operation and Development (OECD)
- Dataset: Structural business statistics by size class and economic activity
  (ISIC Rev. 4)
- Dataflow identifier:
  `OECD.SDD.TPS,DSD_SDBSBSC_ISIC4@DF_SDBS_ISIC4,1.0`
- Data Explorer:
  <https://data-explorer.oecd.org/vis?df%5Bag%5D=OECD.SDD.TPS&df%5Bds%5D=dsDisseminateFinalDMZ&df%5Bid%5D=DSD_SDBSBSC_ISIC4%40DF_SDBS_ISIC4&df%5Bvs%5D=1.0>
- Versioned structure query:
  <https://sdmx.oecd.org/public/rest/dataflow/OECD.SDD.TPS/DSD_SDBSBSC_ISIC4@DF_SDBS_ISIC4/1.0?references=all>
- OECD API guide:
  <https://www.oecd.org/en/data/insights/data-explainers/2024/09/api.html>
- API best practices and current rate limits:
  <https://www.oecd.org/en/data/insights/data-explainers/2024/11/Api-best-practices-and-recommendations.html>
- Terms and conditions:
  <https://www.oecd.org/en/about/terms-conditions.html>

The structure query returned HTTP 200 and SDMX 2.1 structure XML on the retrieval
date. Its dimensions, in order, were `FREQ`, `REF_AREA`, `MEASURE`, `ACTIVITY`,
`SIZE_CLASS`, `UNIT_MEASURE`, and `TIME_PERIOD`.

## Relevant coverage

The dataflow covers annual measures including enterprise counts, turnover, value
added and employment, subject to country/activity availability. Its activity
codelist contained `J5820` (Software publishing) and `J6201` (Computer programming
activities). Always resolve measures and units from the live structure; do not
infer them from an activity name.

**Türkiye availability gate:** although `TUR`, `J5820`, and `J6201` exist in the
structure, live queries for Türkiye enterprise counts at those detailed activities
returned HTTP 404 `NoResultsFound` on 2026-09-09. The tested ordered keys were
`A.TUR.ENTR.J5820._T.ENT` and `A.TUR.ENTR.J6201._T.ENT`. A valid code
combination is not evidence that observations exist. OECD SDBS is therefore
suitable for software-sector peer benchmarks only after an availability query; it
is not currently approved as the sole Türkiye source.

## Acquisition contract

The public REST base is:

```text
https://sdmx.oecd.org/public/rest/
```

Use a versioned dataflow, query the structure first, and construct the ordered
series key from current codes. Request `dimensionAtObservation=AllDimensions` and
an explicit format such as `csvfilewithlabels` or `jsondata`. Bound the period and
request only necessary countries, activities, measures and size classes.

Before downloading observations:

1. query the dataflow structure and preserve it with a checksum;
2. use a content-constraint/availability query when possible;
3. record the full ordered key, version, format and retrieval timestamp;
4. preserve `OBS_STATUS`, `UNIT_MULT`, `DECIMALS`, units and labels;
5. save the response bytes or, if not committed, their SHA-256 and byte size.

No API key is required. OECD currently documents a maximum of 60 data downloads
per hour and recommends local caching. Implement backoff for 429/5xx responses and
never fan out one request per observation.

## Revisions, licensing and interpretation

- Pin the dataflow version. The OECD warns that a latest-version query can receive
  non-backward-compatible structural changes.
- Snapshot each analysis input: current values may be corrected or extended, and
  the standard interface is not a complete vintage database.
- OECD terms generally permit extraction, adaptation and redistribution with the
  dataset's prescribed citation, including commercial use, unless additional
  restrictions or third-party ownership apply. Check the selected dataset's source
  metadata before committing raw data.
- Cross-country comparisons require the same ISIC activity, unit, size-class
  coverage, reference period and status flags. Missing observations are not zero.
- Turnover of firms classified as software publishers is not the same as customer
  expenditure on software in that country.
