package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSummarizeAndGeneratedOutputsAreDeterministic(t *testing.T) {
	t.Parallel()
	observations := syntheticObservations()
	summaries, err := summarize(observations)
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != len(geographies) {
		t.Fatalf("got %d summaries, want %d", len(summaries), len(geographies))
	}
	eu, err := findSummary(summaries, "EU27_2020")
	if err != nil {
		t.Fatal(err)
	}
	if eu.Change != 1 || eu.LargeMinusSmall2025 != 20 {
		t.Fatalf("wrong calculations: %#v", eu)
	}

	directory := t.TempDir()
	receipt := acquisition{RetrievedAt: "2026-09-09T14:00:00+03:00", DatasetUpdated: "2026-06-15T11:00:00+0200"}
	first := filepath.Join(directory, "first.svg")
	second := filepath.Join(directory, "second.svg")
	if err := writeSVG(first, summaries, receipt); err != nil {
		t.Fatal(err)
	}
	if err := writeSVG(second, summaries, receipt); err != nil {
		t.Fatal(err)
	}
	left, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	right, err := os.ReadFile(second)
	if err != nil {
		t.Fatal(err)
	}
	if string(left) != string(right) {
		t.Fatal("SVG generation is not deterministic")
	}
	for _, wanted := range []string{`role="img"`, `<title id="title">`, `<desc id="desc">`, "Türkiye", "Modified analysis"} {
		if !strings.Contains(string(left), wanted) {
			t.Errorf("SVG missing %q", wanted)
		}
	}
}

func TestReconcilePublishedEU(t *testing.T) {
	t.Parallel()
	summaries := []countrySummary{{
		GeoCode: "EU27_2020", All2024: 6.88, All2025: 11.75,
		Small2025: 9.87, Medium2025: 18.22, Large2025: 35.04,
	}}
	if err := reconcilePublishedEU(summaries); err != nil {
		t.Fatal(err)
	}
	summaries[0].All2025 = 12.25
	if err := reconcilePublishedEU(summaries); err == nil || !strings.Contains(err.Error(), "published rounded value") {
		t.Fatalf("unexpected drift error: %v", err)
	}
}

func TestQueryFiltersAreBounded(t *testing.T) {
	t.Parallel()
	encoded := queryFilters().Encode()
	wanted := "freq=A&indic_is=E_AI_TTM&lang=en&nace_r2=C10-S951_X_K&sinceTimePeriod=2024&unit=PC_ENT&untilTimePeriod=2025"
	if encoded != wanted {
		t.Fatalf("query = %q, want %q", encoded, wanted)
	}
}

func syntheticObservations() []observation {
	result := make([]observation, 0, rowWant)
	for geoIndex, geography := range geographies {
		for sizeIndex, size := range sizes {
			for yearIndex, year := range years {
				value := float64(geoIndex) + float64(sizeIndex*10) + float64(yearIndex)
				result = append(result, observation{
					GeoCode: geography.Code, GeoLabel: geography.Code, GeoRole: geography.Role,
					Year: year, SizeCode: size.Code, SizeLabel: size.Label,
					Value: value, RawValue: format2(value),
				})
			}
		}
	}
	return result
}
