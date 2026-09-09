// Command usedcarinflation compares TÜİK's monthly passenger-car transfer
// count with headline CPI inflation. Transfers are an explicitly labelled
// proxy for second-hand market activity, not a TÜİK sales measure.
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
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tzercin/analytics/tuik"
)

const (
	vehicleDataflowID      = "DF_MOTORLU_KARA_TASIT_DEVRI_YAPILAN_V3"
	vehicleDataflowVersion = "1.0"
	vehicleDSDID           = "DSD_MOTORLU_KARA_TASITLARI_V5"
	vehicleDSDVersion      = "1.5"
	vehicleMonthCodes      = "M01+M02+M03+M04+M05+M06+M07+M08+M09+M10+M11+M12"
	vehicleSeriesKey       = "TR.M.U_DYMKTS." + vehicleMonthCodes + "._Z._Z._Z._Z._Z.1.PN._Z._Z._Z"

	cpiDataflowID      = "DF_TUFE_SDMX_TT01"
	cpiDataflowVersion = "1.0"
	cpiDSDID           = "DSD_TUFE"
	cpiDSDVersion      = "1.12"
	cpiSeriesKey       = "TR.M.TUFE.4._Z.2025.2026_01._Z.0.F_TFE"

	startPeriod        = "2010-01"
	endPeriod          = "2026-07"
	observationWant    = 199
	cpiEndPeriod       = "2026-08"
	cpiObservationWant = 200
	yoyWant            = 187
)

var vehicleExpected = map[string]string{
	"REF_AREA":          "TR",
	"FREQ":              "M",
	"INDICATOR":         "U_DYMKTS",
	"TASIT_KAYIT":       "_Z",
	"ARAC_YAS_GRUP":     "_Z",
	"ARAC_RENK":         "_Z",
	"YAKIT_TUR":         "_Z",
	"ARAC_KULLANIM_TUR": "_Z",
	"ARAC_TUR":          "1",
	"UNIT_MEASURE":      "PN",
	"ARAC_SILINDIR":     "_Z",
	"MODEL_YIL":         "_Z",
	"MARKA":             "_Z",
}

var cpiExpected = map[string]string{
	"REF_AREA":         "TR",
	"FREQ":             "M",
	"SINIFLAMA_DUZEYI": "TUFE",
	"DEGISIM":          "4",
	"OZEL_KAPSAM_TUFE": "_Z",
	"BASE_PER":         "2025",
	"YAYIM_DONEMI":     "2026_01",
	"COICOP_1999":      "_Z",
	"COICOP_2018":      "0",
	"INDICATOR":        "F_TFE",
}

type point struct {
	Period time.Time
	Label  string
	Value  float64
	Raw    string
}

type monthlyRow struct {
	Period      time.Time
	Label       string
	Transfers   int64
	TransferYoY *float64
	CPIYoY      float64
	CPIRaw      string
}

type annualRow struct {
	Year               int
	Months             int
	Coverage           string
	Transfers          int64
	ComparablePrevious *int64
	TransferYoY        *float64
	AverageCPIYoY      float64
}

type responseRecord struct {
	RequestURL          string `json:"request_url"`
	StatusCode          int    `json:"status_code"`
	ResponseDate        string `json:"response_date,omitempty"`
	ResponseContentType string `json:"response_content_type"`
	MessageID           string `json:"message_id"`
	MessagePrepared     string `json:"message_prepared"`
	RawByteCount        int64  `json:"raw_byte_count"`
	RawSHA256           string `json:"raw_sha256"`
	RawStored           bool   `json:"raw_stored"`
}

type sourceSnapshot struct {
	Purpose             string `json:"purpose"`
	CanonicalURL        string `json:"canonical_url"`
	RetrievalURL        string `json:"retrieval_url,omitempty"`
	RetrievedAt         string `json:"retrieved_at"`
	StatusCode          int    `json:"status_code"`
	ResponseDate        string `json:"response_date"`
	ResponseContentType string `json:"response_content_type"`
	RawByteCount        int64  `json:"raw_byte_count"`
	RawSHA256           string `json:"raw_sha256"`
	RawStored           bool   `json:"raw_stored"`
}

type datasetProvenance struct {
	OfficialTitle        string            `json:"official_title"`
	DataflowID           string            `json:"dataflow_id"`
	DataflowVersion      string            `json:"dataflow_version"`
	DataStructureID      string            `json:"data_structure_id"`
	DataStructureVersion string            `json:"data_structure_version"`
	SeriesKey            string            `json:"series_key"`
	Dimensions           map[string]string `json:"dimensions"`
	TurkishLabels        map[string]string `json:"turkish_labels"`
	Measure              string            `json:"measure"`
	Unit                 string            `json:"unit"`
	Frequency            string            `json:"frequency"`
	StartPeriod          string            `json:"start_period"`
	EndPeriod            string            `json:"end_period"`
	ObservationCount     int               `json:"observation_count"`
	ObservationStatus    string            `json:"observation_status"`
	Response             responseRecord    `json:"response"`
}

type provenance struct {
	AnalysisID            string            `json:"analysis_id"`
	Status                string            `json:"status"`
	SourceOwner           string            `json:"source_owner"`
	RetrievedAt           string            `json:"retrieved_at"`
	RevisionVintage       string            `json:"revision_vintage"`
	Locale                string            `json:"locale"`
	Encoding              string            `json:"encoding"`
	DocumentationURL      string            `json:"documentation_url"`
	VehicleTransfers      datasetProvenance `json:"vehicle_transfers"`
	HeadlineCPI           datasetProvenance `json:"headline_cpi"`
	VerifiedPublicSources []sourceSnapshot  `json:"verified_public_sources"`
	Method                map[string]any    `json:"method"`
	AccessAndReuse        map[string]string `json:"access_and_reuse"`
	OutputSHA256          map[string]string `json:"output_sha256"`
	CrossChecks           []string          `json:"cross_checks"`
	Validation            []string          `json:"validation"`
}

