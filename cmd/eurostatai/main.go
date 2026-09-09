// Command eurostatai acquires and renders a fixed Eurostat comparison of
// enterprise use of AI text-mining technology in 2024 and 2025.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html"
	"io"
	"log"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/tzercin/analytics/eurostat"
)

const (
	datasetCode  = "isoc_eb_ai"
	datasetTitle = "Artificial intelligence by size class of enterprise"
	frequency    = "A"
	indicator    = "E_AI_TTM"
	unit         = "PC_ENT"
	nace         = "C10-S951_X_K"
	startYear    = "2024"
	endYear      = "2025"
	rowWant      = 232 // 29 geographies × 4 size classes × 2 years.
)

var (
	expectedDimensions = []string{"freq", "size_emp", "nace_r2", "indic_is", "unit", "geo", "time"}
	years              = []string{startYear, endYear}
	sizes              = []sizeSpec{
		{Code: "GE10", Label: "10 persons employed or more", Short: "All enterprises"},
		{Code: "10-49", Label: "From 10 to 49 persons employed", Short: "Small (10–49)"},
		{Code: "50-249", Label: "From 50 to 249 persons employed", Short: "Medium (50–249)"},
		{Code: "GE250", Label: "250 persons employed or more", Short: "Large (250+)"},
	}
	geographies = []geoSpec{
		{Code: "EU27_2020", Role: "eu27_aggregate"},
		{Code: "BE", Role: "eu27_member"}, {Code: "BG", Role: "eu27_member"},
		{Code: "CZ", Role: "eu27_member"}, {Code: "DK", Role: "eu27_member"},
		{Code: "DE", Role: "eu27_member"}, {Code: "EE", Role: "eu27_member"},
		{Code: "IE", Role: "eu27_member"}, {Code: "EL", Role: "eu27_member"},
		{Code: "ES", Role: "eu27_member"}, {Code: "FR", Role: "eu27_member"},
		{Code: "HR", Role: "eu27_member"}, {Code: "IT", Role: "eu27_member"},
		{Code: "CY", Role: "eu27_member"}, {Code: "LV", Role: "eu27_member"},
		{Code: "LT", Role: "eu27_member"}, {Code: "LU", Role: "eu27_member"},
		{Code: "HU", Role: "eu27_member"}, {Code: "MT", Role: "eu27_member"},
		{Code: "NL", Role: "eu27_member"}, {Code: "AT", Role: "eu27_member"},
		{Code: "PL", Role: "eu27_member"}, {Code: "PT", Role: "eu27_member"},
		{Code: "RO", Role: "eu27_member"}, {Code: "SI", Role: "eu27_member"},
		{Code: "SK", Role: "eu27_member"}, {Code: "FI", Role: "eu27_member"},
		{Code: "SE", Role: "eu27_member"},
		{Code: "TR", Role: "comparison_country"},
	}
)

type sizeSpec struct {
	Code  string
	Label string
	Short string
}

type geoSpec struct {
	Code string `json:"code"`
	Role string `json:"role"`
}

type observation struct {
	GeoCode        string
	GeoLabel       string
	GeoRole        string
	Year           string
	SizeCode       string
	SizeLabel      string
	Value          float64
	RawValue       string
	Status         string
	StatusLabel    string
	IndicatorLabel string
	UnitLabel      string
	NACELabel      string
}

type countrySummary struct {
	GeoCode             string
	GeoLabel            string
	GeoRole             string
	All2024             float64
	All2025             float64
	Change              float64
	Small2025           float64
	Medium2025          float64
	Large2025           float64
	LargeMinusSmall2025 float64
	StatusFlags         string
}

type acquisition struct {
	RequestURL          string `json:"request_url"`
	RetrievedAt         string `json:"retrieved_at"`
	ResponseDate        string `json:"response_date,omitempty"`
	ResponseContentType string `json:"response_content_type"`
	RawByteCount        int    `json:"raw_byte_count"`
	RawSHA256           string `json:"raw_sha256"`
	DatasetUpdated      string `json:"dataset_updated"`
}

type provenance struct {
	Status                  string                      `json:"status"`
	Question                string                      `json:"question"`
	SourceOwner             string                      `json:"source_owner"`
	SourceType              string                      `json:"source_type"`
	DatasetTitle            string                      `json:"dataset_title"`
	DatasetCode             string                      `json:"dataset_code"`
	DatasetDOI              string                      `json:"dataset_doi"`
	DatasetVersion          string                      `json:"dataset_version"`
	DataStructureID         string                      `json:"data_structure_id"`
	DataStructureVersion    string                      `json:"data_structure_version"`
	LandingPage             string                      `json:"landing_page"`
	APIDocumentation        string                      `json:"api_documentation"`
	MethodologyURL          string                      `json:"methodology_url"`
	ResultsReportURL        string                      `json:"results_report_url"`
	ReuseURL                string                      `json:"reuse_url"`
	RevisionPolicyURL       string                      `json:"revision_policy_url"`
	ReuseConclusion         string                      `json:"reuse_conclusion"`
	Request                 acquisition                 `json:"request"`
	DatasetUpdated          string                      `json:"dataset_updated"`
	StructureUpdated        string                      `json:"structure_updated"`
	Dimensions              []dimensionSelection        `json:"dimensions"`
	IncludedGeographies     []geoSpec                   `json:"included_geographies"`
	ObservationCount        int                         `json:"observation_count"`
	MissingObservationCount int                         `json:"missing_observation_count"`
	StatusCounts            map[string]int              `json:"status_counts"`
	StatusLabels            map[string]string           `json:"status_labels"`
	MultiplierMetadata      string                      `json:"multiplier_metadata"`
	DecimalsMetadata        string                      `json:"decimals_metadata"`
	Calculations            map[string]string           `json:"calculations"`
	CrossChecks             []string                    `json:"cross_checks"`
	Validation              []string                    `json:"validation"`
	Limitations             []string                    `json:"limitations"`
	TransformationCommand   string                      `json:"transformation_command"`
	RefreshCommand          string                      `json:"refresh_command"`
	GoVersion               string                      `json:"go_version"`
	Outputs                 map[string]outputProvenance `json:"outputs"`
	UnresolvedGates         []string                    `json:"unresolved_gates"`
	ReviewGate              string                      `json:"review_gate"`
}

