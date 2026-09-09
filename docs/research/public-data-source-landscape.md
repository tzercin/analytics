# Authoritative public-data source landscape beyond TÜİK

**Status:** Research draft for Chef review; no implementation or publication approval<br>
**Verified:** 2026-09-08 (Europe/Istanbul)<br>
**Scope:** Public, authoritative publishers that can support reproducible custom
analyses, prioritising Türkiye and complementarity with the repository's existing
headline CPI analysis.

## How to read this note

- **Verified fact** means the cited official page, metadata, documentation, terms,
  catalogue response, or a small read-only HTTP request directly supported the
  statement on the verification date.
- **Analysis hypothesis** is an idea to test, not a finding or causal claim.
- **Unverified / gate** identifies facts that must be resolved before acquisition,
  storage, redistribution, or publication.
- No endpoint, table/series identifier, license, or revision guarantee is inferred
  from a product name. Dataset-level identifiers are included only where observed
  in an official catalogue or live response.

Repository review found no GitHub issues, one merged PR (`#1`, the TÜİK SDMX client
and CPI analysis), and no enabled GitHub Discussions. The current branch was clean
before this note was added.

## Ranking rubric

Each dimension is scored **1 (weak) to 5 (strong)** with equal weight; total is out
of 30. Scores are **opinion**, based on verified access and metadata rather than on
an analysis result.

- **R — public relevance:** likely value to a broad Türkiye-focused audience.
- **N — novelty:** distance from a routine headline chart.
- **A — reproducibility/access stability:** machine access, authentication burden,
  stable discovery, and snapshot feasibility.
- **Q — quality/provenance:** source ownership, definitions, quality and revision
  disclosure.
- **F — feasible scope:** can a first defensible result be produced without a
  sprawling model or unsafe data handling?
- **C — CPI complement:** adds an explanatory or welfare-relevant domain rather
  than repeating the existing CPI series.

| Rank | Candidate | R | N | A | Q | F | C | Total |
|---:|---|---:|---:|---:|---:|---:|---:|---:|
| 1 | TCMB — Konut Fiyat Endeksi | 5 | 4 | 4 | 5 | 5 | 5 | **28** |
| 2 | SGK — Sigortalı İstatistikleri | 5 | 5 | 3 | 5 | 4 | 5 | **27** |
| 3 | BDDK — FİNTÜRK | 5 | 5 | 3 | 5 | 4 | 5 | **27** |
| 4 | EEA — Türkiye air-quality time series | 5 | 4 | 5 | 5 | 3 | 4 | **26** |
| 5 | EPİAŞ — electricity price and generation | 5 | 5 | 3 | 4 | 3 | 5 | **25** |
| 6 | AFAD — earthquake event catalogue | 5 | 4 | 5 | 4 | 4 | 3 | **25** |
| 7 | Eurostat — Türkiye NEET rates | 4 | 3 | 5 | 5 | 5 | 3 | **25** |
| 8 | HMB/Muhasebat — provincial budget statistics | 5 | 4 | 3 | 4 | 3 | 4 | **23** |
| 9 | World Bank — WDI financial-access comparison | 4 | 2 | 5 | 4 | 5 | 3 | **23** |
| 10 | İBB — hourly traffic density | 4 | 5 | 3 | 3 | 2 | 4 | **21** |

## Top-five shortlist

### 1. Türkiye Cumhuriyet Merkez Bankası — Konut Fiyat Endeksi (KFE)

