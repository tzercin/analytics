// Package eurostat provides the small JSON-stat and HTTP surface needed by
// reproducible Eurostat analyses in this repository.
package eurostat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"
)

// Dataset is the documented Eurostat statistics API JSON-stat 2 dataset shape.
type Dataset struct {
	Version   string               `json:"version"`
	Class     string               `json:"class"`
	Label     string               `json:"label"`
	Source    string               `json:"source"`
	Updated   string               `json:"updated"`
	Value     SparseNumbers        `json:"value"`
	Status    SparseStrings        `json:"status"`
	ID        []string             `json:"id"`
	Size      []int                `json:"size"`
	Dimension map[string]Dimension `json:"dimension"`
	Extension Extension            `json:"extension"`

	product int
}

type Dimension struct {
	Label    string   `json:"label"`
	Category Category `json:"category"`
}

type Category struct {
	Index map[string]int    `json:"index"`
	Label map[string]string `json:"label"`
}

type Extension struct {
	Lang                string             `json:"lang"`
	ID                  string             `json:"id"`
	AgencyID            string             `json:"agencyId"`
	Version             string             `json:"version"`
	DataStructure       StructureReference `json:"datastructure"`
	Annotation          []Annotation       `json:"annotation"`
	Status              StatusMetadata     `json:"status"`
	PositionsWithNoData map[string][]int   `json:"positions-with-no-data"`
}

type StructureReference struct {
	ID       string `json:"id"`
	AgencyID string `json:"agencyId"`
	Version  string `json:"version"`
}

type Annotation struct {
	Type  string `json:"type"`
	Title string `json:"title,omitempty"`
	Text  string `json:"text,omitempty"`
	Date  string `json:"date,omitempty"`
	Href  string `json:"href,omitempty"`
}

type StatusMetadata struct {
	Label map[string]string `json:"label"`
}

// SparseNumbers accepts either the sparse object used by Eurostat or a dense
// JSON-stat array. JSON numbers are retained lexically for faithful CSV output.
type SparseNumbers map[int]json.Number

func (values *SparseNumbers) UnmarshalJSON(data []byte) error {
	result := make(SparseNumbers)
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		*values = result
		return nil
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return fmt.Errorf("empty JSON value")
	}
	if bytes.TrimSpace(data)[0] == '{' {
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			return err
		}
		for key, item := range raw {
			index, err := strconv.Atoi(key)
			if err != nil || index < 0 {
				return fmt.Errorf("invalid sparse value index %q", key)
			}
			if bytes.Equal(bytes.TrimSpace(item), []byte("null")) {
				continue
			}
			var number json.Number
			if err := json.Unmarshal(item, &number); err != nil {
				return fmt.Errorf("value %d: %w", index, err)
			}
			result[index] = number
		}
		*values = result
		return nil
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("values must be an object or array: %w", err)
	}
	for index, item := range raw {
		if bytes.Equal(bytes.TrimSpace(item), []byte("null")) {
			continue
		}
		var number json.Number
		if err := json.Unmarshal(item, &number); err != nil {
			return fmt.Errorf("value %d: %w", index, err)
		}
		result[index] = number
	}
	*values = result
	return nil
}

// SparseStrings accepts either a sparse status object or a dense status array.
type SparseStrings map[int]string

func (values *SparseStrings) UnmarshalJSON(data []byte) error {
	result := make(SparseStrings)
	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("null")) || len(trimmed) == 0 {
		*values = result
		return nil
	}
	if trimmed[0] == '{' {
		var raw map[string]string
		if err := json.Unmarshal(data, &raw); err != nil {
			return err
		}
		for key, value := range raw {
			index, err := strconv.Atoi(key)
			if err != nil || index < 0 {
				return fmt.Errorf("invalid sparse status index %q", key)
			}
			result[index] = value
		}
		*values = result
		return nil
	}
	var raw []*string
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("statuses must be an object or array: %w", err)
	}
	for index, value := range raw {
		if value != nil && *value != "" {
			result[index] = *value
		}
	}
	*values = result
	return nil
}

type Cell struct {
	Index       int
	Value       *float64
	RawValue    string
	Status      string
	StatusLabel string
}