type dimensionSelection struct {
	ID       string            `json:"id"`
	Label    string            `json:"label"`
	Selected map[string]string `json:"selected_codes"`
}

type outputProvenance struct {
	Layer  string `json:"layer"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

type config struct {
	OutputDir string
	Fetch     bool
	BaseURL   string
}

func main() {
	output := flag.String("output", "analyses/eurostat-enterprise-ai-text-mining", "analysis output directory")
	fetch := flag.Bool("fetch", false, "refresh the authoritative raw snapshot before transforming")
	baseURL := flag.String("base-url", eurostat.DefaultBaseURL, "Eurostat statistics API data base URL")
	flag.Parse()
	if err := run(config{OutputDir: *output, Fetch: *fetch, BaseURL: *baseURL}); err != nil {
		log.Fatal(err)
	}
}

func run(configuration config) error {
	if configuration.OutputDir == "" {
		return errors.New("output directory is required")
	}
	rawPath := filepath.Join(configuration.OutputDir, "raw", "isoc_eb_ai_E_AI_TTM_2024_2025.json")
	receiptPath := filepath.Join(configuration.OutputDir, "raw", "acquisition.json")
	client := eurostat.NewClient()
	client.BaseURL = configuration.BaseURL
	requestURL, err := client.DataURL(datasetCode, queryFilters())
	if err != nil {
		return err
	}
	if configuration.Fetch {
		if err := fetchSnapshot(client, requestURL, rawPath, receiptPath); err != nil {
			return err
		}
	}
	raw, receipt, err := readSnapshot(rawPath, receiptPath, requestURL)
	if err != nil {
		if os.IsNotExist(err) && !configuration.Fetch {
			return fmt.Errorf("raw snapshot not found; run with -fetch once: %w", err)
		}
		return err
	}
	dataset, err := eurostat.DecodeDataset(raw)
	if err != nil {
		return err
	}
	if err := validateMetadata(dataset); err != nil {
		return err
	}
	observations, err := extractObservations(dataset)
	if err != nil {
		return err
	}
	summaries, err := summarize(observations)
	if err != nil {
		return err
	}
	if err := reconcilePublishedEU(summaries); err != nil {
		return err
	}

	if err := os.MkdirAll(configuration.OutputDir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	observationsPath := filepath.Join(configuration.OutputDir, "observations.csv")
	summaryPath := filepath.Join(configuration.OutputDir, "country-summary.csv")
	chartPath := filepath.Join(configuration.OutputDir, "text-mining-adoption.svg")
	readmePath := filepath.Join(configuration.OutputDir, "README.md")
	if err := writeObservationsCSV(observationsPath, observations); err != nil {
		return err
	}
	if err := writeSummaryCSV(summaryPath, summaries); err != nil {
		return err
	}
	if err := writeSVG(chartPath, summaries, receipt); err != nil {
		return err
	}
	if err := writeReadme(readmePath, summaries, receipt, requestURL); err != nil {
		return err
	}

	outputs := make(map[string]outputProvenance)
	for _, item := range []struct{ path, layer string }{
		{rawPath, "raw"}, {receiptPath, "raw metadata"},
		{observationsPath, "derived"}, {summaryPath, "derived"},
		{chartPath, "presentation"}, {readmePath, "presentation"},
	} {
		info, err := fileProvenance(item.path, item.layer)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(configuration.OutputDir, item.path)
		if err != nil {
			return err
		}
		outputs[filepath.ToSlash(relative)] = info
	}
	manifest, err := makeProvenance(dataset, receipt, observations, outputs)
	if err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(configuration.OutputDir, "provenance.json"), manifest); err != nil {
		return err
	}
	fmt.Printf("wrote %d validated observations for %d geographies to %s\n", len(observations), len(summaries), configuration.OutputDir)
	return nil
}

func queryFilters() url.Values {
	return url.Values{
		"freq":            {frequency},
		"indic_is":        {indicator},
		"lang":            {"en"},
		"nace_r2":         {nace},
		"sinceTimePeriod": {startYear},
		"unit":            {unit},
		"untilTimePeriod": {endYear},
	}
}

func fetchSnapshot(client *eurostat.Client, requestURL, rawPath, receiptPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	response, err := client.Fetch(ctx, requestURL)
	if err != nil {
		return err
	}
	dataset, err := eurostat.DecodeDataset(response.Body)
	if err != nil {
		return fmt.Errorf("refuse to store invalid raw response: %w", err)
	}
	if err := validateMetadata(dataset); err != nil {
		return fmt.Errorf("refuse to store schema-drifted raw response: %w", err)
	}
	location, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		return err
	}
	receipt := acquisition{
		RequestURL:          response.URL,
		RetrievedAt:         response.RetrievedAt.In(location).Format(time.RFC3339),
		ResponseDate:        response.Date,
		ResponseContentType: response.ContentType,
		RawByteCount:        len(response.Body),
		RawSHA256:           bytesSHA256(response.Body),
		DatasetUpdated:      dataset.Updated,
	}
	if err := os.MkdirAll(filepath.Dir(rawPath), 0o755); err != nil {
		return fmt.Errorf("create raw directory: %w", err)
	}
	if err := writeAtomic(rawPath, func(writer io.Writer) error {
		_, err := writer.Write(response.Body)
		return err
	}); err != nil {
		return err
	}
	return writeJSON(receiptPath, receipt)
}

func readSnapshot(rawPath, receiptPath, expectedURL string) ([]byte, acquisition, error) {
	raw, err := os.ReadFile(rawPath)
	if err != nil {
		return nil, acquisition{}, err
	}
	receiptBytes, err := os.ReadFile(receiptPath)
	if err != nil {
		return nil, acquisition{}, err
	}
	var receipt acquisition
	decoder := json.NewDecoder(strings.NewReader(string(receiptBytes)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		return nil, acquisition{}, fmt.Errorf("decode acquisition receipt: %w", err)
	}
	if receipt.RequestURL != expectedURL {
		return nil, acquisition{}, fmt.Errorf("receipt URL %q does not match query %q", receipt.RequestURL, expectedURL)
	}
	if receipt.RawByteCount != len(raw) || receipt.RawSHA256 != bytesSHA256(raw) {
		return nil, acquisition{}, fmt.Errorf("raw snapshot does not match acquisition receipt")
	}
	if !strings.HasPrefix(strings.ToLower(receipt.ResponseContentType), "application/json") {
		return nil, acquisition{}, fmt.Errorf("unexpected response media type %q", receipt.ResponseContentType)
	}
	if _, err := time.Parse(time.RFC3339, receipt.RetrievedAt); err != nil {
		return nil, acquisition{}, fmt.Errorf("invalid retrieval timestamp: %w", err)
	}
	return raw, receipt, nil
}

func validateMetadata(dataset *eurostat.Dataset) error {
	if dataset.Label != datasetTitle || dataset.Source != "ESTAT" {
		return fmt.Errorf("unexpected dataset label/source %q/%q", dataset.Label, dataset.Source)
	}
	if !equalStrings(dataset.ID, expectedDimensions) {
		return fmt.Errorf("dimension order is %v, want %v", dataset.ID, expectedDimensions)
	}
	if dataset.Extension.ID != "ISOC_EB_AI" || dataset.Extension.AgencyID != "ESTAT" || dataset.Extension.Version != "1.0" {
		return fmt.Errorf("unexpected dataflow identity %#v", dataset.Extension)
	}
	if dataset.Extension.DataStructure.ID != "ISOC_EB_AI" || dataset.Extension.DataStructure.AgencyID != "ESTAT" || dataset.Extension.DataStructure.Version == "" {
		return fmt.Errorf("unexpected data structure identity %#v", dataset.Extension.DataStructure)
	}
	expectedSingles := map[string]map[string]string{
		"freq":     {frequency: "Annual"},
		"nace_r2":  {nace: "All activities (except agriculture, forestry and fishing, and mining and quarrying), without financial sector"},
		"indic_is": {indicator: "Enterprises using AI technologies performing analysis of written language (text mining)"},
		"unit":     {unit: "Percentage of enterprises"},
		"time":     {startYear: startYear, endYear: endYear},
	}
	for dimension, wanted := range expectedSingles {
		codes, err := dataset.Codes(dimension)
		if err != nil {
			return err
		}
		if len(codes) != len(wanted) {
			return fmt.Errorf("dimension %s contains %v, want exactly selected codes", dimension, codes)
		}
		for code, label := range wanted {
			if dataset.Dimension[dimension].Category.Label[code] != label {
				return fmt.Errorf("dimension %s code %s label drifted: %q", dimension, code, dataset.Dimension[dimension].Category.Label[code])
			}
		}
	}
	for _, size := range sizes {
		if dataset.Dimension["size_emp"].Category.Label[size.Code] != size.Label {
			return fmt.Errorf("size code %s missing or label drifted", size.Code)
		}
	}
	for _, geography := range geographies {
		if dataset.Dimension["geo"].Category.Label[geography.Code] == "" {
			return fmt.Errorf("required geography code %s missing", geography.Code)
		}
	}
	if annotationDate(dataset, "UPDATE_DATA") != dataset.Updated {
		return fmt.Errorf("dataset updated timestamp does not match UPDATE_DATA annotation")
	}
	return nil
}

func extractObservations(dataset *eurostat.Dataset) ([]observation, error) {
	result := make([]observation, 0, rowWant)
	seen := make(map[string]bool, rowWant)
	for _, geography := range geographies {
		geoLabel := dataset.Dimension["geo"].Category.Label[geography.Code]
		for _, size := range sizes {
			for _, year := range years {
				coordinates := map[string]string{
					"freq": frequency, "size_emp": size.Code, "nace_r2": nace,
					"indic_is": indicator, "unit": unit, "geo": geography.Code, "time": year,
				}
				cell, err := dataset.Cell(coordinates)
				if err != nil {
					return nil, err
				}
				key := geography.Code + "|" + size.Code + "|" + year
				if seen[key] {
					return nil, fmt.Errorf("duplicate observation key %s", key)
				}
				seen[key] = true
				if cell.Value == nil {
					return nil, fmt.Errorf("selected observation %s is missing (status %q: %s)", key, cell.Status, cell.StatusLabel)
				}
				if *cell.Value < 0 || *cell.Value > 100 {
					return nil, fmt.Errorf("selected observation %s is outside percentage range: %g", key, *cell.Value)
				}
				result = append(result, observation{
					GeoCode: geography.Code, GeoLabel: geoLabel, GeoRole: geography.Role,
					Year: year, SizeCode: size.Code, SizeLabel: size.Label,
					Value: *cell.Value, RawValue: cell.RawValue,
					Status: cell.Status, StatusLabel: cell.StatusLabel,
					IndicatorLabel: dataset.Dimension["indic_is"].Category.Label[indicator],
					UnitLabel:      dataset.Dimension["unit"].Category.Label[unit],
					NACELabel:      dataset.Dimension["nace_r2"].Category.Label[nace],
				})
			}
		}
	}
	if len(result) != rowWant {
		return nil, fmt.Errorf("got %d selected observations, want %d", len(result), rowWant)
	}
	return result, nil
}

func summarize(observations []observation) ([]countrySummary, error) {
	byKey := make(map[string]observation, len(observations))
	for _, item := range observations {
		key := item.GeoCode + "|" + item.SizeCode + "|" + item.Year
		if _, exists := byKey[key]; exists {
			return nil, fmt.Errorf("duplicate summary input %s", key)
		}
		byKey[key] = item
	}
	result := make([]countrySummary, 0, len(geographies))
	for _, geography := range geographies {
		get := func(size, year string) (observation, error) {
			item, ok := byKey[geography.Code+"|"+size+"|"+year]
			if !ok {
				return observation{}, fmt.Errorf("summary cell missing for %s/%s/%s", geography.Code, size, year)
			}
			return item, nil
		}
		all2024, err := get("GE10", "2024")
		if err != nil {
			return nil, err
		}
		all2025, err := get("GE10", "2025")
		if err != nil {
			return nil, err
		}
		small, err := get("10-49", "2025")
		if err != nil {
			return nil, err
		}
		medium, err := get("50-249", "2025")
		if err != nil {
			return nil, err
		}
		large, err := get("GE250", "2025")
		if err != nil {
			return nil, err
		}
		flags := uniqueStatuses(all2024, all2025, small, medium, large)
		result = append(result, countrySummary{
			GeoCode: geography.Code, GeoLabel: all2025.GeoLabel, GeoRole: geography.Role,
			All2024: all2024.Value, All2025: all2025.Value, Change: all2025.Value - all2024.Value,
			Small2025: small.Value, Medium2025: medium.Value, Large2025: large.Value,
			LargeMinusSmall2025: large.Value - small.Value, StatusFlags: flags,
		})
	}
	return result, nil
}

func reconcilePublishedEU(summaries []countrySummary) error {
	eu, err := findSummary(summaries, "EU27_2020")
	if err != nil {
		return err
	}
	checks := []struct {
		name         string
		got, rounded float64
	}{
		{"all 2024", eu.All2024, 6.9}, {"all 2025", eu.All2025, 11.8},
		{"small 2025", eu.Small2025, 9.9}, {"medium 2025", eu.Medium2025, 18.2},
		{"large 2025", eu.Large2025, 35.0},
	}
	for _, check := range checks {
		if math.Abs(math.Round(check.got*10)/10-check.rounded) > 0.000001 {
			return fmt.Errorf("EU %s %.2f does not reconcile to published rounded value %.1f", check.name, check.got, check.rounded)
		}
	}
	return nil
}

func writeObservationsCSV(path string, observations []observation) error {
	return writeAtomic(path, func(writer io.Writer) error {
		csvWriter := csv.NewWriter(writer)
		header := []string{"geo_code", "geo_label", "geo_role", "year", "size_code", "size_label", "indicator_code", "indicator_label", "nace_r2_code", "nace_r2_label", "unit_code", "unit_label", "value_percent", "status", "status_label"}
		if err := csvWriter.Write(header); err != nil {
			return err
		}
		for _, item := range observations {
			row := []string{item.GeoCode, item.GeoLabel, item.GeoRole, item.Year, item.SizeCode, item.SizeLabel,
				indicator, item.IndicatorLabel, nace, item.NACELabel, unit, item.UnitLabel,
				item.RawValue, item.Status, item.StatusLabel}
			if err := csvWriter.Write(row); err != nil {
				return err
			}
		}
		csvWriter.Flush()
		return csvWriter.Error()
	})
}

func writeSummaryCSV(path string, summaries []countrySummary) error {
	return writeAtomic(path, func(writer io.Writer) error {
		csvWriter := csv.NewWriter(writer)
		header := []string{"geo_code", "geo_label", "geo_role", "all_enterprises_2024_percent", "all_enterprises_2025_percent", "change_2024_2025_percentage_points", "small_2025_percent", "medium_2025_percent", "large_2025_percent", "large_minus_small_2025_percentage_points", "source_status_flags"}
		if err := csvWriter.Write(header); err != nil {
			return err
		}
		for _, item := range summaries {
			row := []string{item.GeoCode, item.GeoLabel, item.GeoRole,
				format2(item.All2024), format2(item.All2025), format2(item.Change),
				format2(item.Small2025), format2(item.Medium2025), format2(item.Large2025),
				format2(item.LargeMinusSmall2025), item.StatusFlags}
			if err := csvWriter.Write(row); err != nil {
				return err
			}
		}
		csvWriter.Flush()
		return csvWriter.Error()
	})
}

func writeSVG(path string, summaries []countrySummary, receipt acquisition) error {
	eu, err := findSummary(summaries, "EU27_2020")
	if err != nil {
		return err
	}
	for _, item := range summaries {
		if item.All2024 > 35 || item.All2025 > 35 || item.Small2025 > 70 || item.Medium2025 > 70 || item.Large2025 > 70 {
			return fmt.Errorf("chart scale no longer covers %s; review the presentation", item.GeoCode)
		}
	}
	countries := append([]countrySummary(nil), summaries[1:]...)
	sort.SliceStable(countries, func(i, j int) bool {
		if countries[i].All2025 == countries[j].All2025 {
			return countries[i].GeoLabel < countries[j].GeoLabel
		}
		return countries[i].All2025 > countries[j].All2025
	})
	rows := append([]countrySummary{eu}, countries...)
	const width, height = 1500.0, 1640.0
	const labelX, top, rowHeight = 190.0, 245.0, 43.0
	const leftA, rightA = 230.0, 720.0
	const leftB, rightB = 880.0, 1450.0
	xA := func(value float64) float64 { return leftA + value/35*(rightA-leftA) }
	xB := func(value float64) float64 { return leftB + value/70*(rightB-leftB) }
	return writeAtomic(path, func(writer io.Writer) error {
		writef := func(format string, args ...any) { _, _ = fmt.Fprintf(writer, format, args...) }
		writef(`<svg xmlns="http://www.w3.org/2000/svg" width="1500" height="1640" viewBox="0 0 1500 1640" role="img" aria-labelledby="title desc">` + "\n")
		writef(`<title id="title">Enterprise use of AI text mining in the EU-27 and Türkiye, 2024 to 2025</title>` + "\n")
		writef(`<desc id="desc">Two aligned dot plots. The first compares 2024 and 2025 percentages for enterprises with at least 10 persons employed. The second compares small, medium and large enterprises in 2025. Countries are sorted by the 2025 all-enterprise share. Exact values are in the adjacent CSV files.</desc>` + "\n")
		writef(`<rect width="1500" height="1640" fill="#fbfaf7"/>` + "\n")
		writef(`<text x="55" y="55" font-family="Inter,system-ui,sans-serif" font-size="30" font-weight="700" fill="#17202a">Enterprise use of AI text mining</text>` + "\n")
		writef(`<text x="55" y="91" font-family="Inter,system-ui,sans-serif" font-size="17" fill="#4e5a65">Share using AI to analyse written language • EU-27 members and Türkiye • percent of enterprises</text>` + "\n")
		writef(`<text x="230" y="155" font-family="Inter,system-ui,sans-serif" font-size="20" font-weight="700" fill="#17202a">A. All enterprises (10+), 2024 → 2025</text>` + "\n")
		writef(`<text x="880" y="155" font-family="Inter,system-ui,sans-serif" font-size="20" font-weight="700" fill="#17202a">B. Enterprise size, 2025</text>` + "\n")
		writef(`<g font-family="Inter,system-ui,sans-serif" font-size="13" fill="#58636f">`)
		writef(`<circle cx="245" cy="190" r="6" fill="#fbfaf7" stroke="#66737f" stroke-width="2"/><text x="258" y="195">2024</text>`)
		writef(`<circle cx="325" cy="190" r="6" fill="#176b87"/><text x="338" y="195">2025</text>`)
		writef(`<circle cx="895" cy="190" r="6" fill="#d18b00"/><text x="908" y="195">Small</text>`)
		writef(`<path d="M974 184 l7 7 l-7 7 l-7 -7 Z" fill="#7456a4"/><text x="988" y="195">Medium</text>`)
		writef(`<rect x="1063" y="184" width="12" height="12" fill="#ba3d54"/><text x="1082" y="195">Large</text></g>` + "\n")
		for tick := 0.0; tick <= 30; tick += 10 {
			x := xA(tick)
			writef(`<line x1="%.1f" y1="218" x2="%.1f" y2="1500" stroke="#e1e5e8"/><text x="%.1f" y="215" text-anchor="middle" font-family="Inter,system-ui,sans-serif" font-size="12" fill="#66737f">%.0f%%</text>`+"\n", x, x, x, tick)
		}
		for tick := 0.0; tick <= 70; tick += 10 {
			x := xB(tick)
			writef(`<line x1="%.1f" y1="218" x2="%.1f" y2="1500" stroke="#e1e5e8"/><text x="%.1f" y="215" text-anchor="middle" font-family="Inter,system-ui,sans-serif" font-size="12" fill="#66737f">%.0f%%</text>`+"\n", x, x, x, tick)
		}
		for index, row := range rows {
			y := top + float64(index)*rowHeight
			if index == 1 {
				writef(`<line x1="55" y1="%.1f" x2="1450" y2="%.1f" stroke="#aeb7be" stroke-width="2"/>`+"\n", y-rowHeight/2, y-rowHeight/2)
			}
			labelColor, weight := "#26313a", "400"
			if row.GeoCode == "EU27_2020" {
				weight = "700"
			}
			if row.GeoCode == "TR" {
				labelColor, weight = "#9b4d00", "700"
			}
			writef(`<g><title>%s: all enterprises %.2f%% in 2024 and %.2f%% in 2025; small %.2f%%, medium %.2f%% and large %.2f%% in 2025.</title>`+"\n", html.EscapeString(row.GeoLabel), row.All2024, row.All2025, row.Small2025, row.Medium2025, row.Large2025)
			writef(`<text x="%.1f" y="%.1f" text-anchor="end" font-family="Inter,system-ui,sans-serif" font-size="14" font-weight="%s" fill="%s">%s</text>`+"\n", labelX, y+5, weight, labelColor, html.EscapeString(shortGeoLabel(row)))
			writef(`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#87939d" stroke-width="2"/>`, xA(row.All2024), y, xA(row.All2025), y)
			writef(`<circle cx="%.1f" cy="%.1f" r="6" fill="#fbfaf7" stroke="#66737f" stroke-width="2"/>`, xA(row.All2024), y)
			writef(`<circle cx="%.1f" cy="%.1f" r="6" fill="#176b87"/>`, xA(row.All2025), y)
			writef(`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#a7afb6" stroke-width="2"/>`, xB(row.Small2025), y, xB(row.Large2025), y)
			writef(`<circle cx="%.1f" cy="%.1f" r="6" fill="#d18b00"/>`, xB(row.Small2025), y)
			writef(`<path d="M%.1f %.1f l7 7 l-7 7 l-7 -7 Z" fill="#7456a4"/>`, xB(row.Medium2025), y-7)
			writef(`<rect x="%.1f" y="%.1f" width="12" height="12" fill="#ba3d54"/></g>`+"\n", xB(row.Large2025)-6, y-6)
		}
		writef(`<text x="55" y="1555" font-family="Inter,system-ui,sans-serif" font-size="12" fill="#606b75">Source: Eurostat isoc_eb_ai, E_AI_TTM / PC_ENT; retrieved %s. EU aggregate shown as context; country rows sorted by 2025 all-enterprise share.</text>`+"\n", html.EscapeString(receipt.RetrievedAt[:10]))
		writef(`<text x="55" y="1580" font-family="Inter,system-ui,sans-serif" font-size="12" fill="#606b75">Covered activities: NACE Rev. 2 C–J, L–N and group 95.1; financial sector excluded. Values are survey estimates, not enterprise counts.</text>` + "\n")
		writef(`<text x="55" y="1605" font-family="Inter,system-ui,sans-serif" font-size="11" fill="#78828b">Modified analysis; Eurostat is not responsible for this presentation. Exact values, source flags and methods: observations.csv and README.md.</text>` + "\n")
		writef(`</svg>` + "\n")
		return nil
	})
}

