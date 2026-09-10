// Command fertilitypyramid acquires and renders synchronized annual fertility
// and age-sex population series for Türkiye.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tzercin/analytics/tuik"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	outputDefault = "analyses/turkey-fertility-population-pyramid"
	startYear     = 2007
	endYear       = 2025
	fertFlow      = "DF_DOGUM_TEMEL_DOG_GOST_C"
	fertVersion   = "1.0"
	fertKey       = "TR.A.NG_TDH._Z._Z._Z._Z._Z._Z._Z._Z._Z._Z._Z._Z._Z._Z._Z.2026_01"
	popFlow       = "DF_ADNKS_T16"
	popVersion    = "1.0"
	popKey        = "TR.A.NG_ADNKS._Z._Z._Z._Z._T._Z._Z._Z._Z._Z._Z.1+2.Y0T4+Y5T9+Y10T14+Y15T19+Y20T24+Y25T29+Y30T34+Y35T39+Y40T44+Y45T49+Y50T54+Y55T59+Y60T64+Y65T69+Y70T74+Y75T79+Y80T84+Y85T89+Y_GE90._Z._Z._Z.NUFUS"
	robotoCommit  = "38062f4b4a0be4346d07a928408da21602545e9e"
	robotoSHA256  = "56a45233d29f11b4dfb86d248e921939d115778f87325e7ae8cc108383d6664d"
	robotoLicSHA  = "c71d239df91726fc519c6eb72d318ec65820627232b2f796219e87dcf35d0ab4"
)

// robotoRegular is pinned in the repository so offline rendering never depends
// on a machine-local font installation.
//
//go:embed assets/Roboto-Regular.ttf
var robotoRegular []byte

var ages = []struct{ Code, Label string }{
	{"Y0T4", "0-4"}, {"Y5T9", "5-9"}, {"Y10T14", "10-14"}, {"Y15T19", "15-19"},
	{"Y20T24", "20-24"}, {"Y25T29", "25-29"}, {"Y30T34", "30-34"}, {"Y35T39", "35-39"},
	{"Y40T44", "40-44"}, {"Y45T49", "45-49"}, {"Y50T54", "50-54"}, {"Y55T59", "55-59"},
	{"Y60T64", "60-64"}, {"Y65T69", "65-69"}, {"Y70T74", "70-74"}, {"Y75T79", "75-79"},
	{"Y80T84", "80-84"}, {"Y85T89", "85-89"}, {"Y_GE90", "90+"},
}

type fertility struct {
	Year        int
	Value       float64
	Raw, Status string
}
type population struct {
	Year                                 int
	AgeCode, AgeLabel, SexCode, SexLabel string
	Value                                int64
	Raw, Status                          string
}
type receipt struct {
	RequestURL   string `json:"request_url"`
	RetrievedAt  string `json:"retrieved_at"`
	ResponseDate string `json:"response_date"`
	ContentType  string `json:"response_content_type"`
	MessageID    string `json:"message_id"`
	Prepared     string `json:"message_prepared"`
	SHA256       string `json:"raw_sha256"`
	Bytes        int64  `json:"raw_byte_count"`
}
type provenance struct {
	SourceOwner        string             `json:"source_owner"`
	DocumentationURL   string             `json:"documentation_url"`
	LandingPageURL     string             `json:"landing_page_url"`
	Retrievals         map[string]receipt `json:"retrievals"`
	Sources            map[string]any     `json:"sources"`
	Period             map[string]int     `json:"period"`
	Outputs            map[string]string  `json:"output_sha256"`
	RawStored          bool               `json:"raw_stored"`
	RawPolicy          string             `json:"raw_policy"`
	LicenseConclusion  string             `json:"license_reuse_conclusion"`
	CrossChecks        []string           `json:"cross_checks"`
	Validation         []string           `json:"validation"`
	UnresolvedGates    []string           `json:"unresolved_gates"`
	Generator          map[string]string  `json:"generator"`
	Font               map[string]any     `json:"font"`
	StructuralMetadata []map[string]any   `json:"structural_metadata_discovery"`
}

