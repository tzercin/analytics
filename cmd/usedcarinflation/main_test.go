package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tzercin/analytics/tuik"
)

func TestValidateAlignAndCalculate(t *testing.T) {
	t.Parallel()

	transfers, err := validateVehicle(validVehicleMessage())
	if err != nil {
		t.Fatal(err)
	}
	inflation, err := validateCPI(validCPIMessage())
	if err != nil {
		t.Fatal(err)
	}
	monthly, annual, err := alignAndCalculate(transfers, inflation)
	if err != nil {
		t.Fatal(err)
	}
	if len(monthly) != observationWant || monthly[0].TransferYoY != nil || monthly[12].TransferYoY == nil {
		t.Fatalf("unexpected monthly output: count=%d first_yoy=%v thirteenth_yoy=%v", len(monthly), monthly[0].TransferYoY, monthly[12].TransferYoY)
	}
	if monthly[0].Label != startPeriod || monthly[len(monthly)-1].Label != endPeriod {
		t.Fatalf("unexpected monthly range: %s to %s", monthly[0].Label, monthly[len(monthly)-1].Label)
	}
	if len(annual) != 17 || annual[0].Months != 12 || annual[len(annual)-1].Months != 7 || annual[len(annual)-1].Coverage != "Ocak-Temmuz" {
		t.Fatalf("unexpected annual output: first=%#v last=%#v count=%d", annual[0], annual[len(annual)-1], len(annual))
	}
}