func writeReadme(path string, summaries []countrySummary, receipt acquisition, requestURL string) error {
	eu, err := findSummary(summaries, "EU27_2020")
	if err != nil {
		return err
	}
	tr, err := findSummary(summaries, "TR")
	if err != nil {
		return err
	}
	countries := append([]countrySummary(nil), summaries[1:]...)
	byLevel := append([]countrySummary(nil), countries...)
	sort.SliceStable(byLevel, func(i, j int) bool { return byLevel[i].All2025 > byLevel[j].All2025 })
	byChange := append([]countrySummary(nil), countries...)
	sort.SliceStable(byChange, func(i, j int) bool { return byChange[i].Change > byChange[j].Change })
	allIncreased := true
	allSizeOrdered := true
	for _, item := range countries {
		if item.Change <= 0 {
			allIncreased = false
		}
		if !(item.Small2025 < item.Medium2025 && item.Medium2025 < item.Large2025) {
			allSizeOrdered = false
		}
	}
	if !allIncreased {
		return fmt.Errorf("finding drift: not every included country increased")
	}
	if !allSizeOrdered {
		return fmt.Errorf("finding drift: not every included country has large > medium > small")
	}
	return writeAtomic(path, func(writer io.Writer) error {
		writef := func(format string, args ...any) { _, _ = fmt.Fprintf(writer, format, args...) }
		writef("# Enterprise use of AI text mining: EU-27 and Türkiye, 2024–2025\n\n")
		writef("**Status:** Local analysis draft; not approved for publication  \n")
		writef("**Retrieved:** %s  \n**Eurostat data updated:** %s\n\n", receipt.RetrievedAt, receipt.DatasetUpdated)
		writef("## Question and frozen inclusion rules\n\n")
		writef("Across the EU-27 member countries and Türkiye, how did the share of enterprises using AI technology to **analyse written language (text mining)** change from 2024 to 2025, and how did the 2025 share differ between small, medium and large enterprises?\n\n")
		writef("The selection was fixed before interpreting results: annual frequency `A`; indicator `E_AI_TTM`; denominator `PC_ENT` (percentage of enterprises); NACE Rev. 2 aggregate `C10-S951_X_K`; years 2024 and 2025; sizes `GE10`, `10-49`, `50-249`, and `GE250`; the 27 members represented by `EU27_2020`, plus Türkiye as a separately labelled comparison. The EU aggregate is context and is not ranked as a country. This is one specific AI technology, **not** the broader “uses at least one AI technology” indicator.\n\n")
		writef("## Findings\n\n![AI text-mining adoption](text-mining-adoption.svg)\n\n")
		writef("- **Source fact:** Eurostat reports the EU-27 aggregate at **%.2f%% in 2024** and **%.2f%% in 2025** for enterprises with 10 or more persons employed. The 2026 Eurostat results report independently presents these as 6.9%% and 11.8%% after rounding.\n", eu.All2024, eu.All2025)
		writef("- **Calculation:** the EU-27 aggregate difference is **%+.2f percentage points**. All %d comparison countries (27 EU members plus Türkiye) have a positive published difference in this snapshot. The largest are %s (**%+.2f pp**), %s (**%+.2f pp**) and %s (**%+.2f pp**). This calculation describes two cross-sectional survey estimates; it does not track the same enterprises.\n", eu.Change, len(countries), byChange[0].GeoLabel, byChange[0].Change, byChange[1].GeoLabel, byChange[1].Change, byChange[2].GeoLabel, byChange[2].Change)
		writef("- **Calculation:** the highest 2025 all-enterprise shares are %s (**%.2f%%**), %s (**%.2f%%**) and %s (**%.2f%%**); the lowest are %s (**%.2f%%**), %s (**%.2f%%**) and %s (**%.2f%%**). Türkiye moves from **%.2f%%** to **%.2f%%** (**%+.2f pp**).\n", byLevel[0].GeoLabel, byLevel[0].All2025, byLevel[1].GeoLabel, byLevel[1].All2025, byLevel[2].GeoLabel, byLevel[2].All2025, byLevel[len(byLevel)-1].GeoLabel, byLevel[len(byLevel)-1].All2025, byLevel[len(byLevel)-2].GeoLabel, byLevel[len(byLevel)-2].All2025, byLevel[len(byLevel)-3].GeoLabel, byLevel[len(byLevel)-3].All2025, tr.All2024, tr.All2025, tr.Change)
		writef("- **Calculation:** in 2025 the EU-27 values are **%.2f%% small**, **%.2f%% medium** and **%.2f%% large**, a large-minus-small difference of **%.2f pp**. Every included country has large > medium > small in this snapshot. Türkiye's corresponding values are **%.2f%%**, **%.2f%%** and **%.2f%%** (difference **%.2f pp**). These size differences are descriptive, not causal effects of enterprise size.\n", eu.Small2025, eu.Medium2025, eu.Large2025, eu.LargeMinusSmall2025, tr.Small2025, tr.Medium2025, tr.Large2025, tr.LargeMinusSmall2025)
		writef("- **Inference:** none about causes, future adoption, statistical significance, or market value.\n\n")
		writef("## Authoritative source and exact query\n\n")
		writef("Eurostat is the publisher. The live JSON-stat response identifies dataset `ISOC_EB_AI` version `1.0`, title “%s”, DOI [10.2908/ISOC_EB_AI](https://doi.org/10.2908/ISOC_EB_AI), and embeds every selected code and label. Exact query:\n\n```text\n%s\n```\n\n", datasetTitle, requestURL)
		writef("See Eurostat's [Data Browser entry](https://ec.europa.eu/eurostat/databrowser/view/isoc_eb_ai/default/table?lang=en), [statistics API guide](https://ec.europa.eu/eurostat/web/user-guides/data-browser/api-data-access/api-getting-started/api), [enterprise ICT metadata](https://ec.europa.eu/eurostat/cache/metadata/en/isoc_e_esms.htm), and [2026 AI results report](https://ec.europa.eu/eurostat/web/products-statistical-reports/w/ks-01-26-009). Revision and reuse decisions are recorded in [`provenance.json`](provenance.json).\n\n")
		writef("## Outputs and reproduction\n\n- [`raw/isoc_eb_ai_E_AI_TTM_2024_2025.json`](raw/isoc_eb_ai_E_AI_TTM_2024_2025.json): exact API response bytes\n- [`raw/acquisition.json`](raw/acquisition.json): URL, HTTP metadata, retrieval time, byte count and raw SHA-256\n- [`observations.csv`](observations.csv): tidy selected observations with original codes, labels, values and status fields\n- [`country-summary.csv`](country-summary.csv): deterministic differences used in the prose/chart\n- [`text-mining-adoption.svg`](text-mining-adoption.svg): deterministic accessible chart\n- [`provenance.json`](provenance.json): complete machine-readable provenance and checksums\n\nReplay the committed snapshot without network access:\n\n```sh\ngo run ./cmd/eurostatai\n```\n\nExplicitly refresh it (one request, bounded retry/backoff/jitter, no credential):\n\n```sh\ngo run ./cmd/eurostatai -fetch\n```\n\nA refresh replaces the raw snapshot and receipt only after JSON-stat and selection metadata validate. Review all resulting vintage and value changes before accepting them.\n\n")
		writef("## Limitations\n\n- The annual enterprise ICT survey is self-reported and estimated from samples; this slice does not include standard errors, so no significance claim is made. National collection and non-response methods can differ within Eurostat's harmonised framework.\n- The denominator is enterprises in each size class, not employees, users, transactions, or all registered firms. The covered population has at least 10 employees or self-employed persons and covers NACE Rev. 2 sections C–J and L–N plus group 95.1; agriculture, forestry, fishing, mining, quarrying and finance are outside this aggregate.\n- Eurostat's model questionnaire is updated annually. `E_AI_TTM` has the same live code and label in both selected years and Eurostat publishes a direct comparison, but future metadata or national breaks can affect comparability. The broader at-least-one indicator changed scope in 2025 when picture/video/sound generation was added; that is why this analysis does not use it.\n- Values are current-database estimates and may be revised when national authorities submit corrections. The stored raw snapshot fixes this analysis vintage. Status flags are preserved; no selected cell in this snapshot is flagged or missing. Absence of a flag is not a precision guarantee.\n- Aggregate and country values are weighted survey estimates. Do not average country percentages, infer enterprise counts, or interpret size-class differences as causal. Türkiye is a comparison country and is not included in the EU-27 aggregate.\n- Enterprise use of one AI technology is not a count of AI vendors, AI-company revenue, model usage volume, productivity, or AI market size.\n\n")
		writef("## Unresolved gate\n\nEurostat's official statistics API guides checked for this vintage do not state a numeric request-per-second quota. The explicit refresh therefore makes one bounded query, retries only HTTP 429/transient 5xx responses, honours `Retry-After`, and otherwise applies the repository's conservative network policy. This does not block offline reproduction, but a maintainer should re-check the service contract before scheduling refreshes.\n\n")
		writef("Eurostat permits reuse of statistical data with acknowledgement. This repository stores a modified selection and presentation; Eurostat is not responsible for the analysis. Maintainer review of the question, reuse, calculations, caveats and artifacts is required before any publication.\n")
		return nil
	})
}