- **Verified facts:** TCMB publishes the monthly, unadjusted `Konut Fiyat Endeksi`
  through EVDS under data group `bie_kfe`. The current metadata says the comparable
  series begins in 2010, uses 2023 as the base, publishes a first result about 15
  days after month-end and a final result about 45 days after month-end, and covers
  Türkiye plus specified İBBS Düzey 1/2 regions. KFE uses hedonic regression on
  appraisal reports prepared for individual housing-loan applications; a sale or
  loan disbursement need not occur. The metadata explicitly says past values can be
  updated when appraisals arrive late or are corrected. A 2024 methodological
  revision recalculated the history from January 2010 and changed source timing,
  models, weights, and base year. EVDS terms permit use and republication with
  attribution. [Landing page](https://www.tcmb.gov.tr/wps/wcm/connect/TR/TCMB+TR/Main+Menu/Istatistikler/Reel+Sektor+Istatistikleri/Konut+Fiyat+Endeksi/),
  [current metadata](https://www.tcmb.gov.tr/wps/wcm/connect/b4628fa9-11a7-4426-aee6-dae67fc56200/KFE-Metaveri.pdf?MOD=AJPERES),
  [2024 revision notice](https://www.tcmb.gov.tr/wps/wcm/connect/TR/TCMB+TR/Main+Menu/Duyurular/Basin/2024/DUY2024-44),
  [EVDS terms](https://evds3.tcmb.gov.tr/igmevdsms-dis/documents/showDocument?docId=18).
- **Question / grain:** How has quality-adjusted housing value changed relative to
  consumer prices since 2010, nationally and in the published regions? Monthly;
  Türkiye and the official mixed İBBS Düzey 1/2 publication geography.
- **Access:** EVDS browser/download and documented web services (JSON/XML/CSV
  representations). The exact KFE series codes must be resolved from live EVDS
  metadata immediately before implementation; only the verified group code is
  recorded here. [EVDS web-service guide](https://evds2.tcmb.gov.tr/help/videos/EVDS_Web_Service_Usage_Guide.pdf).
- **Joinability:** month to TÜİK CPI; official İBBS codes to TÜİK households or
  population where the grains really match.
- **Method traps:** an appraisal index is not a completed-sales price, affordability
  index, rent index, or construction-cost index; national CPI is not a local cost
  deflator; bases must be re-indexed before taking a ratio; the latest observation
  is revisable; region levels differ; annual weights and the 2024 backcast matter.
- **Analysis hypothesis:** real housing appreciation was heterogeneous across the
  published regions. This is not established until the data are retrieved and
  validated.
- **Unverified / gate:** smoke-test the current EVDS 3/EVDS 2 service contract,
  record exact series codes and flags, and decide whether licensed raw response
  snapshots may be committed.

### 2. Sosyal Güvenlik Kurumu — Sigortalı İstatistikleri

- **Verified facts:** SGK publishes monthly Excel bulletins. Its metadata describes
  active/passive insured people and workplaces, province/sex/sector breakdowns,
  average daily earnings, occupational accidents and diseases, a roughly 60-day
  finalisation lag, and no microdata distribution. SGK can revise for definition,
  scope, classification, method, or error with public notice. The statistics page
  warns that economic activities use NACE Rev. 2.1 from March 2025.
  [Bulletins](https://www.sgk.gov.tr/istatistik/aylik/42919466-593f-4600-937d-1f95c9e252e6),
  [metadata](https://www.sgk.gov.tr/Download/DownloadFile?d=052fa159-65a5-4117-9dfc-8235b7dca01a&f=dd747308-588e-40a0-854a-1089e2089599.pdf),
  [revision policy](https://www.sgk.gov.tr/Download/DownloadFile?d=052fa159-65a5-4117-9dfc-8235b7dca01a&f=b24b67b3-8a10-428e-9d86-61900ceeebcd.pdf),
  [statistics index](https://www.sgk.gov.tr/istatistik/index/6863b1e8-c384-4f46-90c6-511dac2376d2).
- **Question / grain:** Which provinces and sectors gained or lost formal insured
  employment, and did reported average daily earnings keep pace with CPI? Monthly;
  province, sex, sector, and insurance status where the workbook supports them.
- **Access:** selector-generated `.xlsx` downloads; no documented public API was
  verified.
- **Joinability:** province codes/names to TÜİK population; month to CPI; NACE to
  TÜİK sector tables with an explicit Rev. 2 → 2.1 bridge.
- **Method traps:** administrative insured employment is not all employment and
  excludes/handles informal work differently; 4/1-a is workplace-report based;
  contribution floors/ceilings can distort reported earnings; classification and
  legal changes create breaks; provisional and final vintages must not be mixed.
- **Analysis hypothesis:** formal-employment and real reported-earnings paths
  diverged materially across provinces and sectors.
- **Unverified / gate:** workbook schema and stable download discovery, explicit
  reuse/redistribution terms, historical replacement behavior, and whether each
  month's workbook preserves its vintage.

### 3. Bankacılık Düzenleme ve Denetleme Kurumu — FİNTÜRK

- **Verified facts:** FİNTÜRK publishes selected banking-sector data quarterly by
  province, including cash/non-cash credit, deposit types, individual and sectoral
  credit, branch distribution, and ratios. Since the second version introduced in
  September 2010, provincial distributions are based on customers' residence
  rather than branch performance. The official explanation says inputs are
  provisional, later issues may revise prior periods, BDDK may change tables, and
  information may be republished without permission when BDDK is cited.
  [FİNTÜRK explanation](https://www.bddk.org.tr/BultenFinturk/tr/Home/Aciklama),
  [publication calendar](https://www.bddk.org.tr/Veri/Detay/71),
  [official scope description](https://www.bddk.org.tr/Duyuru/Detay/2139).
- **Question / grain:** Where did inflation-adjusted credit per resident, deposit
  dollarisation, gold deposits, or non-performing-credit ratios diverge after 2021?
  Quarterly; province and published bank-group dimensions.
- **Access:** interactive application with bulk/Excel export; no supported public
  API contract was verified.
- **Joinability:** province to TÜİK population; quarter to CPI/TCMB FX; sector to
  SGK/TÜİK only after mapping definitions.
- **Method traps:** residence is not loan-use location; stocks are affected by FX
  valuation and inflation; bank reporting perimeter and ownership classifications
  change; credit per capita is not household debt; revisions overwrite the apparent
  past unless snapshots are kept.
- **Analysis hypothesis:** nominal credit growth masks large province-level
  divergence after deflation and FX adjustment.
- **Unverified / gate:** exact export URLs/parameters, complete data dictionary,
  rate limits, and whether raw workbook redistribution is permitted in addition to
  republishing attributed information.

### 4. European Environment Agency — Air Quality Download Service

- **Verified facts:** the EEA service offers official country-reported monitoring
  time series from 2013 onward as zipped Parquet through a download API. It
  distinguishes verified E1a data from up-to-date/unverified E2a data. Türkiye is
  in the catalogue coverage. The metadata uses CC BY 4.0, warns of continuing
  updates and possible incompleteness/errors, identifies the main update windows,
  and tells users to download all inputs on the same day and keep their own
  versions. [Datahub record](https://www.eea.europa.eu/en/datahub/datahubitem-view/778ef9f5-6293-4846-badd-56a29c70880d),
  [metadata](https://sdi.eea.europa.eu/catalogue/datahub/api/records/1f964ae5-56eb-48c3-b89a-4820893341f7/formatters/xsl-view?approved=true&language=eng&output=pdf),
  [EEA access overview](https://www.eea.europa.eu/en/about/contact-us/faqs/where-can-i-access-the-latest-air-quality-data-in-europe/).
- **Question / grain:** Did validated PM2.5/PM10/NO2 concentrations and threshold
  exceedance days improve consistently across Turkish monitoring stations, and how
  sensitive is the result to station type and coverage? Hourly/daily observations;
  station coordinates; annual validated vintages.
- **Access:** download web API → zipped Parquet; station GIS service also exposes
  JSON/GeoJSON/PBF. No credential requirement was observed in the catalogue.
- **Joinability:** station coordinates to urban areas; population weighting from
  TÜİK only with defensible spatial catchments; İBB traffic for a bounded Istanbul
  extension.
- **Method traps:** monitors are not a representative population sample; station
  type, instrument, coverage, missingness, relocation and pollutant units matter;
  E2a and E1a cannot be silently mixed; association with traffic/weather is not
  causation.
- **Analysis hypothesis:** national summaries conceal station-level and pollutant-
  specific differences.
- **Unverified / gate:** exact Türkiye request payload and returned schema, station
  history fields, completeness criteria, and the source country's validation notes.

### 5. Enerji Piyasaları İşletme A.Ş. — Şeffaflık Platformu

- **Verified facts:** the current technical documentation exposes hourly
  `Piyasa Takas Fiyatı (PTF)` at
  `POST /v1/markets/dam/data/mcp` and source-level hourly generation at
  `POST /v1/generation/data/realtime-generation`; export variants support CSV,
  XLSX, or PDF. The generation documentation says plant data are available through
  the previous day. Since August 2024, authenticated services use registered
  username/password to obtain a two-hour TGT. EPİAŞ says platform and web-service
  access are free with unlimited historical access, while data ownership includes
  EPİAŞ and several reporting entities and EPİAŞ disclaims responsibility for
  data supplied by them. [Electricity API documentation](https://seffaflik.epias.com.tr/electricity-service/technical/tr/index.html),
  [authentication change](https://www.epias.com.tr/tum-duyurular/seffaflik-platformu-kullanici-giris-sisteminin-canliya-alinmasi-hk/),
  [platform basis and FAQ](https://www.epias.com.tr/epias-kurumsal/sss/).
- **Question / grain:** Are renewable-heavy hours associated with lower day-ahead
  PTF after accounting for load, hour, season, and major regime breaks? Hourly;
  national market and generation-source categories.
- **Access:** authenticated REST JSON/XML plus CSV/XLSX/PDF export. Credentials must
  remain runtime-only.
- **Joinability:** hour/date internally; month to TÜİK CPI for real TL/MWh; day to
  TCMB FX; weather only after a separately licensed source is approved.
- **Method traps:** PTF is wholesale day-ahead price, not a retail tariff; generation
  mix is endogenous; load, imports/exports, fuel, hydro conditions, caps and market
  rule changes confound associations; use fixed `+03:00` timestamps explicitly;
  API migrations have occurred.
- **Analysis hypothesis:** a within-hour/season comparison can quantify association,
  but cannot by itself identify the causal price effect of renewables.
- **Unverified / gate:** current registration workflow, rate limits, bulk window
  constraints, revision/correction policy, dataset-specific redistribution and raw
  snapshot rights.

## Remaining landscape

### 6. AFAD — Event Web Service and earthquake catalogue

- **Verified facts/access:** AFAD documents
  `https://deprem.afad.gov.tr/apiv2/event/filter` with time, bounding-box/radius,
  depth, magnitude, pagination/order, and output-format parameters. A small request
  on 2026-09-08 returned HTTP 200 JSON containing `eventID`, coordinates, depth,
  magnitude type/value, local administrative labels, `isEventUpdate`, and
  `lastUpdateDate`. AFAD asks users to cite AFAD/Türkiye Deprem Veri Merkezi Sistemi.
  [API guide](https://deprem.afad.gov.tr/event-service),
  [official catalogue and attribution notice](https://deprem.afad.gov.tr/event-catalog).
- **Question/grain:** describe spatial footprint and decay of aftershock sequences
  for pre-selected major events, with event-level UTC/local time, point, depth and
  magnitude. Join aggregated results to TÜİK population only as exposure context.
- **Revision/terms:** event-level update fields are observable; no general vintage,
  deletion, rate-limit, bulk-snapshot or redistribution policy was verified.
- **Traps:** ML/Mw/other magnitudes are not interchangeable; catalogue completeness
  and network sensitivity change; counts do not measure hazard or forecast risk;
  administrative nearest-place labels are not epicentre polygons.

### 7. Eurostat — `edat_lfse_20`, NEET rates

- **Verified facts/access:** a live public Statistics API call for
  `edat_lfse_20`, `geo=TR` returned JSON-stat labelled “Young persons neither in
  employment nor in education and training by labour status (NEET rates)” and an
  update timestamp. Eurostat also offers public SDMX 2.1/3.0 and CSV. The standard
  database exposes only the latest observation version; revisions normally replace
  values, while selected PEEI datasets separately preserve vintages. Reuse is free
  subject to Eurostat's copyright notice and listed exceptions.
  [live dataset query](https://ec.europa.eu/eurostat/api/dissemination/statistics/1.0/data/edat_lfse_20?geo=TR&sinceTimePeriod=2023&lang=en),
  [API guide](https://ec.europa.eu/eurostat/web/user-guides/data-browser/api-data-access/api-getting-started),
  [revision policy](https://ec.europa.eu/eurostat/data/data-revision-policy),
  [reuse notice](https://ec.europa.eu/eurostat/help/copyright-notice).
- **Question/grain:** how do Türkiye's annual NEET rates and gender/age gaps compare
  with the EU and candidate-country distributions? Annual country/sex/age/labour-
  status cells.
- **Joinability/traps:** harmonised comparison is the value; cross-check Türkiye
  against TÜİK LFS rather than silently substituting it. Preserve flags, survey
  breaks, age denominator, labour-status dimension and latest-only snapshots.

### 8. Hazine ve Maliye Bakanlığı / Muhasebat Genel Müdürlüğü — budgets

- **Verified facts/access:** official pages provide monthly central-government
  budget tables and provincial central-government budget statistics for 2004–2026,
  including spending, revenue and balance. Data are delivered through downloadable
  tables (principally spreadsheets; some reports are PDF), not a verified public
  API. HMB also publishes calendars and revision/error material for fiscal
  statistics. [Provincial series landing page](https://muhasebat.hmb.gov.tr/iller-itibariyle-merkezi-yonetim-butce-istatistikleri-2004-2026),
  [central-government tables](https://muhasebat.hmb.gov.tr/merkezi-yonetim-butce-istatistikleri),
  [fiscal statistics](https://www.hmb.gov.tr/genel-yonetim-sektoru-mali-istatistikleri).
- **Question/grain:** how did real per-capita capital spending and collected revenue
  vary across provinces and years? Monthly/national and annual or within-year
  provincial tables, depending on a schema check; join CPI and TÜİK population.
- **Revision/terms/traps:** exact provisional/final relationship and reuse terms are
  gates. Do not treat administrative booking location as beneficiary location;
  distinguish budget, accrual, collection and cash; detect cumulative/monthly
  layouts and classification changes.

### 9. World Bank — World Development Indicators (WDI)

- **Verified facts/access:** WDI provides a no-key indicators API plus bulk CSV/XLSX,
  indicator metadata, quarterly change notes and downloadable archives. A small
  live request for Türkiye and `SP.POP.TOTL` returned source `2` (WDI), an update
  date, and annual observations. The July 2026 release identifies financial-access
  indicators such as `FB.CBK.BRCH.P5` and `FB.CBK.DPTR.P3`. Default catalogue terms
  are CC BY 4.0 plus World Bank additions, but third-party indicators can carry
  extra restrictions. [WDI access](https://datatopics.worldbank.org/world-development-indicators/),
  [July 2026 release](https://datatopics.worldbank.org/world-development-indicators/release-note/jul-2026.html),
  [archives](https://datatopics.worldbank.org/world-development-indicators/wdi-archives.html),
  [terms](https://data.worldbank.org/summary-terms-of-use).
- **Question/grain:** how does Türkiye's annual commercial-bank access compare with
  declared peer economies, and does the cross-country picture agree with BDDK's
  within-Türkiye FİNTÜRK distribution?
- **Revision/terms/traps:** archive the chosen WDI release. WDI is often a compiler,
  not the originating owner; read each indicator's source/method/terms, avoid an
  outcome-driven peer set, and do not equate branches/depositors with inclusion or
  service quality.

### 10. İstanbul Büyükşehir Belediyesi — Hourly Traffic Density Data Set

- **Verified facts/access:** İBB's CKAN API returned package UUID
  `3ee6d744-5da2-40c8-9cd6-0e3e41f1928f`, title `Hourly Traffic Density Data Set`,
  61 monthly CSV resources from January 2020 through January 2025, and catalogue
  modification time `2025-04-16T11:02:33.454093`. The catalogue note says updates
  will come later; therefore freshness is a material weakness. Files are commonly
  around 100–140 MB per month. The İBB Open Data License permits copying,
  adaptation and commercial/non-commercial use with attribution.
  [CKAN package API](https://data.ibb.gov.tr/api/3/action/package_show?id=3ee6d744-5da2-40c8-9cd6-0e3e41f1928f),
  [dataset page](https://data.ibb.gov.tr/dataset/hourly-traffic-density-data-set),
  [license](https://data.ibb.gov.tr/license).
- **Question/grain:** which grid cells and hours saw persistent post-pandemic
  congestion shifts, and do patterns co-move with validated air-quality stations?
  Hour × spatial grid; join spatially to EEA stations, not directly to CPI.
- **Revision/traps:** no revision policy was found; several resource timestamps show
  later modifications, so checksum every file. Clarify grid/measurement definitions,
  missing cells, sensor coverage and timezone before analysis; use streaming or
  partitioned processing and do not commit multi-gigabyte raw files by default.

## Defer / watchlist

- **Meteoroloji Genel Müdürlüğü (MGM):** climate normals and aggregate official
  bulletins are public, but the official site says meteorological information not
  already published on the site is supplied through the paid MEVBİS order system.
  That prevents an open, unattended station-level pipeline unless access and reuse
  are separately approved. [Public climate statistics](https://www.mgm.gov.tr/Veridegerlendirme/il-ve-ilceler-istatistik.aspx?k=D),
  [MEVBİS/access terms](https://www.mgm.gov.tr/site/bilgi-edinme.aspx?r=a).
- **T.C. Sağlık Bakanlığı Açık Veri Portalı:** the portal and attribution-based
  license are useful, but the current catalogue has only five specialised clinical/
  ML datasets rather than a longitudinal public-health series. The anonymised
  `İzlem Veri Seti` is a fixed random sample of 1,000 records per listed disease
  group, so it is unsuitable for population prevalence or trend claims and demands
  a separate privacy/disclosure review. [Catalogue](https://acikveri.saglik.gov.tr/Home/DataSets),
  [terms](https://acikveri.saglik.gov.tr/app/doc/KullanimKosullari.pdf).

## Recommended first analysis

### Real house-price path since 2010: TCMB KFE deflated by TÜİK CPI

**Recommendation (opinion):** start with TCMB KFE. It is the best balance of
public relevance, new-source value, modest engineering scope, explicit methodology,
documented revisions, and direct complementarity with the existing CPI pipeline.

**Bounded question:** “From January 2010 through the latest common final month, how
did the quality-adjusted TCMB KFE change relative to the TÜİK headline CPI,
nationally and across only the KFE's officially published regional grains?”

**Proposed calculations:**

1. retrieve the exact national/regional KFE index series and TÜİK CPI index level,
   not published year-on-year rates;
2. choose and declare a common month, then calculate
   `real_KFE_t = 100 × (KFE_t / CPI_t) / (KFE_base / CPI_base)`;
3. report nominal KFE, CPI and real KFE as index paths plus 12-month log or percent
   changes, clearly labelled as calculations;
4. freeze retrieval timestamps, response bytes/checksums where terms permit, series
   metadata, base periods, flags and the latest/final status;
5. reconcile selected published points to TCMB's monthly table and rerun against a
   clean checkout.

**Acceptance gates before implementation:**

- Chef approves the question and national-versus-regional scope.
- Live EVDS metadata yields exact KFE codes and documented flags; the service is
  smoke-tested after the EVDS 3 beta transition.
- TÜİK CPI **index-level** series, base/vintage and common period are selected and
  separately approved; the existing annual-rate series cannot be used as a
  deflator.
- Raw snapshot/redistribution rights and secret handling are recorded.
- The output explicitly says “appraisal-based quality-adjusted price index,” not
  sales price, affordability, rent, wealth or return, and labels every claim as
  fact, calculation, inference or opinion.

## Open questions and key risks for Chef

1. Is the desired portfolio primarily economic/welfare analysis (favour KFE, SGK,
   FİNTÜRK) or broader civic/environmental work (favour EEA/AFAD)?
2. May implementations requiring free account credentials (TCMB/EPİAŞ) be scheduled,
   or should the next source be completely anonymous-access?
3. May raw official workbooks/API responses be committed when licenses permit, or
   should the repository retain checksums plus deterministic download instructions?
4. Should the first non-TÜİK analysis remain national to avoid mixing local KFE with
   a national CPI deflator?
5. Cross-cutting risk: several sources replace past values without serving complete
   vintages. Retrieval-time snapshots and hashes are part of the analysis, not an
   optional convenience.
