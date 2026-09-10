# TÜİK fertility and age-sex population source record

**Verified:** 2026-09-10 (Europe/Istanbul)

**Scope:** synchronized 2007–2025 annual fertility and resident-population data

## Publisher and official interfaces

- Publisher: Türkiye İstatistik Kurumu (TÜİK)
- Official portal: <https://veriportali.tuik.gov.tr/>
- SDMX documentation: <https://veriportali.tuik.gov.tr/tr/sdmx-web-service-documentation>
- REST base: <https://nsiws.tuik.gov.tr/rest>

The authenticated interface, token handling, formats and known service caveats
are recorded in [`tuik-sdmx-source.md`](tuik-sdmx-source.md). Discovery began
with a live `dataflow/TR/all/latest?detail=full&references=all` response; IDs
below were selected from that metadata, not inferred from titles.

## Selected primary official datasets

### Fertility

- Dataflow `DF_DOGUM_TEMEL_DOG_GOST_C` version `1.0`
- Turkish title: **Temel Doğurganlık Göstergeleri**
- DSD `DSD_DOGUM` version `1.1`
- Indicator `NG_TDH`: **Toplam Doğurganlık Hızı (Çocuk Sayısı)**
- Ordered key:
  `TR.A.NG_TDH._Z._Z._Z._Z._Z._Z._Z._Z._Z._Z._Z._Z._Z._Z._Z.2026_01`
- Period filter: `startPeriod=2007&endPeriod=2025`
- Indicator codelist: `CL_GOSTERGE` version `6.2`

The live structural metadata verifies the official title and unit in the title
(number of children), but contains no indicator-specific definition annotation.
The analysis therefore describes the conventional period-total-fertility-rate
interpretation and explicitly marks the absence of a separately retrieved TÜİK
methodological definition as an unresolved publication gate; it does not rename
the measure “average births observed per woman.”

### Population

- Dataflow `DF_ADNKS_T16` version `1.0`
- Turkish title: **İl, Yaş Grubu ve Cinsiyete Göre Nüfus**
- DSD `DSD_ADNKS` version `1.7`
- National selection: `IKAMET_YERI=_T` (Toplam)
- Sex codes: `1` (**Erkek**) and `2` (**Kadın**), from `CL_CINSIYET` 1.1
- Age codes: 5-year groups `Y0T4` through `Y85T89` and open group
  `Y_GE90`, from `CL_YAS` 1.1
- Measure: `ADNKS_GOSTERGE=NUFUS` (persons)
- Period filter: `startPeriod=2007&endPeriod=2025`

The full ordered key is recorded in the analysis provenance. The selected
groups have the same codes and labels in all 19 years. These are ADNKS resident
population observations, not projections. No classification bridge is applied.

## Retrieval, revisions, and validation

Acquisition uses exactly two bounded data requests, sequentially, with a
one-second pause. The generator rejects schema drift, duplicate cells,
unexpected categories, invalid values, missing years, and any population status
flag it has not been programmed to interpret. Fertility `OBS_STATUS` is retained:
2020–2024 are currently `R` (**Revize edilmiş değer**); other displayed years
are `A` (**Normal değer**).

The 2025 age-sex cells sum to 86,092,168, matching the independent all-sex,
all-age 2025 observation in dataflow `DF_ADNKS_T27` version `1.1`. The current
2024 fertility value is 1.4881566134354 and flagged revised; an earlier TÜİK
Birth Statistics 2024 headline reported 1.48. Because the vintages differ, this
is documented as a revision discrepancy rather than forced into an exact match.

## Rights, limits, and unresolved gates

The inspected portal and SDMX documentation did not state data-specific
licensing, output-redistribution permission, formal revision timing, or a numeric
API rate limit. Those remain unresolved gates. Public accessibility is not
treated as permission to redistribute raw responses. Consequently the exact raw
bytes are not committed: `provenance.json` records each resolved URL, retrieval
timestamp, HTTP media type, byte count, message metadata and SHA-256 checksum,
and `go run ./cmd/fertilitypyramid -fetch` repeats the same narrow requests.

No output is approved for publication until a maintainer verifies the rights
conclusion, the exact methodological definition, and the revision discrepancy.

## Presentation font and renderer

The font is a presentation dependency, not a statistical source. To avoid an
unreproducible lookup of machine-local Arial, the generator embeds the following
repository-contained asset:

- Family/style: Roboto Regular
- Upstream: the official Google Fonts `googlefonts/roboto-2` repository
- Pinned commit: `38062f4b4a0be4346d07a928408da21602545e9e`
- Resolved file URL:
  <https://raw.githubusercontent.com/googlefonts/roboto-2/38062f4b4a0be4346d07a928408da21602545e9e/src/hinted/Roboto-Regular.ttf>
- Size: 515,100 bytes
- SHA-256:
  `56a45233d29f11b4dfb86d248e921939d115778f87325e7ae8cc108383d6664d`
- License: Apache License 2.0; the exact upstream license response is included at
  `cmd/fertilitypyramid/assets/Roboto-LICENSE.txt` (11,357 bytes; SHA-256
  `c71d239df91726fc519c6eb72d318ec65820627232b2f796219e87dcf35d0ab4`).
  Its pinned source URL is
  <https://raw.githubusercontent.com/googlefonts/roboto-2/38062f4b4a0be4346d07a928408da21602545e9e/LICENSE>.

The archived upstream repository identifies itself as the Roboto family and
publishes the Apache-2.0 license. That license permits redistribution subject to
including the license text, which this repository does. The only direct Go
rendering dependency is pinned `golang.org/x/image` v0.46.0; its OpenType parser
brings pinned `golang.org/x/text` v0.42.0 and `golang.org/x/sys` v0.48.0
transitively. All three modules carry BSD-3-Clause licenses, and their module
checksums are in `go.sum`. Glyph hinting is disabled, the TTF bytes are compiled
into the generator, and no runtime network or host-font discovery occurs. The
generator verifies the embedded TTF checksum before rendering.