func makeProvenance(dataset *eurostat.Dataset, receipt acquisition, observations []observation, outputs map[string]outputProvenance) (provenance, error) {
	dimensions := make([]dimensionSelection, 0, len(expectedDimensions))
	selected := map[string][]string{
		"freq": {frequency}, "size_emp": {"GE10", "10-49", "50-249", "GE250"},
		"nace_r2": {nace}, "indic_is": {indicator}, "unit": {unit}, "geo": {}, "time": years,
	}
	for _, item := range geographies {
		selected["geo"] = append(selected["geo"], item.Code)
	}
	for _, id := range expectedDimensions {
		codes := make(map[string]string)
		for _, code := range selected[id] {
			codes[code] = dataset.Dimension[id].Category.Label[code]
		}
		dimensions = append(dimensions, dimensionSelection{ID: id, Label: dataset.Dimension[id].Label, Selected: codes})
	}
	statusCounts := make(map[string]int)
	missing := 0
	for _, item := range observations {
		if item.RawValue == "" {
			missing++
		}
		if item.Status != "" {
			statusCounts[item.Status]++
		}
	}
	return provenance{
		Status:      "local analysis draft; not approved for publication",
		Question:    "Across EU-27 member countries and Türkiye, how did enterprise use of AI text mining change from 2024 to 2025, and how did 2025 use differ by enterprise size?",
		SourceOwner: "Eurostat", SourceType: "primary official survey dataset",
		DatasetTitle: dataset.Label, DatasetCode: datasetCode, DatasetDOI: "10.2908/ISOC_EB_AI",
		DatasetVersion: dataset.Extension.Version, DataStructureID: dataset.Extension.DataStructure.ID,
		DataStructureVersion: dataset.Extension.DataStructure.Version,
		LandingPage:          "https://ec.europa.eu/eurostat/databrowser/view/isoc_eb_ai/default/table?lang=en",
		APIDocumentation:     "https://ec.europa.eu/eurostat/web/user-guides/data-browser/api-data-access/api-getting-started/api",
		MethodologyURL:       "https://ec.europa.eu/eurostat/cache/metadata/en/isoc_e_esms.htm",
		ResultsReportURL:     "https://ec.europa.eu/eurostat/web/products-statistical-reports/w/ks-01-26-009",
		ReuseURL:             "https://ec.europa.eu/eurostat/help/copyright-notice",
		RevisionPolicyURL:    "https://ec.europa.eu/eurostat/data/data-revision-policy",
		ReuseConclusion:      "Eurostat statistical data and metadata may be reused commercially or non-commercially with source acknowledgement; modified data/presentation must be identified and carry a non-responsibility disclaimer. No third-party material is included.",
		Request:              receipt, DatasetUpdated: dataset.Updated, StructureUpdated: annotationDate(dataset, "UPDATE_STRUCTURE"),
		Dimensions: dimensions, IncludedGeographies: append([]geoSpec(nil), geographies...),
		ObservationCount: len(observations), MissingObservationCount: missing,
		StatusCounts: statusCounts, StatusLabels: cloneMap(dataset.Extension.Status.Label),
		MultiplierMetadata: "not supplied by this JSON-stat response; PC_ENT values are published percentages",
		DecimalsMetadata:   "not supplied by this JSON-stat response; numeric lexemes are preserved in observations.csv and calculations are displayed to two decimal places",
		Calculations: map[string]string{
			"change_2024_2025_percentage_points":       "all_enterprises_2025_percent - all_enterprises_2024_percent",
			"large_minus_small_2025_percentage_points": "large_2025_percent - small_2025_percent",
		},
		CrossChecks: []string{
			"2026 Eurostat report Table 2: EU text-mining shares round to 6.9% (2024), 11.8% (2025), 9.9% small, 18.2% medium and 35.0% large (2025)",
			"2025 Eurostat news article: EU text-mining share 11.8%",
		},
		Validation: []string{
			"duplicate JSON object keys rejected before decoding",
			"JSON-stat id order, sizes, complete category indexes and positional products validated",
			"unexpected top-level, dimension and extension fields rejected",
			"selected identifiers and live labels validated; data and structure versions recorded",
			"232 unique selected keys; every value present, finite and in [0,100]",
			"missing values and source statuses decoded separately; source status labels retained",
			"EU selected observations reconciled to independently presented rounded values in the official 2026 report",
		},
		Limitations: []string{
			"survey self-report and sampling/non-sampling error; no standard errors in selected slice",
			"annual questionnaire and national methodology changes can limit comparisons",
			"latest Eurostat database values are revisable",
			"denominator is enterprises in each size class within covered NACE activities",
			"enterprise technology use is not vendor count, revenue, productivity or market size",
		},
		TransformationCommand: "go run ./cmd/eurostatai", RefreshCommand: "go run ./cmd/eurostatai -fetch",
		GoVersion: runtime.Version(), Outputs: outputs,
		UnresolvedGates: []string{
			"No numeric request-per-second quota was found in the official Statistics API guides checked on 2026-09-09; refresh is intentionally one request and honours HTTP 429 Retry-After.",
		},
		ReviewGate: "Maintainer must review source rights, question, calculations, caveats and generated artifacts before publication.",
	}, nil
}

