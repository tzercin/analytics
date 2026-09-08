package tuik

import (
	"encoding/xml"
	"fmt"
	"io"
)

// GenericData is an SDMX-ML 2.1 Generic Data message.
type GenericData struct {
	Header   GenericHeader
	DataSets []GenericDataSet
}

// GenericHeader contains the provenance fields exposed in an SDMX message
// header.
type GenericHeader struct {
	ID       string
	Test     bool
	Prepared string
	Sender   string
	Receiver string
}

// GenericDataSet contains series from an SDMX Generic Data message.
type GenericDataSet struct {
	Attributes map[string]string
	Series     []GenericSeries
}

// GenericSeries contains a series key, attributes, and observations. Values
// remain strings so callers can distinguish source status codes before numeric
// conversion.
type GenericSeries struct {
	Key          map[string]string
	Attributes   map[string]string
	Observations []GenericObservation
}

// GenericObservation is one observation in a generic series.
type GenericObservation struct {
	DimensionID string
	Dimension   string
	Value       string
	Attributes  map[string]string
}

type xmlGenericData struct {
	XMLName  xml.Name            `xml:"GenericData"`
	Header   xmlGenericHeader    `xml:"Header"`
	DataSets []xmlGenericDataSet `xml:"DataSet"`
}

type xmlGenericHeader struct {
	ID       string `xml:"ID"`
	Test     bool   `xml:"Test"`
	Prepared string `xml:"Prepared"`
	Sender   xmlID  `xml:"Sender"`
	Receiver xmlID  `xml:"Receiver"`
}

type xmlID struct {
	ID string `xml:"id,attr"`
}

type xmlGenericDataSet struct {
	Attributes xmlValues          `xml:"Attributes"`
	Series     []xmlGenericSeries `xml:"Series"`
}

type xmlGenericSeries struct {
	Key        xmlValues               `xml:"SeriesKey"`
	Attributes xmlValues               `xml:"Attributes"`
	Obs        []xmlGenericObservation `xml:"Obs"`
}

type xmlGenericObservation struct {
	Dimension  xmlValue  `xml:"ObsDimension"`
	Value      xmlValue  `xml:"ObsValue"`
	Attributes xmlValues `xml:"Attributes"`
}

type xmlValues struct {
	Values []xmlValue `xml:"Value"`
}

type xmlValue struct {
	ID    string `xml:"id,attr"`
	Value string `xml:"value,attr"`
}

// DecodeGenericData decodes an SDMX-ML 2.1 Generic Data response. It does not
// interpret observation values or collapse missing, provisional, suppressed,
// or other status attributes.
func DecodeGenericData(reader io.Reader) (*GenericData, error) {
	if reader == nil {
		return nil, fmt.Errorf("tuik: SDMX reader is nil")
	}

	var message xmlGenericData
	if err := xml.NewDecoder(reader).Decode(&message); err != nil {
		return nil, fmt.Errorf("tuik: decode SDMX Generic Data: %w", err)
	}

	result := &GenericData{
		Header: GenericHeader{
			ID:       message.Header.ID,
			Test:     message.Header.Test,
			Prepared: message.Header.Prepared,
			Sender:   message.Header.Sender.ID,
			Receiver: message.Header.Receiver.ID,
		},
		DataSets: make([]GenericDataSet, 0, len(message.DataSets)),
	}
	for _, dataSet := range message.DataSets {
		converted := GenericDataSet{
			Attributes: valuesMap(dataSet.Attributes),
			Series:     make([]GenericSeries, 0, len(dataSet.Series)),
		}
		for _, series := range dataSet.Series {
			convertedSeries := GenericSeries{
				Key:          valuesMap(series.Key),
				Attributes:   valuesMap(series.Attributes),
				Observations: make([]GenericObservation, 0, len(series.Obs)),
			}
			for _, observation := range series.Obs {
				convertedSeries.Observations = append(convertedSeries.Observations, GenericObservation{
					DimensionID: observation.Dimension.ID,
					Dimension:   observation.Dimension.Value,
					Value:       observation.Value.Value,
					Attributes:  valuesMap(observation.Attributes),
				})
			}
			converted.Series = append(converted.Series, convertedSeries)
		}
		result.DataSets = append(result.DataSets, converted)
	}
	return result, nil
}

func valuesMap(values xmlValues) map[string]string {
	result := make(map[string]string, len(values.Values))
	for _, value := range values.Values {
		result[value.ID] = value.Value
	}
	return result
}