type summary struct {
	AnalysisID            string            `json:"analysis_id"`
	OperationalDefinition map[string]string `json:"operational_definition"`
	Period                map[string]any    `json:"period"`
	Facts                 map[string]any    `json:"facts"`
	Calculations          map[string]any    `json:"calculations"`
	Inference             map[string]string `json:"inference"`
	Limitations           []string          `json:"limitations"`
}

func main() {
	outputDir := flag.String("output", "analyses/turkey-used-car-transfers-vs-inflation", "directory for generated outputs")
	flag.Parse()
	if err := run(*outputDir); err != nil {
		log.Fatal(err)
	}
}

func run(outputDir string) error {
	apiKey := os.Getenv("TUIK_API_KEY")
	if apiKey == "" {
		return errors.New("TUIK_API_KEY is not set")
	}
	tokens, err := tuik.NewAPIKeyTokenSource(apiKey)
	if err != nil {
		return err
	}
	client, err := tuik.NewClient(tokens)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	vehicleMessage, vehicleResponse, err := retrieve(ctx, client, tuik.DataQuery{
		Agency: "TR", Dataflow: vehicleDataflowID, Version: vehicleDataflowVersion,
		Key: vehicleSeriesKey, StartPeriod: startPeriod, EndPeriod: endPeriod,
	})
	if err != nil {
		return fmt.Errorf("retrieve passenger-car transfers: %w", err)
	}
	cpiMessage, cpiResponse, err := retrieve(ctx, client, tuik.DataQuery{
		Agency: "TR", Dataflow: cpiDataflowID, Version: cpiDataflowVersion,
		Key: cpiSeriesKey, StartPeriod: startPeriod, EndPeriod: cpiEndPeriod,
	})
	if err != nil {
		return fmt.Errorf("retrieve headline CPI: %w", err)
	}

	transfers, err := validateVehicle(vehicleMessage)
	if err != nil {
		return fmt.Errorf("validate passenger-car transfers: %w", err)
	}
	inflation, err := validateCPI(cpiMessage)
	if err != nil {
		return fmt.Errorf("validate headline CPI: %w", err)
	}
	monthly, annual, err := alignAndCalculate(transfers, inflation)
	if err != nil {
		return err
	}
	result := calculateSummary(monthly, annual)

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	paths := map[string]string{
		"monthly.csv":    filepath.Join(outputDir, "monthly.csv"),
		"annual.csv":     filepath.Join(outputDir, "annual.csv"),
		"summary.json":   filepath.Join(outputDir, "summary.json"),
		"comparison.svg": filepath.Join(outputDir, "comparison.svg"),
	}
	if err := writeMonthlyCSV(paths["monthly.csv"], monthly); err != nil {
		return err
	}
	if err := writeAnnualCSV(paths["annual.csv"], annual); err != nil {
		return err
	}
	if err := writeJSON(paths["summary.json"], result); err != nil {
		return err
	}
	if err := writeSVG(paths["comparison.svg"], monthly); err != nil {
		return err
	}

	retrievedAt := time.Now().In(time.FixedZone("Europe/Istanbul", 3*60*60)).Format(time.RFC3339)
	metadata := buildProvenance(retrievedAt, vehicleResponse, cpiResponse)
	for name, path := range paths {
		metadata.OutputSHA256[name], err = fileSHA256(path)
		if err != nil {
			return err
		}
	}
	if err := writeJSON(filepath.Join(outputDir, "provenance.json"), metadata); err != nil {
		return err
	}

	fmt.Printf("wrote %d aligned months (%s to %s) to %s\n", len(monthly), monthly[0].Label, monthly[len(monthly)-1].Label, outputDir)
	return nil
}

func retrieve(ctx context.Context, client *tuik.Client, query tuik.DataQuery) (*tuik.GenericData, responseRecord, error) {
	response, err := client.Data(ctx, query)
	if err != nil {
		return nil, responseRecord{}, err
	}
	defer response.Body.Close()
	hash := sha256.New()
	counter := &countingReader{reader: io.TeeReader(response.Body, hash)}
	message, err := tuik.DecodeGenericData(counter)
	if err != nil {
		return nil, responseRecord{}, err
	}
	return message, responseRecord{
		RequestURL:          response.Request.URL.String(),
		StatusCode:          response.StatusCode,
		ResponseDate:        response.Header.Get("Date"),
		ResponseContentType: response.Header.Get("Content-Type"),
		MessageID:           message.Header.ID,
		MessagePrepared:     message.Header.Prepared,
		RawByteCount:        counter.count,
		RawSHA256:           hex.EncodeToString(hash.Sum(nil)),
		RawStored:           false,
	}, nil
}

