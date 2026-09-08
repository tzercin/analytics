// Command inflation10y retrieves and renders the fixed first analysis of
// Türkiye's headline monthly CPI annual rate of change.
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
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tzercin/analytics/tuik"
)

const (
	dataflowID      = "DF_TUFE_SDMX_TT01"
	dataflowVersion = "1.0"
	seriesKey       = "TR.M.TUFE.4._Z.2025.2026_01._Z.0.F_TFE"
	startPeriod     = "2016-09"
	endPeriod       = "2026-08"
	observationWant = 120
)

var expectedKey = map[string]string{
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

type provenance struct {
	SourceOwner          string            `json:"source_owner"`
	DocumentationURL     string            `json:"documentation_url"`
	DataflowTitle        string            `json:"dataflow_title"`
	DataflowID           string            `json:"dataflow_id"`
	DataflowVersion      string            `json:"dataflow_version"`
	DataStructureID      string            `json:"data_structure_id"`
	DataStructureVersion string            `json:"data_structure_version"`
	SeriesKey            string            `json:"series_key"`
	Dimensions           map[string]string `json:"dimensions"`
	Measure              string            `json:"measure"`
	Unit                 string            `json:"unit"`
	StartPeriod          string            `json:"start_period"`
	EndPeriod            string            `json:"end_period"`
	ObservationCount     int               `json:"observation_count"`
	RequestURL           string            `json:"request_url"`
	RetrievedAt          string            `json:"retrieved_at"`
	ResponseDate         string            `json:"response_date,omitempty"`
	ResponseContentType  string            `json:"response_content_type"`
	MessageID            string            `json:"message_id"`
	MessagePrepared      string            `json:"message_prepared"`
	RawByteCount         int64             `json:"raw_byte_count"`
	RawSHA256            string            `json:"raw_sha256"`
	RawStored            bool              `json:"raw_stored"`
	CSVSHA256            string            `json:"csv_sha256"`
	SVGSHA256            string            `json:"svg_sha256"`
	CrossChecks          []string          `json:"cross_checks"`
	Validation           []string          `json:"validation"`
}

func main() {
	outputDir := flag.String("output", "analyses/turkey-cpi-yoy-10y", "directory for generated outputs")
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

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	response, err := client.Data(ctx, tuik.DataQuery{
		Agency:      "TR",
		Dataflow:    dataflowID,
		Version:     dataflowVersion,
		Key:         seriesKey,
		StartPeriod: startPeriod,
		EndPeriod:   endPeriod,
	})
	if err != nil {
		return err
	}
	defer response.Body.Close()

	hash := sha256.New()
	counter := &countingReader{reader: io.TeeReader(response.Body, hash)}
	message, err := tuik.DecodeGenericData(counter)
	if err != nil {
		return err
	}
	points, err := validateAndConvert(message)
	if err != nil {
		return err
	}

	retrievedAt := time.Now().In(time.FixedZone("Europe/Istanbul", 3*60*60))
	metadata := provenance{
		SourceOwner:          "Türkiye İstatistik Kurumu (TÜİK)",
		DocumentationURL:     "https://veriportali.tuik.gov.tr/tr/sdmx-web-service-documentation",
		DataflowTitle:        "Ana harcama gruplarına göre tüketici fiyat endeksi (TÜFE) ve değişim oranları",
		DataflowID:           dataflowID,
		DataflowVersion:      dataflowVersion,
		DataStructureID:      "DSD_TUFE",
		DataStructureVersion: "1.12",
		SeriesKey:            seriesKey,
		Dimensions:           cloneMap(expectedKey),
		Measure:              "Bir önceki yılın aynı ayına göre (yıllık) değişim oranı (%)",
		Unit:                 "percent",
		StartPeriod:          startPeriod,
		EndPeriod:            endPeriod,
		ObservationCount:     len(points),
		RequestURL:           response.Request.URL.String(),
		RetrievedAt:          retrievedAt.Format(time.RFC3339),
		ResponseDate:         response.Header.Get("Date"),
		ResponseContentType:  response.Header.Get("Content-Type"),
		MessageID:            message.Header.ID,
		MessagePrepared:      message.Header.Prepared,
		RawByteCount:         counter.count,
		RawSHA256:            hex.EncodeToString(hash.Sum(nil)),
		RawStored:            false,
		CrossChecks: []string{
			"https://veriportali.tuik.gov.tr/en/press/58290 (August 2026: 31.51%)",
			"https://veriportali.tuik.gov.tr/en/press/45799 (October 2022: 85.51%)",
		},
		Validation: []string{
			"exactly one data set and one series",
			"series dimensions exactly match the documented selection",
			"120 unique and consecutive monthly observations",
			"all observation values are finite numbers",
			"latest value reconciles to TÜİK August 2026 press release (31.51%)",
		},
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	csvPath := filepath.Join(outputDir, "inflation.csv")
	svgPath := filepath.Join(outputDir, "inflation.svg")
	if err := writeCSV(csvPath, points); err != nil {
		return err
	}
	if err := writeSVG(svgPath, points, retrievedAt); err != nil {
		return err
	}
	metadata.CSVSHA256, err = fileSHA256(csvPath)
	if err != nil {
		return err
	}
	metadata.SVGSHA256, err = fileSHA256(svgPath)
	if err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outputDir, "provenance.json"), metadata); err != nil {
		return err
	}

	fmt.Printf("wrote %d observations (%s to %s) to %s\n", len(points), points[0].Label, points[len(points)-1].Label, outputDir)
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

type countingReader struct {
	reader io.Reader
	count  int64
}

func (r *countingReader) Read(buffer []byte) (int, error) {
	n, err := r.reader.Read(buffer)
	r.count += int64(n)
	return n, err
}

func validateAndConvert(message *tuik.GenericData) ([]point, error) {
	if len(message.DataSets) != 1 {
		return nil, fmt.Errorf("expected one data set, got %d", len(message.DataSets))
	}
	if len(message.DataSets[0].Series) != 1 {
		return nil, fmt.Errorf("expected one series, got %d", len(message.DataSets[0].Series))
	}
	series := message.DataSets[0].Series[0]
	if len(series.Key) != len(expectedKey) {
		return nil, fmt.Errorf("series has %d dimensions, want %d", len(series.Key), len(expectedKey))
	}
	for dimension, value := range expectedKey {
		if series.Key[dimension] != value {
			return nil, fmt.Errorf("dimension %s = %q, want %q", dimension, series.Key[dimension], value)
		}
	}
	if len(series.Observations) != observationWant {
		return nil, fmt.Errorf("got %d observations, want %d", len(series.Observations), observationWant)
	}

	points := make([]point, 0, len(series.Observations))
	seen := make(map[string]bool, len(series.Observations))
	for _, observation := range series.Observations {
		if observation.DimensionID != "TIME_PERIOD" {
			return nil, fmt.Errorf("observation dimension is %q, want TIME_PERIOD", observation.DimensionID)
		}
		period, err := time.Parse("2006-01", observation.Dimension)
		if err != nil {
			return nil, fmt.Errorf("parse period %q: %w", observation.Dimension, err)
		}
		value, err := strconv.ParseFloat(observation.Value, 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, fmt.Errorf("period %s has invalid value %q", observation.Dimension, observation.Value)
		}
		if seen[observation.Dimension] {
			return nil, fmt.Errorf("duplicate period %s", observation.Dimension)
		}
		seen[observation.Dimension] = true
		points = append(points, point{Period: period, Label: observation.Dimension, Value: value, Raw: observation.Value})
	}
	sort.Slice(points, func(i, j int) bool { return points[i].Period.Before(points[j].Period) })
	for index, item := range points {
		want := points[0].Period.AddDate(0, index, 0)
		if !item.Period.Equal(want) {
			return nil, fmt.Errorf("period gap: got %s, want %s", item.Label, want.Format("2006-01"))
		}
	}
	if points[0].Label != startPeriod || points[len(points)-1].Label != endPeriod {
		return nil, fmt.Errorf("period range is %s to %s, want %s to %s", points[0].Label, points[len(points)-1].Label, startPeriod, endPeriod)
	}
	if math.Abs(points[len(points)-1].Value-31.51) > 0.000001 {
		return nil, fmt.Errorf("latest value is %.6g, want press-release value 31.51", points[len(points)-1].Value)
	}
	return points, nil
}

func writeCSV(path string, points []point) error {
	return writeAtomic(path, func(writer io.Writer) error {
		csvWriter := csv.NewWriter(writer)
		if err := csvWriter.Write([]string{"period", "annual_rate_percent"}); err != nil {
			return err
		}
		for _, point := range points {
			if err := csvWriter.Write([]string{point.Label, point.Raw}); err != nil {
				return err
			}
		}
		csvWriter.Flush()
		return csvWriter.Error()
	})
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

func cloneMap(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func writeSVG(path string, points []point, retrievedAt time.Time) error {
	const (
		width       = 1200.0
		height      = 675.0
		left        = 92.0
		right       = 42.0
		top         = 112.0
		bottom      = 82.0
		chartWidth  = width - left - right
		chartHeight = height - top - bottom
	)
	maximum := points[0]
	for _, item := range points[1:] {
		if item.Value > maximum.Value {
			maximum = item
		}
	}
	yMaximum := math.Ceil(maximum.Value/10) * 10
	x := func(index int) float64 { return left + float64(index)*chartWidth/float64(len(points)-1) }
	y := func(value float64) float64 { return top + (yMaximum-value)*chartHeight/yMaximum }

	var line strings.Builder
	var area strings.Builder
	for index, item := range points {
		command := "L"
		if index == 0 {
			command = "M"
		}
		fmt.Fprintf(&line, "%s%.2f %.2f", command, x(index), y(item.Value))
		if index+1 < len(points) {
			line.WriteByte(' ')
		}
	}
	area.WriteString(line.String())
	fmt.Fprintf(&area, " L%.2f %.2f L%.2f %.2f Z", x(len(points)-1), top+chartHeight, x(0), top+chartHeight)

	return writeAtomic(path, func(writer io.Writer) error {
		writef := func(format string, args ...any) { _, _ = fmt.Fprintf(writer, format, args...) }
		writef(`<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="675" viewBox="0 0 1200 675" role="img" aria-labelledby="title desc">` + "\n")
		writef(`<title id="title">Türkiye headline CPI inflation, annual rate, September 2016 to August 2026</title>` + "\n")
		writef(`<desc id="desc">A line chart of 120 monthly year-over-year inflation observations from TÜİK. The peak is %.2f percent in %s and the latest value is %.2f percent in %s.</desc>`+"\n", maximum.Value, maximum.Label, points[len(points)-1].Value, points[len(points)-1].Label)
		writef(`<defs><linearGradient id="area" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#d9364f" stop-opacity="0.28"/><stop offset="1" stop-color="#d9364f" stop-opacity="0.02"/></linearGradient></defs>` + "\n")
		writef(`<rect width="1200" height="675" fill="#fbfaf7"/>` + "\n")
		writef(`<text x="92" y="48" font-family="Inter,system-ui,sans-serif" font-size="28" font-weight="700" fill="#17202a">Türkiye headline CPI inflation</text>` + "\n")
		writef(`<text x="92" y="78" font-family="Inter,system-ui,sans-serif" font-size="16" fill="#58636f">Annual change from the same month one year earlier • Sep 2016–Aug 2026 • percent</text>` + "\n")

		for tick := 0.0; tick <= yMaximum; tick += 10 {
			tickY := y(tick)
			writef(`<line x1="92" y1="%.2f" x2="1158" y2="%.2f" stroke="#dce1e5" stroke-width="1"/>`, tickY, tickY)
			writef(`<text x="78" y="%.2f" text-anchor="end" font-family="Inter,system-ui,sans-serif" font-size="12" fill="#6b7480">%.0f%%</text>`+"\n", tickY+4, tick)
		}
		for index, item := range points {
			if item.Period.Month() != time.January {
				continue
			}
			writef(`<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="#edf0f2" stroke-width="1"/>`, x(index), top, x(index), top+chartHeight)
			writef(`<text x="%.2f" y="%.2f" text-anchor="middle" font-family="Inter,system-ui,sans-serif" font-size="12" fill="#6b7480">%d</text>`+"\n", x(index), top+chartHeight+28, item.Period.Year())
		}

		writef(`<path d="%s" fill="url(#area)"/>`+"\n", area.String())
		writef(`<path d="%s" fill="none" stroke="#c92545" stroke-width="3" stroke-linejoin="round" stroke-linecap="round"/>`+"\n", line.String())

		maximumIndex := sort.Search(len(points), func(i int) bool { return !points[i].Period.Before(maximum.Period) })
		writef(`<circle cx="%.2f" cy="%.2f" r="5" fill="#c92545" stroke="#fbfaf7" stroke-width="2"/>`, x(maximumIndex), y(maximum.Value))
		writef(`<rect x="%.2f" y="%.2f" width="132" height="48" rx="7" fill="#17202a"/>`, x(maximumIndex)-66, y(maximum.Value)-62)
		writef(`<text x="%.2f" y="%.2f" text-anchor="middle" font-family="Inter,system-ui,sans-serif" font-size="13" font-weight="700" fill="white">Peak %.2f%%</text>`, x(maximumIndex), y(maximum.Value)-41, maximum.Value)
		writef(`<text x="%.2f" y="%.2f" text-anchor="middle" font-family="Inter,system-ui,sans-serif" font-size="11" fill="#d9e0e5">%s</text>`+"\n", x(maximumIndex), y(maximum.Value)-25, html.EscapeString(maximum.Label))

		latest := points[len(points)-1]
		writef(`<circle cx="%.2f" cy="%.2f" r="5" fill="#c92545" stroke="#fbfaf7" stroke-width="2"/>`, x(len(points)-1), y(latest.Value))
		writef(`<text x="%.2f" y="%.2f" text-anchor="end" font-family="Inter,system-ui,sans-serif" font-size="13" font-weight="700" fill="#17202a">%.2f%%</text>`+"\n", x(len(points)-1)-9, y(latest.Value)-10, latest.Value)

		writef(`<text x="92" y="640" font-family="Inter,system-ui,sans-serif" font-size="11" fill="#77818b">Source: TÜİK SDMX dataflow %s v%s • Retrieved %s • Values are the published annual change rate, not an annual average.</text>`+"\n", dataflowID, dataflowVersion, retrievedAt.Format("2006-01-02"))
		writef(`</svg>` + "\n")
		return nil
	})
}