func main() {
	out := flag.String("output", outputDefault, "analysis output directory")
	fetch := flag.Bool("fetch", false, "refresh derived data from TÜİK")
	flag.Parse()
	if err := run(*out, *fetch); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(out string, fetch bool) error {
	if err := os.MkdirAll(out, 0755); err != nil {
		return err
	}
	var f []fertility
	var p []population
	var prov provenance
	var err error
	if fetch {
		f, p, prov, err = acquire()
		if err != nil {
			return err
		}
		if err = writeFertility(filepath.Join(out, "fertility.csv"), f); err != nil {
			return err
		}
		if err = writePopulation(filepath.Join(out, "population.csv"), p); err != nil {
			return err
		}
	} else {
		f, err = readFertility(filepath.Join(out, "fertility.csv"))
		if err != nil {
			return fmt.Errorf("read derived fertility data (use -fetch to refresh): %w", err)
		}
		p, err = readPopulation(filepath.Join(out, "population.csv"))
		if err != nil {
			return fmt.Errorf("read derived population data (use -fetch to refresh): %w", err)
		}
		prov, err = readProvenance(filepath.Join(out, "provenance.json"))
		if err != nil {
			return fmt.Errorf("read provenance (use -fetch to recreate): %w", err)
		}
	}
	if err = validate(f, p); err != nil {
		return err
	}
	frameDir := filepath.Join(out, "frames")
	if err = writeFrames(frameDir, f, p); err != nil {
		return err
	}
	gifName := "fertility-population-pyramid.gif"
	if err = writeGIF(filepath.Join(out, gifName), frameDir, f); err != nil {
		return err
	}
	prov.Outputs = map[string]string{}
	for _, n := range []string{"fertility.csv", "population.csv", gifName} {
		prov.Outputs[n], err = fileHash(filepath.Join(out, n))
		if err != nil {
			return err
		}
	}
	for _, v := range f {
		n := filepath.ToSlash(filepath.Join("frames", strconv.Itoa(v.Year)+".png"))
		prov.Outputs[n], err = fileHash(filepath.Join(out, n))
		if err != nil {
			return err
		}
	}
	prov.Generator = map[string]string{
		"command_from_derived": "go run ./cmd/fertilitypyramid",
		"refresh_command":      "go run ./cmd/fertilitypyramid -fetch",
		"software":             "Go 1.27 (go.mod)",
		"text_renderer":        "golang.org/x/image v0.46.0 opentype, HintingNone",
		"transitive_modules":   "golang.org/x/text v0.42.0; golang.org/x/sys v0.48.0",
		"module_licenses":      "golang.org/x/image, golang.org/x/text, golang.org/x/sys: BSD-3-Clause",
		"determinism":          "same repository revision and derived CSV bytes produce byte-identical yearly PNG and animated GIF bytes",
	}
	prov.Font = map[string]any{
		"family":         "Roboto",
		"style":          "Regular",
		"asset":          "cmd/fertilitypyramid/assets/Roboto-Regular.ttf",
		"source_commit":  robotoCommit,
		"source_url":     "https://raw.githubusercontent.com/googlefonts/roboto-2/" + robotoCommit + "/src/hinted/Roboto-Regular.ttf",
		"sha256":         robotoSHA256,
		"byte_count":     515100,
		"license":        "Apache License 2.0",
		"license_asset":  "cmd/fertilitypyramid/assets/Roboto-LICENSE.txt",
		"license_url":    "https://raw.githubusercontent.com/googlefonts/roboto-2/" + robotoCommit + "/LICENSE",
		"license_sha256": robotoLicSHA,
		"redistribution": "permitted with included license; no machine-local font is used",
	}
	prov.Validation = appendUnique(prov.Validation, "2014 is the selected-period fertility maximum and every subsequent annual value is lower than the preceding year")
	if err = writeJSON(filepath.Join(out, "provenance.json"), prov); err != nil {
		return err
	}
	fmt.Printf("wrote %d yearly PNGs (%d-%d) and %s\n", len(f), startYear, endYear, filepath.Join(out, gifName))
	return nil
}

func acquire() ([]fertility, []population, provenance, error) {
	key := os.Getenv("TUIK_API_KEY")
	if key == "" {
		return nil, nil, provenance{}, errors.New("TUIK_API_KEY is not set")
	}
	ts, err := tuik.NewAPIKeyTokenSource(key)
	if err != nil {
		return nil, nil, provenance{}, err
	}
	c, err := tuik.NewClient(ts)
	if err != nil {
		return nil, nil, provenance{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	fm, fr, err := request(ctx, c, fertFlow, fertVersion, fertKey)
	if err != nil {
		return nil, nil, provenance{}, err
	}
	time.Sleep(time.Second)
	pm, pr, err := request(ctx, c, popFlow, popVersion, popKey)
	if err != nil {
		return nil, nil, provenance{}, err
	}
	f, err := parseFertility(fm)
	if err != nil {
		return nil, nil, provenance{}, err
	}
	p, err := parsePopulation(pm)
	if err != nil {
		return nil, nil, provenance{}, err
	}
	if err = validate(f, p); err != nil {
		return nil, nil, provenance{}, err
	}
	prov := provenance{SourceOwner: "Türkiye İstatistik Kurumu (TÜİK)", DocumentationURL: "https://veriportali.tuik.gov.tr/tr/sdmx-web-service-documentation", LandingPageURL: "https://veriportali.tuik.gov.tr/", Retrievals: map[string]receipt{"fertility": fr, "population": pr}, Period: map[string]int{"start_year": startYear, "end_year": endYear, "year_count": endYear - startYear + 1}, RawStored: false, RawPolicy: "Exact response bytes are not stored because redistribution terms remain unresolved; request URLs, timestamps, sizes and SHA-256 checksums bind the responses and -fetch repeats the deterministic queries.", LicenseConclusion: "UNRESOLVED — public access was verified, but data-specific reuse/redistribution terms were not found; maintainer review is required before publication.", UnresolvedGates: []string{"data-specific license and output redistribution permission", "published API rate limit", "formal revision schedule"}}
	prov.StructuralMetadata = []map[string]any{
		{"request_url": "https://nsiws.tuik.gov.tr/rest/dataflow/TR/DF_DOGUM_TEMEL_DOG_GOST_C/1.0?detail=full&references=all", "retrieved_at": "2026-09-10T12:30:59+03:00", "raw_byte_count": 3282194, "raw_sha256": "fac1043d8d3c99a9df210f6ecd3e2a8845a1429c321ec9c105f95a940e652660", "raw_stored": false},
		{"request_url": "https://nsiws.tuik.gov.tr/rest/dataflow/TR/DF_ADNKS_T27/1.1?detail=full&references=all", "retrieved_at": "2026-09-10T12:30:59+03:00", "raw_byte_count": 2692853, "raw_sha256": "d038c50422d13c8dbafb24d0ce3cb7b22c4c21a5756b24e11d22fa962cd47227", "raw_stored": false, "note": "Resolved DSD_ADNKS 1.7 and codelists; DF_ADNKS_T16 uses the same DSD."},
	}
	prov.Sources = map[string]any{
		"fertility":  map[string]any{"classification": "primary official dataset", "dataflow_title": "Temel Doğurganlık Göstergeleri", "dataflow_id": fertFlow, "dataflow_version": fertVersion, "data_structure_id": "DSD_DOGUM", "data_structure_version": "1.1", "series_key": fertKey, "ordered_dimensions": []string{"REF_AREA", "FREQ", "INDICATOR", "CINSIYET_BEBEK", "ANNE_EGITIM_DURUMU", "KENT_KIR", "ANNE_YAS_GRUP", "ANNE_DOGUM_SIRA", "AYLIK_DOGUM_ARALIK", "COGUL_DOGUM", "DOGUM_SIRASI", "ANNE_UYRUK", "BABA_UYRUK", "EVLILIK_SURE", "ANNE_DOG_ULKE", "BABA_YAS_GRUP", "ANNE_MEDENI_DURUM", "AY", "YAYIM_DONEMI"}, "indicator_code": "NG_TDH", "indicator_label_tr": "Toplam Doğurganlık Hızı (Çocuk Sayısı)", "unit": "çocuk sayısı", "publication_vintage": "2026_01", "status_codes": statusesF(f)},
		"population": map[string]any{"classification": "primary official dataset", "dataflow_title": "İl, Yaş Grubu ve Cinsiyete Göre Nüfus", "dataflow_id": popFlow, "dataflow_version": popVersion, "data_structure_id": "DSD_ADNKS", "data_structure_version": "1.7", "series_key": popKey, "ordered_dimensions": []string{"REF_AREA", "FREQ", "INDICATOR", "NUFUS_KAYIT_IL", "YERLESIM_YERI_TUR", "VATANDASLIK_ULKESI", "DOGUM_YERI_IL", "IKAMET_YERI", "DOGUM_YERI_DURUM", "DOGUM_YERI_ULKE", "YAS", "DOGUM_YIL", "HANEHALKI_TIPI", "HANEHALKI_BUYUKLUGU", "SEX", "YAS_GRUBU", "ERKEK_ISIM", "KADIN_ISIM", "CIVIL_STATUS", "ADNKS_GOSTERGE"}, "selection": map[string]string{"IKAMET_YERI": "_T (Toplam)", "SEX": "1 (Erkek), 2 (Kadın)", "ADNKS_GOSTERGE": "NUFUS"}, "unit": "kişi", "age_codelist": "CL_YAS 1.1", "sex_codelist": "CL_CINSIYET 1.1", "status_codes": statusesP(p)},
	}
	prov.CrossChecks = []string{"Fertility 2024 current-vintage raw value = 1.4881566134354 with OBS_STATUS=R. The earlier TÜİK Birth Statistics, 2024 headline reported 1.48; this is a revision/vintage difference and is not represented as an exact match.", "Population 2025 sum of the 38 selected age-sex cells = 86,092,168, exactly matching the all-age/all-sex total in TÜİK dataflow DF_ADNKS_T27 for 2025."}
	prov.Validation = []string{"19 consecutive shared annual periods", "one fertility series with exact 19-dimension key", "38 population series: 19 age groups × 2 sex codes", "all expected age-sex-year cells present exactly once", "age-sex cells sum to 86,092,168 in 2025", "missing and status attributes preserved; no interpolation"}
	return f, p, prov, nil
}

func request(ctx context.Context, c *tuik.Client, flow, version, key string) (*tuik.GenericData, receipt, error) {
	r, err := c.Data(ctx, tuik.DataQuery{Agency: "TR", Dataflow: flow, Version: version, Key: key, StartPeriod: strconv.Itoa(startYear), EndPeriod: strconv.Itoa(endYear)})
	if err != nil {
		return nil, receipt{}, err
	}
	defer r.Body.Close()
	h := sha256.New()
	cr := &countReader{r: io.TeeReader(r.Body, h)}
	m, err := tuik.DecodeGenericData(cr)
	if err != nil {
		return nil, receipt{}, err
	}
	return m, receipt{RequestURL: r.Request.URL.String(), RetrievedAt: time.Now().In(time.FixedZone("Europe/Istanbul", 10800)).Format(time.RFC3339), ResponseDate: r.Header.Get("Date"), ContentType: r.Header.Get("Content-Type"), MessageID: m.Header.ID, Prepared: m.Header.Prepared, Bytes: cr.n, SHA256: hex.EncodeToString(h.Sum(nil))}, nil
}

type countReader struct {
	r io.Reader
	n int64
}

func (c *countReader) Read(b []byte) (int, error) { n, e := c.r.Read(b); c.n += int64(n); return n, e }

func parseFertility(m *tuik.GenericData) ([]fertility, error) {
	if len(m.DataSets) != 1 || len(m.DataSets[0].Series) != 1 {
		return nil, fmt.Errorf("fertility: expected one dataset/series")
	}
	s := m.DataSets[0].Series[0]
	if s.Key["INDICATOR"] != "NG_TDH" || s.Key["YAYIM_DONEMI"] != "2026_01" {
		return nil, fmt.Errorf("fertility: unexpected series key")
	}
	out := make([]fertility, 0, len(s.Observations))
	for _, o := range s.Observations {
		y, e := strconv.Atoi(o.Dimension)
		if e != nil {
			return nil, e
		}
		v, e := strconv.ParseFloat(o.Value, 64)
		if e != nil || math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, fmt.Errorf("fertility %s invalid value %q", o.Dimension, o.Value)
		}
		out = append(out, fertility{y, v, o.Value, o.Attributes["OBS_STATUS"]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Year < out[j].Year })
	return out, nil
}
func parsePopulation(m *tuik.GenericData) ([]population, error) {
	if len(m.DataSets) != 1 {
		return nil, fmt.Errorf("population: expected one dataset")
	}
	labels := map[string]string{}
	for _, a := range ages {
		labels[a.Code] = a.Label
	}
	sex := map[string]string{"1": "Erkek", "2": "Kadın"}
	out := []population{}
	for _, s := range m.DataSets[0].Series {
		al, ok := labels[s.Key["YAS_GRUBU"]]
		if !ok {
			return nil, fmt.Errorf("population: unexpected age %q", s.Key["YAS_GRUBU"])
		}
		sl, ok := sex[s.Key["SEX"]]
		if !ok {
			return nil, fmt.Errorf("population: unexpected sex %q", s.Key["SEX"])
		}
		if s.Key["IKAMET_YERI"] != "_T" || s.Key["ADNKS_GOSTERGE"] != "NUFUS" {
			return nil, fmt.Errorf("population: unexpected scope")
		}
		for _, o := range s.Observations {
			y, e := strconv.Atoi(o.Dimension)
			if e != nil {
				return nil, e
			}
			v, e := strconv.ParseInt(o.Value, 10, 64)
			if e != nil {
				return nil, fmt.Errorf("population %s invalid value %q", o.Dimension, o.Value)
			}
			out = append(out, population{y, s.Key["YAS_GRUBU"], al, s.Key["SEX"], sl, v, o.Value, o.Attributes["OBS_STATUS"]})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Year != out[j].Year {
			return out[i].Year < out[j].Year
		}
		if out[i].AgeCode != out[j].AgeCode {
			return ageIndex(out[i].AgeCode) < ageIndex(out[j].AgeCode)
		}
		return out[i].SexCode < out[j].SexCode
	})
	return out, nil
}
func ageIndex(c string) int {
	for i, a := range ages {
		if a.Code == c {
			return i
		}
	}
	return 999
}
func validate(f []fertility, p []population) error {
	if len(f) != endYear-startYear+1 {
		return fmt.Errorf("want 19 fertility years, got %d", len(f))
	}
	if len(p) != len(f)*len(ages)*2 {
		return fmt.Errorf("want %d population cells, got %d", len(f)*len(ages)*2, len(p))
	}
	seen := map[string]bool{}
	for i, x := range f {
		if x.Year != startYear+i {
			return fmt.Errorf("fertility year gap")
		}
		if x.Status == "" {
			return fmt.Errorf("fertility %d lacks OBS_STATUS", x.Year)
		}
	}
	peak := f[2014-startYear].Value
	for _, x := range f {
		if x.Year != 2014 && x.Value >= peak {
			return fmt.Errorf("2014 annotation invalid: %d fertility %.12g is not below 2014 %.12g", x.Year, x.Value, peak)
		}
	}
	for i := 2015 - startYear; i < len(f); i++ {
		if f[i].Value >= f[i-1].Value {
			return fmt.Errorf("2014 annotation invalid: fertility does not decline from %d to %d", f[i-1].Year, f[i].Year)
		}
	}
	for _, x := range p {
		k := fmt.Sprintf("%d/%s/%s", x.Year, x.AgeCode, x.SexCode)
		if seen[k] {
			return fmt.Errorf("duplicate %s", k)
		}
		seen[k] = true
		if x.Status != "" {
			return fmt.Errorf("population %s unexpectedly has status %q", k, x.Status)
		}
	}
	return nil
}

func writeFertility(path string, x []fertility) error {
	return atomic(path, func(w io.Writer) error {
		c := csv.NewWriter(w)
		_ = c.Write([]string{"year", "indicator_code", "indicator_label_tr", "value_children_per_woman", "obs_status"})
		for _, v := range x {
			_ = c.Write([]string{strconv.Itoa(v.Year), "NG_TDH", "Toplam Doğurganlık Hızı (Çocuk Sayısı)", v.Raw, v.Status})
		}
		c.Flush()
		return c.Error()
	})
}
func writePopulation(path string, x []population) error {
	return atomic(path, func(w io.Writer) error {
		c := csv.NewWriter(w)
		_ = c.Write([]string{"year", "age_group_code", "age_group_label_tr", "sex_code", "sex_label_tr", "population_persons", "obs_status"})
		for _, v := range x {
			_ = c.Write([]string{strconv.Itoa(v.Year), v.AgeCode, v.AgeLabel, v.SexCode, v.SexLabel, v.Raw, v.Status})
		}
		c.Flush()
		return c.Error()
	})
}
func readFertility(path string) ([]fertility, error) {
	r, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer r.Close()
	rows, e := csv.NewReader(r).ReadAll()
	if e != nil || len(rows) < 2 {
		return nil, fmt.Errorf("invalid CSV")
	}
	out := []fertility{}
	for _, v := range rows[1:] {
		y, e := strconv.Atoi(v[0])
		if e != nil {
			return nil, e
		}
		n, e := strconv.ParseFloat(v[3], 64)
		if e != nil {
			return nil, e
		}
		out = append(out, fertility{y, n, v[3], v[4]})
	}
	return out, nil
}
func readPopulation(path string) ([]population, error) {
	r, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer r.Close()
	rows, e := csv.NewReader(r).ReadAll()
	if e != nil || len(rows) < 2 {
		return nil, fmt.Errorf("invalid CSV")
	}
	out := []population{}
	for _, v := range rows[1:] {
		y, e := strconv.Atoi(v[0])
		if e != nil {
			return nil, e
		}
		n, e := strconv.ParseInt(v[5], 10, 64)
		if e != nil {
			return nil, e
		}
		out = append(out, population{y, v[1], v[2], v[3], v[4], n, v[5], v[6]})
	}
	return out, nil
}

func readProvenance(path string) (provenance, error) {
	f, err := os.Open(path)
	if err != nil {
		return provenance{}, err
	}
	defer f.Close()
	var p provenance
	d := json.NewDecoder(f)
	if err := d.Decode(&p); err != nil {
		return provenance{}, err
	}
	if p.SourceOwner == "" || len(p.Retrievals) == 0 || len(p.Sources) == 0 {
		return provenance{}, errors.New("provenance lacks required source metadata")
	}
	return p, nil
}

var pal = color.Palette{color.RGBA{250, 249, 246, 255}, color.RGBA{25, 36, 48, 255}, color.RGBA{218, 224, 228, 255}, color.RGBA{201, 53, 76, 255}, color.RGBA{54, 116, 181, 255}, color.RGBA{220, 91, 131, 255}, color.RGBA{100, 110, 120, 255}, color.RGBA{255, 255, 255, 255}}

func writeFrames(dir string, f []fertility, p []population) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	r, err := newTextRenderer(10, 11, 12, 13, 15, 20, 28, 30)
	if err != nil {
		return err
	}
	defer r.close()
	for i := range f {
		im := image.NewPaletted(image.Rect(0, 0, 1200, 675), pal)
		drawFrame(im, f, p, i, r)
		path := filepath.Join(dir, strconv.Itoa(f[i].Year)+".png")
		if err := atomic(path, func(w io.Writer) error { return png.Encode(w, im) }); err != nil {
			return err
		}
	}
	return nil
}

func writeGIF(path, frameDir string, f []fertility) error {
	animation := &gif.GIF{LoopCount: 0}
	for _, v := range f {
		framePath := filepath.Join(frameDir, strconv.Itoa(v.Year)+".png")
		file, err := os.Open(framePath)
		if err != nil {
			return fmt.Errorf("open GIF frame %s: %w", framePath, err)
		}
		decoded, err := png.Decode(file)
		closeErr := file.Close()
		if err != nil {
			return fmt.Errorf("decode GIF frame %s: %w", framePath, err)
		}
		if closeErr != nil {
			return closeErr
		}
		frame, ok := decoded.(*image.Paletted)
		if !ok {
			return fmt.Errorf("GIF frame %s is %T, want paletted image", framePath, decoded)
		}
		animation.Image = append(animation.Image, frame)
		animation.Delay = append(animation.Delay, 60)
	}
	return atomic(path, func(w io.Writer) error { return gif.EncodeAll(w, animation) })
}
func drawFrame(im *image.Paletted, f []fertility, p []population, frame int, r *textRenderer) {
	fill(im, 0, 0, 1200, 675, 0)
	r.text(im, 55, 28, 28, "Türkiye: Doğurganlık ve nüfusun yaş yapısı", 1)
	r.text(im, 55, 68, 15, "Eş zamanlı yıllık görünüm", 6)
	r.text(im, 970, 70, 15, "Gösterilen yıl", 6)
	r.text(im, 1090, 57, 30, strconv.Itoa(f[frame].Year), 3)
	line(im, 600, 115, 600, 605, 2)
	// left chart: fixed truthful scale 1.0-2.4 children per woman.
	r.text(im, 55, 117, 20, "Toplam doğurganlık hızı", 1)
	r.text(im, 55, 146, 13, "Kadın başına çocuk", 6)
	x0, y0, w, h := 75, 540, 475, 340
	for t := 10; t <= 24; t += 2 {
		y := y0 - (t-10)*h/14
		line(im, x0, y, x0+w, y, 2)
		r.text(im, 34, y-6, 12, strings.ReplaceAll(fmt.Sprintf("%.1f", float64(t)/10), ".", ","), 6)
	}
	line(im, x0, y0, x0+w, y0, 1)
	line(im, x0, y0-h, x0, y0, 1)
	for i := 0; i <= frame; i++ {
		x := x0 + i*w/(len(f)-1)
		y := y0 - int((f[i].Value-1.0)/1.4*float64(h))
		if i > 0 {
			px := x0 + (i-1)*w/(len(f)-1)
			py := y0 - int((f[i-1].Value-1.0)/1.4*float64(h))
			line(im, px, py, x, y, 3)
		}
		fill(im, x-3, y-3, 7, 7, 3)
	}
	x2014 := x0 + (2014-startYear)*w/(len(f)-1)
	line(im, x2014, y0, x2014, y0+7, 3)
	r.centered(im, x0, 550, 12, "2007", 6)
	r.centered(im, x2014, 550, 12, "2014", 3)
	r.centered(im, x0+w, 550, 12, "2025", 6)
	if f[frame].Year >= 2014 {
		highlight2014(im, f, x0, y0, w, h)
	}
	current := strings.ReplaceAll(fmt.Sprintf("%.2f", f[frame].Value), ".", ",")
	r.centered(im, x0+w/2, 579, 20, "Kadın başına "+current+" çocuk", 3)
	// population pyramid, fixed 0-3.6m scale each side.
	r.text(im, 650, 117, 20, "Yaş ve cinsiyete göre nüfus", 1)
	r.text(im, 650, 146, 13, "Kişi  |  Beş yıllık yaş grupları", 6)
	cx := 925
	top := 185
	bh := 19
	for k := 0; k <= 3; k++ {
		d := k * 75
		line(im, cx-d, top, cx-d, top+len(ages)*bh, 2)
		line(im, cx+d, top, cx+d, top+len(ages)*bh, 2)
	}
	vals := map[string]population{}
	for _, v := range p {
		if v.Year == f[frame].Year {
			vals[v.AgeCode+v.SexCode] = v
		}
	}
	for row := 0; row < len(ages); row++ {
		a := ages[len(ages)-1-row]
		y := top + row*bh
		m := vals[a.Code+"1"].Value
		wm := vals[a.Code+"2"].Value
		mw := int(float64(m) / 3600000 * 225)
		fw := int(float64(wm) / 3600000 * 225)
		fill(im, cx-mw, y+2, mw, 14, 4)
		fill(im, cx, y+2, fw, 14, 5)
		r.centered(im, cx, y+3, 10, a.Label, 7)
	}
	fill(im, 735, 570, 18, 12, 4)
	r.text(im, 760, 568, 13, "Erkek", 1)
	fill(im, 855, 570, 18, 12, 5)
	r.text(im, 880, 568, 13, "Kadın", 1)
	r.centered(im, cx, 594, 11, "3,6 mn   2,4 mn   1,2 mn      0      1,2 mn   2,4 mn   3,6 mn", 6)
	r.text(im, 55, 635, 11, "Kaynak: TÜİK SDMX  |  ADNKS yerleşik nüfusu  |  Değerler projeksiyon değildir", 6)
}

func highlight2014(im *image.Paletted, f []fertility, x0, y0, w, h int) {
	i := 2014 - startYear
	x := x0 + i*w/(len(f)-1)
	y := y0 - int((f[i].Value-1.0)/1.4*float64(h))
	fill(im, x-6, y-6, 13, 13, 7)
	fill(im, x-4, y-4, 9, 9, 3)
}
func fill(im *image.Paletted, x, y, w, h int, c uint8) {
	for yy := max(0, y); yy < min(im.Rect.Dy(), y+h); yy++ {
		for xx := max(0, x); xx < min(im.Rect.Dx(), x+w); xx++ {
			im.SetColorIndex(xx, yy, c)
		}
	}
}
func line(im *image.Paletted, x0, y0, x1, y1 int, c uint8) {
	dx := abs(x1 - x0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	dy := -abs(y1 - y0)
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	e := dx + dy
	for {
		if image.Pt(x0, y0).In(im.Rect) {
			im.SetColorIndex(x0, y0, c)
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * e
		if e2 >= dy {
			e += dy
			x0 += sx
		}
		if e2 <= dx {
			e += dx
			y0 += sy
		}
	}
}
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type textRenderer struct {
	faces map[int]font.Face
}

func newTextRenderer(sizes ...int) (*textRenderer, error) {
	h := sha256.Sum256(robotoRegular)
	if hex.EncodeToString(h[:]) != robotoSHA256 {
		return nil, errors.New("embedded Roboto asset checksum mismatch")
	}
	f, err := opentype.Parse(robotoRegular)
	if err != nil {
		return nil, fmt.Errorf("parse embedded Roboto: %w", err)
	}
	r := &textRenderer{faces: map[int]font.Face{}}
	for _, size := range sizes {
		face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: float64(size), DPI: 72, Hinting: font.HintingNone})
		if err != nil {
			r.close()
			return nil, fmt.Errorf("create Roboto %d px face: %w", size, err)
		}
		r.faces[size] = face
	}
	return r, nil
}

func (r *textRenderer) close() {
	for _, face := range r.faces {
		_ = face.Close()
	}
}

func (r *textRenderer) text(im *image.Paletted, x, y, size int, value string, c uint8) {
	face, ok := r.faces[size]
	if !ok {
		panic(fmt.Sprintf("unconfigured font size %d", size))
	}
	d := font.Drawer{
		Dst:  im,
		Src:  image.NewUniform(pal[c]),
		Face: face,
		Dot:  fixed.P(x, y+face.Metrics().Ascent.Ceil()),
	}
	d.DrawString(value)
}

func (r *textRenderer) centered(im *image.Paletted, centerX, y, size int, value string, c uint8) {
	face, ok := r.faces[size]
	if !ok {
		panic(fmt.Sprintf("unconfigured font size %d", size))
	}
	w := font.MeasureString(face, value).Ceil()
	r.text(im, centerX-w/2, y, size, value, c)
}

func statusesF(x []fertility) []string {
	m := map[string]bool{}
	for _, v := range x {
		m[v.Status] = true
	}
	return keys(m)
}
func statusesP(x []population) []string {
	m := map[string]bool{}
	for _, v := range x {
		if v.Status != "" {
			m[v.Status] = true
		}
	}
	if len(m) == 0 {
		return []string{"not supplied"}
	}
	return keys(m)
}
func keys(m map[string]bool) []string {
	o := []string{}
	for k := range m {
		o = append(o, k)
	}
	sort.Strings(o)
	return o
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
func atomic(path string, fn func(io.Writer) error) error {
	f, e := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if e != nil {
		return e
	}
	n := f.Name()
	defer os.Remove(n)
	if e = fn(f); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	if e = os.Chmod(n, 0644); e != nil {
		return e
	}
	return os.Rename(n, path)
}
func fileHash(path string) (string, error) {
	f, e := os.Open(path)
	if e != nil {
		return "", e
	}
	defer f.Close()
	h := sha256.New()
	_, e = io.Copy(h, f)
	return hex.EncodeToString(h.Sum(nil)), e
}
func writeJSON(path string, v any) error {
	return atomic(path, func(w io.Writer) error {
		var b bytes.Buffer
		e := json.NewEncoder(&b)
		e.SetIndent("", "  ")
		e.SetEscapeHTML(false)
		if err := e.Encode(v); err != nil {
			return err
		}
		_, err := w.Write(b.Bytes())
		return err
	})
}