func validateVehicle(message *tuik.GenericData) ([]point, error) {
	if len(message.DataSets) != 1 {
		return nil, fmt.Errorf("expected one data set, got %d", len(message.DataSets))
	}
	series := message.DataSets[0].Series
	if len(series) != 12 {
		return nil, fmt.Errorf("expected 12 month-specific series, got %d", len(series))
	}
	seenMonths := make(map[string]bool, 12)
	seenPeriods := make(map[string]bool, observationWant)
	points := make([]point, 0, observationWant)
	for _, item := range series {
		if len(item.Key) != len(vehicleExpected)+1 {
			return nil, fmt.Errorf("vehicle series has %d dimensions, want %d", len(item.Key), len(vehicleExpected)+1)
		}
		for dimension, value := range vehicleExpected {
			if item.Key[dimension] != value {
				return nil, fmt.Errorf("vehicle dimension %s = %q, want %q", dimension, item.Key[dimension], value)
			}
		}
		monthCode := item.Key["AY"]
		month, ok := parseMonthCode(monthCode)
		if !ok {
			return nil, fmt.Errorf("AY = %q is not one exact calendar month", monthCode)
		}
		if seenMonths[monthCode] {
			return nil, fmt.Errorf("duplicate vehicle AY series %s", monthCode)
		}
		seenMonths[monthCode] = true
		for _, observation := range item.Observations {
			period, err := parseObservationPeriod(observation)
			if err != nil {
				return nil, err
			}
			if period.Month() != month {
				return nil, fmt.Errorf("period %s is in AY series %s", observation.Dimension, monthCode)
			}
			value, err := strconv.ParseInt(observation.Value, 10, 64)
			if err != nil || value < 0 {
				return nil, fmt.Errorf("period %s has invalid non-negative integer count %q", observation.Dimension, observation.Value)
			}
			if len(observation.Attributes) != 0 {
				return nil, fmt.Errorf("period %s has unreviewed observation status attributes", observation.Dimension)
			}
			if seenPeriods[observation.Dimension] {
				return nil, fmt.Errorf("duplicate vehicle period %s", observation.Dimension)
			}
			seenPeriods[observation.Dimension] = true
			points = append(points, point{Period: period, Label: observation.Dimension, Value: float64(value), Raw: observation.Value})
		}
	}
	if len(seenMonths) != 12 {
		return nil, fmt.Errorf("got %d unique vehicle month series, want 12", len(seenMonths))
	}
	if err := validateRange(points, observationWant, startPeriod, endPeriod); err != nil {
		return nil, err
	}
	latest := points[len(points)-1]
	if latest.Value != 588504 {
		return nil, fmt.Errorf("latest passenger-car transfer count is %.0f, want published-table value 588504", latest.Value)
	}
	if math.Round((latest.Value/907846)*1000)/10 != 64.8 {
		return nil, errors.New("latest passenger-car count does not reconcile to the release total and rounded share")
	}
	return points, nil
}

func validateCPI(message *tuik.GenericData) ([]point, error) {
	if len(message.DataSets) != 1 {
		return nil, fmt.Errorf("expected one data set, got %d", len(message.DataSets))
	}
	if len(message.DataSets[0].Series) != 1 {
		return nil, fmt.Errorf("expected one CPI series, got %d", len(message.DataSets[0].Series))
	}
	series := message.DataSets[0].Series[0]
	if len(series.Key) != len(cpiExpected) {
		return nil, fmt.Errorf("CPI series has %d dimensions, want %d", len(series.Key), len(cpiExpected))
	}
	for dimension, value := range cpiExpected {
		if series.Key[dimension] != value {
			return nil, fmt.Errorf("CPI dimension %s = %q, want %q", dimension, series.Key[dimension], value)
		}
	}
	points := make([]point, 0, len(series.Observations))
	seen := make(map[string]bool, len(series.Observations))
	for _, observation := range series.Observations {
		period, err := parseObservationPeriod(observation)
		if err != nil {
			return nil, err
		}
		value, err := strconv.ParseFloat(observation.Value, 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, fmt.Errorf("period %s has invalid CPI value %q", observation.Dimension, observation.Value)
		}
		if len(observation.Attributes) != 0 {
			return nil, fmt.Errorf("period %s has unreviewed observation status attributes", observation.Dimension)
		}
		if seen[observation.Dimension] {
			return nil, fmt.Errorf("duplicate CPI period %s", observation.Dimension)
		}
		seen[observation.Dimension] = true
		points = append(points, point{Period: period, Label: observation.Dimension, Value: value, Raw: observation.Value})
	}
	if err := validateRange(points, cpiObservationWant, startPeriod, cpiEndPeriod); err != nil {
		return nil, err
	}
	if math.Abs(points[len(points)-1].Value-31.51) > 0.000001 {
		return nil, fmt.Errorf("latest CPI annual rate is %.6g, want August 2026 release value 31.51", points[len(points)-1].Value)
	}
	if math.Abs(points[len(points)-2].Value-31.75) > 0.000001 {
		return nil, fmt.Errorf("common-period ending CPI annual rate is %.6g, want July 2026 release value 31.75", points[len(points)-2].Value)
	}
	return points, nil
}

func parseMonthCode(code string) (time.Month, bool) {
	if len(code) != 3 || !strings.HasPrefix(code, "M") {
		return 0, false
	}
	month, err := strconv.Atoi(code[1:])
	return time.Month(month), err == nil && month >= 1 && month <= 12
}

func parseObservationPeriod(observation tuik.GenericObservation) (time.Time, error) {
	if observation.DimensionID != "TIME_PERIOD" {
		return time.Time{}, fmt.Errorf("observation dimension is %q, want TIME_PERIOD", observation.DimensionID)
	}
	period, err := time.Parse("2006-01", observation.Dimension)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse period %q: %w", observation.Dimension, err)
	}
	return period, nil
}

func validateRange(points []point, want int, wantStart, wantEnd string) error {
	if len(points) != want {
		return fmt.Errorf("got %d observations, want %d", len(points), want)
	}
	sort.Slice(points, func(i, j int) bool { return points[i].Period.Before(points[j].Period) })
	for index, item := range points {
		wantPeriod := points[0].Period.AddDate(0, index, 0)
		if !item.Period.Equal(wantPeriod) {
			return fmt.Errorf("period gap: got %s, want %s", item.Label, wantPeriod.Format("2006-01"))
		}
	}
	if points[0].Label != wantStart || points[len(points)-1].Label != wantEnd {
		return fmt.Errorf("period range is %s to %s, want %s to %s", points[0].Label, points[len(points)-1].Label, wantStart, wantEnd)
	}
	return nil
}

