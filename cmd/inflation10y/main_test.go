package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tzercin/analytics/tuik"
)

func TestValidateAndConvert(t *testing.T) {
	t.Parallel()

	message := validMessage()
	points, err := validateAndConvert(message)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != observationWant || points[0].Label != startPeriod || points[len(points)-1].Label != endPeriod {
		t.Fatalf("unexpected points: first=%#v last=%#v count=%d", points[0], points[len(points)-1], len(points))
	}
}

func TestValidateAndConvertRejectsDimensionDrift(t *testing.T) {
	t.Parallel()

	message := validMessage()
	message.DataSets[0].Series[0].Key["DEGISIM"] = "2"
	_, err := validateAndConvert(message)
	if err == nil || !strings.Contains(err.Error(), "dimension DEGISIM") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGeneratedOutputs(t *testing.T) {
	t.Parallel()

	points, err := validateAndConvert(validMessage())
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	if err := writeCSV(filepath.Join(directory, "inflation.csv"), points); err != nil {
		t.Fatal(err)
	}
	if err := writeSVG(filepath.Join(directory, "inflation.svg"), points, time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}

	csvBytes, err := os.ReadFile(filepath.Join(directory, "inflation.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(csvBytes), "period,annual_rate_percent\n2016-09,") || !strings.Contains(string(csvBytes), "2026-08,31.51\n") {
		t.Fatalf("unexpected CSV: %s", csvBytes)
	}
	svgBytes, err := os.ReadFile(filepath.Join(directory, "inflation.svg"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Türkiye headline CPI inflation", "Peak 85.51%", "31.51%"} {
		if !strings.Contains(string(svgBytes), want) {
			t.Errorf("SVG does not contain %q", want)
		}
	}
}

func validMessage() *tuik.GenericData {
	observations := make([]tuik.GenericObservation, 0, observationWant)
	period, _ := time.Parse("2006-01", startPeriod)
	for index := range observationWant {
		value := "10"
		if index == 73 {
			value = "85.51"
		}
		if index == observationWant-1 {
			value = "31.51"
		}
		observations = append(observations, tuik.GenericObservation{
			DimensionID: "TIME_PERIOD",
			Dimension:   period.AddDate(0, index, 0).Format("2006-01"),
			Value:       value,
		})
	}
	return &tuik.GenericData{
		DataSets: []tuik.GenericDataSet{{
			Series: []tuik.GenericSeries{{
				Key:          cloneMap(expectedKey),
				Observations: observations,
			}},
		}},
	}
}