func TestValidateVehicleRejectsCumulativeMonthCode(t *testing.T) {
	t.Parallel()

	message := validVehicleMessage()
	message.DataSets[0].Series[0].Key["AY"] = "M01_07"
	_, err := validateVehicle(message)
	if err == nil || !strings.Contains(err.Error(), "not one exact calendar month") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateVehicleRejectsPeriodInWrongMonthSeries(t *testing.T) {
	t.Parallel()

	message := validVehicleMessage()
	message.DataSets[0].Series[0].Observations[0].Dimension = "2010-02"
	_, err := validateVehicle(message)
	if err == nil || !strings.Contains(err.Error(), "is in AY series") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCPIRejectsDimensionDrift(t *testing.T) {
	t.Parallel()

	message := validCPIMessage()
	message.DataSets[0].Series[0].Key["DEGISIM"] = "2"
	_, err := validateCPI(message)
	if err == nil || !strings.Contains(err.Error(), "CPI dimension DEGISIM") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGeneratedOutputsAreDeterministic(t *testing.T) {
	t.Parallel()

	transfers, err := validateVehicle(validVehicleMessage())
	if err != nil {
		t.Fatal(err)
	}
	inflation, err := validateCPI(validCPIMessage())
	if err != nil {
		t.Fatal(err)
	}
	monthly, annual, err := alignAndCalculate(transfers, inflation)
	if err != nil {
		t.Fatal(err)
	}

	first, second := t.TempDir(), t.TempDir()
	for _, directory := range []string{first, second} {
		if err := writeMonthlyCSV(filepath.Join(directory, "monthly.csv"), monthly); err != nil {
			t.Fatal(err)
		}
		if err := writeAnnualCSV(filepath.Join(directory, "annual.csv"), annual); err != nil {
			t.Fatal(err)
		}
		if err := writeJSON(filepath.Join(directory, "summary.json"), calculateSummary(monthly, annual)); err != nil {
			t.Fatal(err)
		}
		if err := writeSVG(filepath.Join(directory, "comparison.svg"), monthly); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"monthly.csv", "annual.csv", "summary.json", "comparison.svg"} {
		one, err := os.ReadFile(filepath.Join(first, name))
		if err != nil {
			t.Fatal(err)
		}
		two, err := os.ReadFile(filepath.Join(second, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(one, two) {
			t.Errorf("%s differs across identical renders", name)
		}
	}
	monthlyCSV, _ := os.ReadFile(filepath.Join(first, "monthly.csv"))
	if !strings.HasPrefix(string(monthlyCSV), "period,passenger_car_transfers_count,passenger_car_transfers_yoy_percent,headline_cpi_yoy_percent\n2010-01,") || !strings.Contains(string(monthlyCSV), "2026-07,588504,") {
		t.Fatalf("unexpected monthly CSV: %s", monthlyCSV)
	}
	svg, _ := os.ReadFile(filepath.Join(first, "comparison.svg"))
	for _, want := range []string{"Passenger-car transfers vs headline inflation", "Devri Yapılan", "588,504", "31.75%"} {
		if !strings.Contains(string(svg), want) {
			t.Errorf("SVG does not contain %q", want)
		}
	}
}

func TestStatistics(t *testing.T) {
	t.Parallel()

	if got := pearson([]float64{1, 2, 3}, []float64{2, 4, 6}); got != 1 {
		t.Fatalf("pearson = %v, want 1", got)
	}
	want := []float64{1.5, 1.5, 3}
	got := ranks([]float64{4, 4, 9})
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("ranks = %#v, want %#v", got, want)
		}
	}
}

func TestCheckedInArtifactManifest(t *testing.T) {
	t.Parallel()

	directory := filepath.Clean("../../analyses/turkey-used-car-transfers-vs-inflation")
	manifestBytes, err := os.ReadFile(filepath.Join(directory, "provenance.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		OutputSHA256 map[string]string `json:"output_sha256"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"monthly.csv", "annual.csv", "summary.json", "comparison.svg"} {
		got, err := fileSHA256(filepath.Join(directory, name))
		if err != nil {
			t.Fatal(err)
		}
		if got != manifest.OutputSHA256[name] {
			t.Errorf("%s SHA-256 = %s, manifest has %s; regenerate artifacts", name, got, manifest.OutputSHA256[name])
		}
	}

	monthlyFile, err := os.Open(filepath.Join(directory, "monthly.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer monthlyFile.Close()
	records, err := csv.NewReader(monthlyFile).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != observationWant+1 {
		t.Fatalf("monthly.csv has %d rows including header, want %d", len(records), observationWant+1)
	}
	latest := records[len(records)-1]
	if latest[0] != endPeriod || latest[1] != "588504" || latest[2] != "-13.469801" || latest[3] != "31.75" {
		t.Fatalf("unexpected latest checked-in row: %#v", latest)
	}
}

func validVehicleMessage() *tuik.GenericData {
	series := make([]tuik.GenericSeries, 12)
	start, _ := time.Parse("2006-01", startPeriod)
	for index := 0; index < observationWant; index++ {
		period := start.AddDate(0, index, 0)
		monthIndex := int(period.Month()) - 1
		value := int64(200000 + index*2000 + monthIndex)
		if period.Format("2006-01") == endPeriod {
			value = 588504
		}
		key := cloneMap(vehicleExpected)
		key["AY"] = "M" + period.Format("01")
		if series[monthIndex].Key == nil {
			series[monthIndex] = tuik.GenericSeries{Key: key}
		}
		series[monthIndex].Observations = append(series[monthIndex].Observations, tuik.GenericObservation{
			DimensionID: "TIME_PERIOD", Dimension: period.Format("2006-01"), Value: strconv.FormatInt(value, 10),
		})
	}
	return &tuik.GenericData{DataSets: []tuik.GenericDataSet{{Series: series}}}
}

func validCPIMessage() *tuik.GenericData {
	observations := make([]tuik.GenericObservation, 0, cpiObservationWant)
	start, _ := time.Parse("2006-01", startPeriod)
	for index := 0; index < cpiObservationWant; index++ {
		value := strconv.FormatFloat(8+float64(index%37)/3, 'f', 2, 64)
		if index == cpiObservationWant-2 {
			value = "31.75"
		}
		if index == cpiObservationWant-1 {
			value = "31.51"
		}
		observations = append(observations, tuik.GenericObservation{
			DimensionID: "TIME_PERIOD", Dimension: start.AddDate(0, index, 0).Format("2006-01"), Value: value,
		})
	}
	return &tuik.GenericData{DataSets: []tuik.GenericDataSet{{Series: []tuik.GenericSeries{{Key: cloneMap(cpiExpected), Observations: observations}}}}}
}