func alignAndCalculate(transfers, inflation []point) ([]monthlyRow, []annualRow, error) {
	inflationByPeriod := make(map[string]point, len(inflation))
	for _, item := range inflation {
		inflationByPeriod[item.Label] = item
	}
	monthly := make([]monthlyRow, 0, len(transfers))
	for index, transfer := range transfers {
		item, ok := inflationByPeriod[transfer.Label]
		if !ok {
			return nil, nil, fmt.Errorf("transfer period %s has no CPI observation", transfer.Label)
		}
		row := monthlyRow{Period: transfer.Period, Label: transfer.Label, Transfers: int64(transfer.Value), CPIYoY: item.Value, CPIRaw: item.Raw}
		if index >= 12 {
			prior := int64(transfers[index-12].Value)
			rate := (float64(row.Transfers)/float64(prior) - 1) * 100
			row.TransferYoY = &rate
		}
		monthly = append(monthly, row)
	}
	if len(monthly)-12 != yoyWant {
		return nil, nil, fmt.Errorf("got %d transfer YoY comparisons, want %d", len(monthly)-12, yoyWant)
	}

	annual := make([]annualRow, 0, endYear()-startYear()+1)
	for year := startYear(); year <= endYear(); year++ {
		var rows []monthlyRow
		for _, row := range monthly {
			if row.Period.Year() == year {
				rows = append(rows, row)
			}
		}
		result := annualRow{Year: year, Months: len(rows), Transfers: sumTransfers(rows), AverageCPIYoY: averageCPI(rows)}
		if len(rows) == 12 {
			result.Coverage = "Ocak-Aralık"
		} else {
			result.Coverage = "Ocak-" + turkishMonth(rows[len(rows)-1].Period.Month())
		}
		if year > startYear() {
			var previous []monthlyRow
			for _, row := range monthly {
				if row.Period.Year() == year-1 && int(row.Period.Month()) <= len(rows) {
					previous = append(previous, row)
				}
			}
			prior := sumTransfers(previous)
			rate := (float64(result.Transfers)/float64(prior) - 1) * 100
			result.ComparablePrevious = &prior
			result.TransferYoY = &rate
		}
		annual = append(annual, result)
	}
	return monthly, annual, nil
}

func calculateSummary(monthly []monthlyRow, annual []annualRow) summary {
	latest := monthly[len(monthly)-1]
	complete := annual[len(annual)-2]
	var transferRates, cpiRates []float64
	var sensitivityTransfer, sensitivityCPI []float64
	for _, row := range monthly {
		if row.TransferYoY == nil {
			continue
		}
		transferRates = append(transferRates, *row.TransferYoY)
		cpiRates = append(cpiRates, row.CPIYoY)
		if row.Label < "2020-03" || row.Label > "2021-06" {
			sensitivityTransfer = append(sensitivityTransfer, *row.TransferYoY)
			sensitivityCPI = append(sensitivityCPI, row.CPIYoY)
		}
	}
	annualTransfer, annualCPI := make([]float64, 0, 15), make([]float64, 0, 15)
	for _, row := range annual {
		if row.Months == 12 && row.TransferYoY != nil {
			annualTransfer = append(annualTransfer, *row.TransferYoY)
			annualCPI = append(annualCPI, row.AverageCPIYoY)
		}
	}
	return summary{
		AnalysisID: "turkey-used-car-transfers-vs-inflation",
		OperationalDefinition: map[string]string{
			"requested_concept": "second-hand car market sales",
			"implemented_proxy": "Türkiye genelinde noterler aracılığıyla devri yapılan otomobil sayısı",
			"source_measure":    "Devri Yapılan Motorlu Kara Taşıtları Sayısı — Otomobil — Sayı",
			"warning":           "TÜİK bu seriyi satış, satış değeri veya tekil araç sayısı olarak tanımlamaz.",
		},
		Period: map[string]any{"start": startPeriod, "end": endPeriod, "monthly_observations": len(monthly), "monthly_yoy_observations": len(transferRates)},
		Facts: map[string]any{
			"latest_period": latest.Label, "latest_passenger_car_transfers": latest.Transfers,
			"latest_headline_cpi_yoy_percent": latest.CPIYoY,
			"latest_complete_year":            complete.Year, "latest_complete_year_passenger_car_transfers": complete.Transfers,
		},
		Calculations: map[string]any{
			"latest_passenger_car_transfers_yoy_percent":                    round(*latest.TransferYoY, 6),
			"latest_complete_year_transfers_yoy_percent":                    round(*complete.TransferYoY, 6),
			"latest_complete_year_average_monthly_headline_cpi_yoy_percent": round(complete.AverageCPIYoY, 6),
			"monthly_yoy_pearson":                                           map[string]any{"coefficient": round(pearson(transferRates, cpiRates), 6), "n": len(transferRates)},
			"monthly_yoy_spearman":                                          map[string]any{"coefficient": round(pearson(ranks(transferRates), ranks(cpiRates)), 6), "n": len(transferRates)},
			"monthly_yoy_pearson_excluding_2020_03_through_2021_06":         map[string]any{"coefficient": round(pearson(sensitivityTransfer, sensitivityCPI), 6), "n": len(sensitivityTransfer)},
			"complete_year_growth_vs_average_inflation_pearson":             map[string]any{"coefficient": round(pearson(annualTransfer, annualCPI), 6), "n": len(annualTransfer)},
		},
		Inference: map[string]string{
			"descriptive_result": "The full-period year-over-year association is weak and not robust enough to support a causal claim.",
			"causality":          "None inferred; inflation and transfers may both respond to omitted economic, credit, tax, supply, and policy factors.",
		},
		Limitations: []string{
			"The proxy records the published administrative transfer measure, not sale contracts or consideration paid.",
			"The public aggregate does not identify unique vehicles, buyers, sellers, prices, or whether multiple transfers of one vehicle within a period are deduplicated.",
			"Passenger cars only: minibuses, buses, motorcycles, small trucks, trucks, special-purpose vehicles, and tractors are excluded.",
			"Counts are not seasonally adjusted; year-over-year comparisons reduce but do not eliminate calendar effects.",
			"Correlation is descriptive, serial dependence is present, and no statistical or causal model is estimated.",
		},
	}
}