// DecodeDataset rejects duplicate JSON keys, unknown fields, malformed category
// indexes, invalid values and other shape drift before positional decoding.
func DecodeDataset(data []byte) (*Dataset, error) {
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	var dataset Dataset
	if err := decoder.Decode(&dataset); err != nil {
		return nil, fmt.Errorf("decode JSON-stat: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return nil, err
	}
	if err := dataset.validate(); err != nil {
		return nil, err
	}
	return &dataset, nil
}

func (dataset *Dataset) validate() error {
	if dataset.Version != "2.0" || dataset.Class != "dataset" {
		return fmt.Errorf("unexpected JSON-stat identity version=%q class=%q", dataset.Version, dataset.Class)
	}
	if dataset.Label == "" || dataset.Source == "" || dataset.Updated == "" {
		return fmt.Errorf("dataset label, source and updated timestamp are required")
	}
	if len(dataset.ID) == 0 || len(dataset.ID) != len(dataset.Size) {
		return fmt.Errorf("dimension id/size length mismatch: %d/%d", len(dataset.ID), len(dataset.Size))
	}
	seenDimensions := make(map[string]bool, len(dataset.ID))
	product := 1
	for position, id := range dataset.ID {
		if id == "" || seenDimensions[id] {
			return fmt.Errorf("empty or duplicate dimension id %q", id)
		}
		seenDimensions[id] = true
		size := dataset.Size[position]
		if size <= 0 || product > math.MaxInt/size {
			return fmt.Errorf("invalid dimension size %d for %s", size, id)
		}
		product *= size
		dimension, ok := dataset.Dimension[id]
		if !ok {
			return fmt.Errorf("dimension %s metadata missing", id)
		}
		if dimension.Label == "" || len(dimension.Category.Index) != size || len(dimension.Category.Label) != size {
			return fmt.Errorf("dimension %s category metadata has wrong size", id)
		}
		positions := make([]bool, size)
		for code, index := range dimension.Category.Index {
			if code == "" || index < 0 || index >= size || positions[index] {
				return fmt.Errorf("dimension %s has invalid or duplicate category index %d for %q", id, index, code)
			}
			if _, ok := dimension.Category.Label[code]; !ok {
				return fmt.Errorf("dimension %s category %s has no label", id, code)
			}
			positions[index] = true
		}
	}
	if len(dataset.Dimension) != len(seenDimensions) {
		return fmt.Errorf("unexpected dimension metadata: got %d dimensions, want %d", len(dataset.Dimension), len(seenDimensions))
	}
	for index, number := range dataset.Value {
		if index < 0 || index >= product {
			return fmt.Errorf("value index %d outside product size %d", index, product)
		}
		value, err := number.Float64()
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return fmt.Errorf("value index %d is not finite: %q", index, number)
		}
	}
	for index, status := range dataset.Status {
		if index < 0 || index >= product {
			return fmt.Errorf("status index %d outside product size %d", index, product)
		}
		if status == "" {
			return fmt.Errorf("status index %d is empty", index)
		}
		if _, ok := dataset.Extension.Status.Label[status]; !ok {
			return fmt.Errorf("status %q at index %d has no label", status, index)
		}
	}
	dataset.product = product
	return nil
}

// Codes returns category codes in their positional JSON-stat order.
func (dataset *Dataset) Codes(dimensionID string) ([]string, error) {
	dimension, ok := dataset.Dimension[dimensionID]
	if !ok {
		return nil, fmt.Errorf("unknown dimension %q", dimensionID)
	}
	result := make([]string, len(dimension.Category.Index))
	for code, index := range dimension.Category.Index {
		result[index] = code
	}
	return result, nil
}

// Cell returns one positional observation. Value is nil for a missing cell;
// status and its label are retained independently of missingness.
func (dataset *Dataset) Cell(coordinates map[string]string) (Cell, error) {
	if len(coordinates) != len(dataset.ID) {
		return Cell{}, fmt.Errorf("got %d coordinates, want %d", len(coordinates), len(dataset.ID))
	}
	linear := 0
	for position, id := range dataset.ID {
		code, ok := coordinates[id]
		if !ok {
			return Cell{}, fmt.Errorf("coordinate for dimension %s missing", id)
		}
		categoryPosition, ok := dataset.Dimension[id].Category.Index[code]
		if !ok {
			return Cell{}, fmt.Errorf("dimension %s has no code %q", id, code)
		}
		linear = linear*dataset.Size[position] + categoryPosition
	}
	cell := Cell{Index: linear, Status: dataset.Status[linear]}
	if cell.Status != "" {
		cell.StatusLabel = dataset.Extension.Status.Label[cell.Status]
	}
	if number, ok := dataset.Value[linear]; ok {
		value, _ := number.Float64()
		cell.Value = &value
		cell.RawValue = number.String()
	}
	return cell, nil
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return fmt.Errorf("trailing JSON: %w", err)
	}
	return nil
}

// rejectDuplicateJSONKeys walks the token stream because encoding/json would
// otherwise silently keep the final value of a repeated object key.
func rejectDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := walkJSON(decoder, "$"); err != nil {
		return err
	}
	return ensureEOF(decoder)
}

func walkJSON(decoder *json.Decoder, path string) error {
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("invalid JSON at %s: %w", path, err)
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]bool)
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return fmt.Errorf("invalid object at %s: %w", path, err)
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("non-string object key at %s", path)
			}
			if seen[key] {
				return fmt.Errorf("duplicate JSON key %q at %s", key, path)
			}
			seen[key] = true
			if err := walkJSON(decoder, path+"."+key); err != nil {
				return err
			}
		}
	case '[':
		for index := 0; decoder.More(); index++ {
			if err := walkJSON(decoder, fmt.Sprintf("%s[%d]", path, index)); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unexpected delimiter %q at %s", delimiter, path)
	}
	if _, err := decoder.Token(); err != nil {
		return fmt.Errorf("unclosed value at %s: %w", path, err)
	}
	return nil
}
