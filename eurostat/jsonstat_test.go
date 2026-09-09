package eurostat

import (
	"os"
	"strings"
	"testing"
)

func TestDecodeDatasetPositionalCellsAndMissingStatus(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("testdata/minimal.json")
	if err != nil {
		t.Fatal(err)
	}
	dataset, err := DecodeDataset(data)
	if err != nil {
		t.Fatal(err)
	}
	cell, err := dataset.Cell(map[string]string{"geo": "TR", "time": "2025"})
	if err != nil {
		t.Fatal(err)
	}
	if cell.Index != 3 || cell.Value == nil || *cell.Value != 4.75 || cell.RawValue != "4.75" {
		t.Fatalf("wrong positional cell: %#v", cell)
	}
	missing, err := dataset.Cell(map[string]string{"geo": "TR", "time": "2024"})
	if err != nil {
		t.Fatal(err)
	}
	if missing.Index != 2 || missing.Value != nil || missing.Status != "|C" || missing.StatusLabel != "|confidential" {
		t.Fatalf("missing/status information not preserved: %#v", missing)
	}
	codes, err := dataset.Codes("geo")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(codes, ",") != "EU27_2020,TR" {
		t.Fatalf("codes are not in positional order: %v", codes)
	}
}

func TestDecodeDatasetRejectsDuplicateJSONKey(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("testdata/minimal.json")
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), `"label": "Fixture"`, `"label": "Fixture", "label": "Drift"`, 1))
	_, err = DecodeDataset(data)
	if err == nil || !strings.Contains(err.Error(), "duplicate JSON key") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecodeDatasetRejectsInvalidCategoryAndUnknownField(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("testdata/minimal.json")
	if err != nil {
		t.Fatal(err)
	}
	invalidIndex := []byte(strings.Replace(string(data), `"TR": 1, "EU27_2020": 0`, `"TR": 0, "EU27_2020": 0`, 1))
	if _, err := DecodeDataset(invalidIndex); err == nil || !strings.Contains(err.Error(), "duplicate category index") {
		t.Fatalf("unexpected index error: %v", err)
	}
	unknown := []byte(strings.Replace(string(data), `"source": "ESTAT",`, `"source": "ESTAT", "surprise": true,`, 1))
	if _, err := DecodeDataset(unknown); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unexpected field error: %v", err)
	}
}