func buildProvenance(retrievedAt string, vehicle, cpi responseRecord) provenance {
	return provenance{
		AnalysisID:       "turkey-used-car-transfers-vs-inflation",
		Status:           "local review draft; not approved for publication",
		SourceOwner:      "Türkiye İstatistik Kurumu (TÜİK)",
		RetrievedAt:      retrievedAt,
		RevisionVintage:  "current live SDMX API vintage retrieved on 2026-09-08; vehicle release vintage Temmuz 2026 (published 2026-08-17); CPI selection YAYIM_DONEMI=2026_01 and BASE_PER=2025",
		Locale:           "Turkish labels selected from SDMX xml:lang=tr and the Turkish press-release metadata",
		Encoding:         "UTF-8, as declared by the source response content types",
		DocumentationURL: "https://veriportali.tuik.gov.tr/tr/sdmx-web-service-documentation",
		VehicleTransfers: datasetProvenance{
			OfficialTitle: "Aylara Göre Devri Yapılan Motorlu Kara Taşıtları Sayısı",
			DataflowID:    vehicleDataflowID, DataflowVersion: vehicleDataflowVersion,
			DataStructureID: vehicleDSDID, DataStructureVersion: vehicleDSDVersion,
			SeriesKey: vehicleSeriesKey, Dimensions: selectedVehicleDimensions(),
			TurkishLabels: map[string]string{
				"REF_AREA=TR": "Türkiye", "FREQ=M": "Aylık", "INDICATOR=U_DYMKTS": "Devri Yapılan Motorlu Kara Taşıtları Sayısı",
				"AY=M01": "Ocak", "AY=M02": "Şubat", "AY=M03": "Mart", "AY=M04": "Nisan", "AY=M05": "Mayıs", "AY=M06": "Haziran",
				"AY=M07": "Temmuz", "AY=M08": "Ağustos", "AY=M09": "Eylül", "AY=M10": "Ekim", "AY=M11": "Kasım", "AY=M12": "Aralık",
				"ARAC_TUR=1": "Otomobil", "UNIT_MEASURE=PN": "Sayı",
			},
			Measure: "Devri Yapılan Motorlu Kara Taşıtları Sayısı — Otomobil", Unit: "Sayı (tam sayı)", Frequency: "Aylık",
			StartPeriod: startPeriod, EndPeriod: endPeriod, ObservationCount: observationWant,
			ObservationStatus: "no SDMX observation-status attributes were present", Response: vehicle,
		},
		HeadlineCPI: datasetProvenance{
			OfficialTitle: "Ana harcama gruplarına göre tüketici fiyat endeksi (TÜFE) ve değişim oranları",
			DataflowID:    cpiDataflowID, DataflowVersion: cpiDataflowVersion,
			DataStructureID: cpiDSDID, DataStructureVersion: cpiDSDVersion,
			SeriesKey: cpiSeriesKey, Dimensions: cloneMap(cpiExpected),
			TurkishLabels: map[string]string{
				"REF_AREA=TR": "Türkiye", "FREQ=M": "Aylık", "DEGISIM=4": "Bir önceki yılın aynı ayına göre (yıllık) değişim oranı (%)",
				"COICOP_2018=0": "Genel", "INDICATOR=F_TFE": "Tüketici Fiyat Endeksi (TÜFE)",
			},
			Measure: "Bir önceki yılın aynı ayına göre (yıllık) değişim oranı (%)", Unit: "yüzde", Frequency: "Aylık",
			StartPeriod: startPeriod, EndPeriod: cpiEndPeriod, ObservationCount: cpiObservationWant,
			ObservationStatus: "no SDMX observation-status attributes were present", Response: cpi,
		},
		VerifiedPublicSources: []sourceSnapshot{
			{Purpose: "vehicle dataflow title, description, and DSD reference", CanonicalURL: "https://nsiws.tuik.gov.tr/rest/dataflow/TR/DF_MOTORLU_KARA_TASIT_DEVRI_YAPILAN_V3/1.0?detail=full", RetrievedAt: "2026-09-08T16:18:26+03:00", StatusCode: 200, ResponseDate: "Tue, 08 Sep 2026 13:18:26 GMT", ResponseContentType: "application/vnd.sdmx.structure+xml; charset=utf-8; version=2.1", RawByteCount: 1525, RawSHA256: "ca479ece2c74a616585dbff158c6142b660f1e4ea0ee38da8993f2fb15c69a41", RawStored: false},
			{Purpose: "vehicle DSD dimensions and Turkish codelist labels", CanonicalURL: "https://nsiws.tuik.gov.tr/rest/datastructure/TR/DSD_MOTORLU_KARA_TASITLARI_V5/1.5?detail=full&references=all", RetrievedAt: "2026-09-08T16:18:49+03:00", StatusCode: 200, ResponseDate: "Tue, 08 Sep 2026 13:18:49 GMT", ResponseContentType: "application/vnd.sdmx.structure+xml; version=2.1; charset=utf-8", RawByteCount: 2610847, RawSHA256: "c6927015c693aa0697257e14cb62fb873523683709c5de301fd480cdf188e0c3", RawStored: false},
			{Purpose: "current vehicle definition, coverage, sources, calculation, and revision statement", CanonicalURL: "https://veriportali.tuik.gov.tr/tr/press/58045/metadata", RetrievalURL: "https://veriportali.tuik.gov.tr/api/tr/press/58045", RetrievedAt: "2026-09-08T16:25:20+03:00", StatusCode: 200, ResponseDate: "Tue, 08 Sep 2026 13:25:20 GMT", ResponseContentType: "application/json; charset=utf-8", RawByteCount: 366143, RawSHA256: "a5eefdb62a6d7effad7a575c7a51a63a231ebe588eccd90235f58fbad72e5907", RawStored: false},
			{Purpose: "reuse and attribution terms", CanonicalURL: "https://www.tuik.gov.tr/Kurumsal/Yasal_Uyari", RetrievedAt: "2026-09-08T16:30:47+03:00", StatusCode: 200, ResponseDate: "Tue, 08 Sep 2026 13:30:47 GMT", ResponseContentType: "text/html; charset=utf-8", RawByteCount: 127672, RawSHA256: "032bc496d1121ce4995eb34c0639f8e9171e40c8a58ebc44c58034f92ceb2bc6", RawStored: false},
		},
		Method: map[string]any{
			"operational_definition": "noterler aracılığıyla devri yapılan otomobil sayısı; second-hand activity proxy only",
			"join":                   "one-to-one inner join on 199 unique consecutive YYYY-MM periods",
			"transfer_yoy":           "100 * (transfers_t / transfers_t_minus_12 - 1), unrounded inputs",
			"annual":                 "sum transfer counts; arithmetic mean of the 12 monthly headline CPI annual rates; 2026 is January-July YTD and compared with January-July 2025",
			"association":            "Pearson and tie-aware Spearman correlations of the two monthly YoY series; descriptive only",
			"lagged_correlations":    "not reported: serial dependence plus 25 candidate lags would invite multiple-testing interpretation without a prespecified economic model",
		},
		AccessAndReuse: map[string]string{
			"authentication": "TÜİK's SDMX service requires a bearer token obtained with a Data Portal API key; TUIK_API_KEY is supplied only at runtime and is never stored or logged",
			"reuse":          "TÜİK's Yasal Uyarı states that website, publication, and database data may be reused without permission when TÜİK is cited",
			"rate_limits":    "no numeric rate limit was stated in the inspected SDMX documentation; this generator makes only two sequential data requests per run",
			"raw_storage":    "source response bytes are not committed; only byte counts and SHA-256 digests are retained",
		},
		OutputSHA256: make(map[string]string),
		CrossChecks: []string{
			"https://veriportali.tuik.gov.tr/tr/press/58045 (Temmuz 2026: 907,846 total transfers; passenger cars 64.8%; selected exact count 588,504)",
			"https://veriportali.tuik.gov.tr/tr/press/58297 (Temmuz 2026 headline CPI annual rate: 31.75%)",
		},
		Validation: []string{
			"vehicle response contains one data set and exactly 12 non-cumulative calendar-month series (AY=M01 through M12)",
			"vehicle and CPI dimensions exactly match documented selections; Turkish categories are not normalized",
			"199 unique consecutive vehicle observations and 200 unique consecutive CPI observations; one-to-one common-period join for 2010-01 through 2026-07",
			"transfer values are non-negative integers; CPI values and all calculated rates are finite",
			"no observation-level status attributes, duplicates, or missing months; every selected vehicle month has exactly one CPI match (the source-only August 2026 CPI observation is outside the common period)",
			"latest values reconcile to the July 2026 vehicle and CPI releases",
			"CSV, JSON, and SVG writers are deterministic for identical validated inputs; output hashes record artifact drift",
		},
	}
}

