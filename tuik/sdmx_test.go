package tuik

import (
	"strings"
	"testing"
)

func TestDecodeGenericDataPreservesKeysValuesAndStatuses(t *testing.T) {
	t.Parallel()

	const input = `<?xml version="1.0" encoding="utf-8"?>
<message:GenericData
 xmlns:message="http://www.sdmx.org/resources/sdmxml/schemas/v2_1/message"
 xmlns:generic="http://www.sdmx.org/resources/sdmxml/schemas/v2_1/data/generic">
 <message:Header>
  <message:ID>example</message:ID><message:Test>false</message:Test>
  <message:Prepared>2026-09-08T12:00:00+03:00</message:Prepared>
  <message:Sender id="TR"/><message:Receiver id="client"/>
 </message:Header>
 <message:DataSet>
  <generic:Attributes><generic:Value id="UNIT_MEASURE" value="PT"/></generic:Attributes>
  <generic:Series>
   <generic:SeriesKey><generic:Value id="FREQ" value="M"/><generic:Value id="DEGISIM" value="4"/></generic:SeriesKey>
   <generic:Attributes><generic:Value id="DECIMALS" value="2"/></generic:Attributes>
   <generic:Obs>
    <generic:ObsDimension id="TIME_PERIOD" value="2026-08"/>
    <generic:ObsValue value="31.51"/>
    <generic:Attributes><generic:Value id="OBS_STATUS" value="A"/></generic:Attributes>
   </generic:Obs>
   <generic:Obs>
    <generic:ObsDimension id="TIME_PERIOD" value="2026-09"/>
    <generic:ObsValue value=""/>
    <generic:Attributes><generic:Value id="OBS_STATUS" value="M"/></generic:Attributes>
   </generic:Obs>
  </generic:Series>
 </message:DataSet>
</message:GenericData>`

	message, err := DecodeGenericData(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if message.Header.ID != "example" || message.Header.Sender != "TR" || message.Header.Receiver != "client" || message.Header.Test {
		t.Fatalf("header = %#v", message.Header)
	}
	if message.Header.Prepared != "2026-09-08T12:00:00+03:00" {
		t.Fatalf("prepared = %v", message.Header.Prepared)
	}
	if len(message.DataSets) != 1 || message.DataSets[0].Attributes["UNIT_MEASURE"] != "PT" {
		t.Fatalf("datasets = %#v", message.DataSets)
	}
	series := message.DataSets[0].Series
	if len(series) != 1 || series[0].Key["FREQ"] != "M" || series[0].Key["DEGISIM"] != "4" || series[0].Attributes["DECIMALS"] != "2" {
		t.Fatalf("series = %#v", series)
	}
	observations := series[0].Observations
	if len(observations) != 2 || observations[0].DimensionID != "TIME_PERIOD" || observations[0].Dimension != "2026-08" || observations[0].Value != "31.51" || observations[0].Attributes["OBS_STATUS"] != "A" {
		t.Fatalf("observations = %#v", observations)
	}
	if observations[1].Value != "" || observations[1].Attributes["OBS_STATUS"] != "M" {
		t.Fatalf("missing observation was not preserved: %#v", observations[1])
	}
}

func TestDecodeGenericDataRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	if _, err := DecodeGenericData(nil); err == nil {
		t.Fatal("expected nil-reader error")
	}
	if _, err := DecodeGenericData(strings.NewReader("<broken")); err == nil {
		t.Fatal("expected XML error")
	}
	if _, err := DecodeGenericData(strings.NewReader("<Structure/>")); err == nil {
		t.Fatal("expected wrong-root error")
	}
}
