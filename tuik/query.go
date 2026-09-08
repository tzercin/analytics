package tuik

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// StructureResource identifies an SDMX structural-metadata resource.
type StructureResource string

const (
	Dataflows       StructureResource = "dataflow"
	DataStructures  StructureResource = "datastructure"
	Codelists       StructureResource = "codelist"
	ConceptSchemes  StructureResource = "conceptscheme"
	CategorySchemes StructureResource = "categoryscheme"
	Categorisations StructureResource = "categorisation"
)

// StructureQuery describes a structural-metadata request. Agency and ID are
// required. Version is optional because TÜİK documents both versioned resources
// and bulk paths without an explicit version.
type StructureQuery struct {
	Resource   StructureResource
	Agency     string
	ID         string
	Version    string
	Detail     string
	References string
}

// Structure executes a structural-metadata request.
func (c *Client) Structure(ctx context.Context, query StructureQuery) (*http.Response, error) {
	if !validStructureResource(query.Resource) {
		return nil, fmt.Errorf("tuik: unsupported structure resource %q", query.Resource)
	}
	if err := validatePathSegment("agency", query.Agency, false); err != nil {
		return nil, err
	}
	if err := validatePathSegment("structure ID", query.ID, false); err != nil {
		return nil, err
	}
	if err := validatePathSegment("version", query.Version, true); err != nil {
		return nil, err
	}

	segments := []string{string(query.Resource), query.Agency, query.ID}
	if query.Version != "" {
		segments = append(segments, query.Version)
	}
	parameters := make(url.Values)
	if query.Detail != "" {
		parameters.Set("detail", query.Detail)
	}
	if query.References != "" {
		parameters.Set("references", query.References)
	}
	return c.get(ctx, segments, parameters)
}

func validStructureResource(resource StructureResource) bool {
	switch resource {
	case Dataflows, DataStructures, Codelists, ConceptSchemes, CategorySchemes, Categorisations:
		return true
	default:
		return false
	}
}

// DataFormat is a TÜİK format query value.
type DataFormat string

const (
	// FormatDefault requests the documented default SDMX-ML 2.1 Generic Data.
	FormatDefault DataFormat = ""
	// FormatSDMX21Structured requests SDMX-ML 2.1 Structured Data.
	FormatSDMX21Structured DataFormat = "SDMX_2.1_STRUCTURED"
	// FormatSDMXCSV requests SDMX-CSV 1.0.
	FormatSDMXCSV DataFormat = "SDMX-CSV"
	// FormatJSON requests JSON-stat.
	FormatJSON DataFormat = "JSON"
)

// DataQuery describes an SDMX data request. Key may be empty to request an
// unfiltered dataset. Its dimensions must already be ordered according to the
// dataflow's DSD; empty dimensions are represented by adjacent dots and
// multiple values by plus signs.
type DataQuery struct {
	Agency      string
	Dataflow    string
	Version     string
	Key         string
	StartPeriod string
	EndPeriod   string
	Format      DataFormat
}

// Data executes an SDMX data request.
func (c *Client) Data(ctx context.Context, query DataQuery) (*http.Response, error) {
	if err := validatePathSegment("agency", query.Agency, false); err != nil {
		return nil, err
	}
	if err := validatePathSegment("dataflow", query.Dataflow, false); err != nil {
		return nil, err
	}
	if err := validatePathSegment("version", query.Version, false); err != nil {
		return nil, err
	}
	if err := validatePathSegment("series key", query.Key, true); err != nil {
		return nil, err
	}
	if !validDataFormat(query.Format) {
		return nil, fmt.Errorf("tuik: unsupported data format %q", query.Format)
	}

	segments := []string{"data", query.Agency + "," + query.Dataflow + "," + query.Version}
	if query.Key != "" {
		segments = append(segments, query.Key)
	}
	parameters := make(url.Values)
	if query.StartPeriod != "" {
		parameters.Set("startPeriod", query.StartPeriod)
	}
	if query.EndPeriod != "" {
		parameters.Set("endPeriod", query.EndPeriod)
	}
	if query.Format != FormatDefault {
		parameters.Set("format", string(query.Format))
	}
	return c.get(ctx, segments, parameters)
}

func validDataFormat(format DataFormat) bool {
	switch format {
	case FormatDefault, FormatSDMX21Structured, FormatSDMXCSV, FormatJSON:
		return true
	default:
		return false
	}
}

func validatePathSegment(name, value string, allowEmpty bool) error {
	if value == "" {
		if allowEmpty {
			return nil
		}
		return fmt.Errorf("tuik: %s is empty", name)
	}
	if value == "." || value == ".." {
		return fmt.Errorf("tuik: %s is not a valid path value", name)
	}
	if strings.ContainsAny(value, "/?#\\") {
		return fmt.Errorf("tuik: %s contains a path or query delimiter", name)
	}
	if strings.IndexFunc(value, func(r rune) bool { return r <= ' ' || r == 0x7f }) >= 0 {
		return fmt.Errorf("tuik: %s contains whitespace or a control character", name)
	}
	return nil
}