func writeMonthlyCSV(path string, rows []monthlyRow) error {
	return writeAtomic(path, func(writer io.Writer) error {
		w := csv.NewWriter(writer)
		if err := w.Write([]string{"period", "passenger_car_transfers_count", "passenger_car_transfers_yoy_percent", "headline_cpi_yoy_percent"}); err != nil {
			return err
		}
		for _, row := range rows {
			rate := ""
			if row.TransferYoY != nil {
				rate = formatFloat(*row.TransferYoY)
			}
			if err := w.Write([]string{row.Label, strconv.FormatInt(row.Transfers, 10), rate, row.CPIRaw}); err != nil {
				return err
			}
		}
		w.Flush()
		return w.Error()
	})
}

func writeAnnualCSV(path string, rows []annualRow) error {
	return writeAtomic(path, func(writer io.Writer) error {
		w := csv.NewWriter(writer)
		if err := w.Write([]string{"year", "months_observed", "period_coverage_tr", "passenger_car_transfers_count", "comparable_prior_period_transfers_count", "passenger_car_transfers_yoy_percent", "average_monthly_headline_cpi_yoy_percent"}); err != nil {
			return err
		}
		for _, row := range rows {
			prior, rate := "", ""
			if row.ComparablePrevious != nil {
				prior = strconv.FormatInt(*row.ComparablePrevious, 10)
				rate = formatFloat(*row.TransferYoY)
			}
			if err := w.Write([]string{strconv.Itoa(row.Year), strconv.Itoa(row.Months), row.Coverage, strconv.FormatInt(row.Transfers, 10), prior, rate, formatFloat(row.AverageCPIYoY)}); err != nil {
				return err
			}
		}
		w.Flush()
		return w.Error()
	})
}