func annotationDate(dataset *eurostat.Dataset, annotationType string) string {
	for _, item := range dataset.Extension.Annotation {
		if item.Type == annotationType {
			return item.Date
		}
	}
	return ""
}

func findSummary(summaries []countrySummary, code string) (countrySummary, error) {
	for _, item := range summaries {
		if item.GeoCode == code {
			return item, nil
		}
	}
	return countrySummary{}, fmt.Errorf("summary %s not found", code)
}

func uniqueStatuses(items ...observation) string {
	seen := make(map[string]bool)
	var values []string
	for _, item := range items {
		if item.Status != "" && !seen[item.Status] {
			seen[item.Status] = true
			values = append(values, item.Status)
		}
	}
	sort.Strings(values)
	return strings.Join(values, " ")
}

func shortGeoLabel(item countrySummary) string {
	if item.GeoCode == "EU27_2020" {
		return "EU-27 aggregate"
	}
	return item.GeoLabel
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func format2(value float64) string { return fmt.Sprintf("%.2f", value) }

func cloneMap(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func bytesSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func fileProvenance(path, layer string) (outputProvenance, error) {
	file, err := os.Open(path)
	if err != nil {
		return outputProvenance{}, fmt.Errorf("open %s for checksum: %w", path, err)
	}
	defer file.Close()
	hash := sha256.New()
	count, err := io.Copy(hash, file)
	if err != nil {
		return outputProvenance{}, fmt.Errorf("checksum %s: %w", path, err)
	}
	return outputProvenance{Layer: layer, Bytes: count, SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}

func writeJSON(path string, value any) error {
	return writeAtomic(path, func(writer io.Writer) error {
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		encoder.SetEscapeHTML(false)
		return encoder.Encode(value)
	})
}

func writeAtomic(path string, write func(io.Writer) error) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary output: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := write(temporary); err != nil {
		temporary.Close()
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := temporary.Chmod(0o644); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close %s: %w", path, err)
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}
