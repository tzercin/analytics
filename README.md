# analytics

Go tools for reproducible analyses using authoritative public data.

## TÜİK SDMX client

The `tuik` package is a small, dependency-free client for the authenticated
TÜİK SDMX REST API documented by TÜİK:

- [TÜİK SDMX Web Service Documentation](https://veriportali.tuik.gov.tr/tr/sdmx-web-service-documentation)
- [SDMX REST API specification 1.5](https://github.com/sdmx-twg/sdmx-rest/tree/v1.5.0)

It obtains and caches access tokens, adds bearer authentication, constructs
structural-metadata and data requests, and returns successful response bodies
as streams. It decodes SDMX-ML 2.1 Generic Data while preserving keys,
attributes, and source status codes. Parsing other representations remains with
callers until an analysis establishes that they are needed.

```go
tokens, err := tuik.NewAPIKeyTokenSource(apiKey)
if err != nil {
	log.Fatal(err)
}

client, err := tuik.NewClient(tokens)
if err != nil {
	log.Fatal(err)
}

response, err := client.Structure(ctx, tuik.StructureQuery{
	Resource: tuik.Dataflows,
	Agency:   "TR",
	ID:       "all",
	Version:  "latest",
	Detail:   "full",
})
if err != nil {
	log.Fatal(err)
}
defer response.Body.Close()
```

API keys, client secrets, and access tokens must be supplied at runtime. Do not
put them in source files, command-line arguments, logs, or the public repository.

Run the package checks with:

```sh
go test ./...
go vet ./...
```

No TÜİK dataset, table, series, or revision vintage is selected by this
package itself. Those remain analysis-specific provenance decisions.

## First analysis

[`analyses/turkey-cpi-yoy-10y`](analyses/turkey-cpi-yoy-10y) retrieves exactly
one TÜİK headline CPI series and generates a validated CSV, SVG chart, and
provenance manifest for September 2016–August 2026. It is a local review draft;
no output has been published.

## Passenger-car transfers vs inflation

[`analyses/turkey-used-car-transfers-vs-inflation`](analyses/turkey-used-car-transfers-vs-inflation)
answers a second-hand car market question with the narrow measure TÜİK actually
publishes: notarized passenger-car transfers as an explicitly labelled proxy,
compared with headline CPI inflation from January 2010 through July 2026. It
produces validated monthly and annual CSV files, a machine-readable summary, an
SVG chart, and a detailed provenance manifest. It is a local review draft; no
output has been published.

## Housing and car-market activity

[`analyses/turkey-housing-and-car-sales`](analyses/turkey-housing-and-car-sales)
contains separate monthly housing-sales and passenger-car-transfer charts for
January 2013–July 2026, with TÜİK SDMX provenance and a TCMB EVDS catalog
cross-check.