func writeSVG(path string, rows []monthlyRow) error {
	const (
		width       = 1200.0
		height      = 900.0
		left        = 92.0
		right       = 44.0
		plotWidth   = width - left - right
		topOne      = 142.0
		panelHeight = 260.0
		topTwo      = 500.0
	)
	x := func(index int) float64 { return left + float64(index)*plotWidth/float64(len(rows)-1) }
	yCount := func(value float64) float64 { return topOne + (900000-value)*panelHeight/900000 }
	yRate := func(value float64) float64 { return topTwo + (130-value)*panelHeight/190 }

	var countLine, transferLine, cpiLine strings.Builder
	for index, row := range rows {
		command := "L"
		if index == 0 {
			command = "M"
		}
		fmt.Fprintf(&countLine, "%s%.2f %.2f ", command, x(index), yCount(float64(row.Transfers)))
		if row.TransferYoY != nil {
			command = "L"
			if transferLine.Len() == 0 {
				command = "M"
			}
			fmt.Fprintf(&transferLine, "%s%.2f %.2f ", command, x(index), yRate(*row.TransferYoY))
		}
		fmt.Fprintf(&cpiLine, "%s%.2f %.2f ", commandFor(index), x(index), yRate(row.CPIYoY))
	}
	latest := rows[len(rows)-1]

	return writeAtomic(path, func(writer io.Writer) error {
		writef := func(format string, args ...any) { _, _ = fmt.Fprintf(writer, format, args...) }
		writef(`<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="900" viewBox="0 0 1200 900" role="img" aria-labelledby="title desc">` + "\n")
		writef(`<title id="title">Türkiye passenger-car transfers and headline CPI inflation, January 2010 to July 2026</title>` + "\n")
		writef(`<desc id="desc">Two panels. The upper panel shows monthly notarized passenger-car transfer counts. The lower panel compares their year-over-year change with headline CPI annual inflation. Transfers are a second-hand market activity proxy, not sales or sales value.</desc>` + "\n")
		writef(`<rect width="1200" height="900" fill="#fbfaf7"/>` + "\n")
		writef(`<text x="92" y="48" font-family="Inter,system-ui,sans-serif" font-size="28" font-weight="700" fill="#17202a">Passenger-car transfers vs headline inflation</text>` + "\n")
		writef(`<text x="92" y="78" font-family="Inter,system-ui,sans-serif" font-size="16" fill="#58636f">Türkiye • January 2010–July 2026 • monthly • TÜİK</text>` + "\n")
		writef(`<rect x="92" y="94" width="1010" height="30" rx="6" fill="#fff0dc"/>` + "\n")
		writef(`<text x="106" y="114" font-family="Inter,system-ui,sans-serif" font-size="12" font-weight="600" fill="#7a4b12">Proxy: “Devri Yapılan … Otomobil” counts notarized transfers; TÜİK does not label this series as sales or sales value.</text>` + "\n")

		writef(`<text x="92" y="137" font-family="Inter,system-ui,sans-serif" font-size="14" font-weight="700" fill="#17202a">Monthly passenger-car transfers</text>` + "\n")
		for tick := 0.0; tick <= 900000; tick += 300000 {
			y := yCount(tick)
			writef(`<line x1="92" y1="%.2f" x2="1156" y2="%.2f" stroke="#dce1e5"/>`, y, y)
			writef(`<text x="78" y="%.2f" text-anchor="end" font-family="Inter,system-ui,sans-serif" font-size="11" fill="#6b7480">%.0fk</text>`+"\n", y+4, tick/1000)
		}
		writef(`<path d="%s" fill="none" stroke="#177e89" stroke-width="2.5" stroke-linejoin="round" stroke-linecap="round"/>`+"\n", strings.TrimSpace(countLine.String()))
		writef(`<circle cx="%.2f" cy="%.2f" r="4.5" fill="#177e89" stroke="#fbfaf7" stroke-width="2"/>`, x(len(rows)-1), yCount(float64(latest.Transfers)))
		writef(`<text x="%.2f" y="%.2f" text-anchor="end" font-family="Inter,system-ui,sans-serif" font-size="12" font-weight="700" fill="#17202a">%s</text>`+"\n", x(len(rows)-1)-8, yCount(float64(latest.Transfers))-9, formatInteger(latest.Transfers))

		writef(`<text x="92" y="477" font-family="Inter,system-ui,sans-serif" font-size="14" font-weight="700" fill="#17202a">Year-over-year change</text>` + "\n")
		writef(`<line x1="810" y1="472" x2="840" y2="472" stroke="#177e89" stroke-width="3"/><text x="848" y="476" font-family="Inter,system-ui,sans-serif" font-size="12" fill="#58636f">Passenger-car transfers</text>` + "\n")
		writef(`<line x1="1010" y1="472" x2="1040" y2="472" stroke="#c92545" stroke-width="3"/><text x="1048" y="476" font-family="Inter,system-ui,sans-serif" font-size="12" fill="#58636f">Headline CPI</text>` + "\n")
		for tick := -60.0; tick <= 120; tick += 30 {
			y := yRate(tick)
			color, stroke := "#dce1e5", "1"
			if tick == 0 {
				color, stroke = "#89939d", "1.5"
			}
			writef(`<line x1="92" y1="%.2f" x2="1156" y2="%.2f" stroke="%s" stroke-width="%s"/>`, y, y, color, stroke)
			writef(`<text x="78" y="%.2f" text-anchor="end" font-family="Inter,system-ui,sans-serif" font-size="11" fill="#6b7480">%.0f%%</text>`+"\n", y+4, tick)
		}
		for index, row := range rows {
			if row.Period.Month() == time.January && row.Period.Year()%2 == 0 {
				writef(`<line x1="%.2f" y1="142" x2="%.2f" y2="760" stroke="#edf0f2"/>`, x(index), x(index))
				writef(`<text x="%.2f" y="790" text-anchor="middle" font-family="Inter,system-ui,sans-serif" font-size="11" fill="#6b7480">%d</text>`+"\n", x(index), row.Period.Year())
			}
		}
		writef(`<path d="%s" fill="none" stroke="#177e89" stroke-width="2.5" stroke-linejoin="round" stroke-linecap="round"/>`+"\n", strings.TrimSpace(transferLine.String()))
		writef(`<path d="%s" fill="none" stroke="#c92545" stroke-width="2.5" stroke-linejoin="round" stroke-linecap="round"/>`+"\n", strings.TrimSpace(cpiLine.String()))
		writef(`<circle cx="%.2f" cy="%.2f" r="4" fill="#177e89"/><text x="%.2f" y="%.2f" text-anchor="end" font-family="Inter,system-ui,sans-serif" font-size="11" font-weight="700" fill="#177e89">%.1f%%</text>`, x(len(rows)-1), yRate(*latest.TransferYoY), x(len(rows)-1)-8, yRate(*latest.TransferYoY)-8, *latest.TransferYoY)
		writef(`<circle cx="%.2f" cy="%.2f" r="4" fill="#c92545"/><text x="%.2f" y="%.2f" text-anchor="end" font-family="Inter,system-ui,sans-serif" font-size="11" font-weight="700" fill="#c92545">%.2f%%</text>`+"\n", x(len(rows)-1), yRate(latest.CPIYoY), x(len(rows)-1)-8, yRate(latest.CPIYoY)-8, latest.CPIYoY)
		writef(`<text x="92" y="842" font-family="Inter,system-ui,sans-serif" font-size="11" fill="#77818b">Source: TÜİK SDMX • Transfers: %s v%s, Otomobil, Sayı • Inflation: %s v%s, headline annual rate</text>`+"\n", vehicleDataflowID, vehicleDataflowVersion, cpiDataflowID, cpiDataflowVersion)
		writef(`<text x="92" y="862" font-family="Inter,system-ui,sans-serif" font-size="11" fill="#77818b">Unadjusted counts; transfer YoY calculated from unrounded values. Correlation is descriptive and does not imply causation.</text>` + "\n")
		writef(`</svg>` + "\n")
		return nil
	})
}

func commandFor(index int) string {
	if index == 0 {
		return "M"
	}
	return "L"
}

func sumTransfers(rows []monthlyRow) int64 {
	var total int64
	for _, row := range rows {
		total += row.Transfers
	}
	return total
}

func averageCPI(rows []monthlyRow) float64 {
	var total float64
	for _, row := range rows {
		total += row.CPIYoY
	}
	return total / float64(len(rows))
}

func pearson(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 2 {
		return math.NaN()
	}
	var meanX, meanY float64
	for index := range x {
		meanX += x[index]
		meanY += y[index]
	}
	meanX /= float64(len(x))
	meanY /= float64(len(y))
	var covariance, varianceX, varianceY float64
	for index := range x {
		dx, dy := x[index]-meanX, y[index]-meanY
		covariance += dx * dy
		varianceX += dx * dx
		varianceY += dy * dy
	}
	return covariance / math.Sqrt(varianceX*varianceY)
}

func ranks(values []float64) []float64 {
	type ranked struct {
		value float64
		index int
	}
	ordered := make([]ranked, len(values))
	for index, value := range values {
		ordered[index] = ranked{value: value, index: index}
	}
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].value < ordered[j].value })
	result := make([]float64, len(values))
	for first := 0; first < len(ordered); {
		last := first
		for last+1 < len(ordered) && ordered[last+1].value == ordered[first].value {
			last++
		}
		rank := float64(first+last+2) / 2
		for index := first; index <= last; index++ {
			result[ordered[index].index] = rank
		}
		first = last + 1
	}
	return result
}

func round(value float64, places int) float64 {
	scale := math.Pow10(places)
	return math.Round(value*scale) / scale
}

func formatFloat(value float64) string { return strconv.FormatFloat(value, 'f', 6, 64) }

func formatInteger(value int64) string {
	digits := strconv.FormatInt(value, 10)
	for index := len(digits) - 3; index > 0; index -= 3 {
		digits = digits[:index] + "," + digits[index:]
	}
	return digits
}

func turkishMonth(month time.Month) string {
	return []string{"", "Ocak", "Şubat", "Mart", "Nisan", "Mayıs", "Haziran", "Temmuz", "Ağustos", "Eylül", "Ekim", "Kasım", "Aralık"}[month]
}

func startYear() int { return 2010 }
func endYear() int   { return 2026 }

type countingReader struct {
	reader io.Reader
	count  int64
}

func (reader *countingReader) Read(buffer []byte) (int, error) {
	n, err := reader.reader.Read(buffer)
	reader.count += int64(n)
	return n, err
}

func cloneMap(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func selectedVehicleDimensions() map[string]string {
	result := cloneMap(vehicleExpected)
	result["AY"] = vehicleMonthCodes
	return result
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
		return fmt.Errorf("set permissions for %s: %w", path, err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close %s: %w", path, err)
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %s for checksum: %w", path, err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("checksum %s: %w", path, err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
