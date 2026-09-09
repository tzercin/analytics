// Command marketsales produces two separate monthly time-series charts for
// Türkiye-wide housing sales and passenger-car transfers.
package main

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"html"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/tzercin/analytics/tuik"
)

const (
	start       = "2013-01"
	end         = "2026-07"
	housingFlow = "DF_SATIS_SEKLI_SATIS_DURUMU_V3"
	housingKey  = "M._T.TR._Z._Z.2._Z._Z._T.MII_KSS._Z"
	carFlow     = "DF_MOTORLU_KARA_TASIT_DEVRI_YAPILAN_V3"
	carKey      = "TR.M.U_DYMKTS.M01+M02+M03+M04+M05+M06+M07+M08+M09+M10+M11+M12._Z._Z._Z._Z._Z.1.PN._Z._Z._Z"
)

type point struct {
	Period string
	Value  int64
}

type ratePoint struct {
	Period string
	Value  float64
}

func main() {
	out := "analyses/turkey-housing-and-car-sales"
	if len(os.Args) == 2 {
		out = os.Args[1]
	}
	if err := run(out); err != nil {
		log.Fatal(err)
	}
}

func run(out string) error {
	key := os.Getenv("TUIK_API_KEY")
	if key == "" {
		return errors.New("TUIK_API_KEY is not set")
	}
	ts, err := tuik.NewAPIKeyTokenSource(key)
	if err != nil {
		return err
	}
	c, err := tuik.NewClient(ts)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	h, err := get(ctx, c, housingFlow, housingKey)
	if err != nil {
		return fmt.Errorf("housing sales: %w", err)
	}
	v, err := get(ctx, c, carFlow, carKey)
	if err != nil {
		return fmt.Errorf("car transfers: %w", err)
	}
	if len(h) != 163 || len(v) != 163 {
		return fmt.Errorf("unexpected observation counts: housing=%d car=%d", len(h), len(v))
	}
	rates := monthlyPolicyRates()
	if err := os.MkdirAll(out, 0755); err != nil {
		return err
	}
	for _, x := range []struct {
		name, title, subtitle string
		p                     []point
	}{
		{"housing-sales", "Housing sales", "Türkiye · monthly registered sales", h},
		{"car-sales", "Car sales proxy", "Türkiye · monthly notarized passenger-car transfers", v},
	} {
		if err := writeCSV(filepath.Join(out, x.name+".csv"), x.p); err != nil {
			return err
		}
		if err := writeSVG(filepath.Join(out, x.name+".svg"), x.title, x.subtitle, x.p); err != nil {
			return err
		}
	}
	if err := writeRatesCSV(filepath.Join(out, "interest-rates.csv"), rates); err != nil {
		return err
	}
	if err := writeComparison(filepath.Join(out, "comparison.svg"), h, v, rates); err != nil {
		return err
	}
	return nil
}

func get(ctx context.Context, c *tuik.Client, flow, key string) ([]point, error) {
	r, err := c.Data(ctx, tuik.DataQuery{Agency: "TR", Dataflow: flow, Version: "1.0", Key: key, StartPeriod: start, EndPeriod: end})
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	m, err := tuik.DecodeGenericData(r.Body)
	if err != nil {
		return nil, err
	}
	if len(m.DataSets) != 1 || len(m.DataSets[0].Series) == 0 {
		return nil, fmt.Errorf("wanted at least one series")
	}
	var p []point
	seen := make(map[string]bool)
	for _, series := range m.DataSets[0].Series {
		for _, o := range series.Observations {
			f, err := strconv.ParseFloat(o.Value, 64)
			if err != nil || f < 0 || f != math.Trunc(f) {
				return nil, fmt.Errorf("invalid %s value %q", o.Dimension, o.Value)
			}
			if seen[o.Dimension] {
				return nil, fmt.Errorf("duplicate period %s", o.Dimension)
			}
			seen[o.Dimension] = true
			p = append(p, point{o.Dimension, int64(f)})
		}
	}
	sort.Slice(p, func(i, j int) bool { return p[i].Period < p[j].Period })
	for i := range p {
		want := time.Date(2013, time.January, 1, 0, 0, 0, 0, time.UTC).AddDate(0, i, 0).Format("2006-01")
		if p[i].Period != want {
			return nil, fmt.Errorf("period %d is %s, want %s", i, p[i].Period, want)
		}
	}
	return p, nil
}

func writeCSV(path string, p []point) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{"period", "count"})
	for _, x := range p {
		_ = w.Write([]string{x.Period, strconv.FormatInt(x.Value, 10)})
	}
	return w.Error()
}

func writeSVG(path, title, subtitle string, p []point) error {
	const W, H = 1200, 620
	const l, r, t, b = 92., 35., 112., 78.
	pw, ph := W-l-r, H-t-b
	max := int64(0)
	for _, x := range p {
		if x.Value > max {
			max = x.Value
		}
	}
	ymax := math.Ceil(float64(max)/100000) * 100000
	x := func(i int) float64 { return l + float64(i)*pw/float64(len(p)-1) }
	y := func(v int64) float64 { return t + ph - float64(v)/ymax*ph }
	s := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d"><rect width="100%%" height="100%%" fill="#f7f4ee"/><style>text{font-family:Inter,Arial,sans-serif;fill:#17232c}.title{font-size:32px;font-weight:700}.sub{font-size:17px;fill:#56636c}.axis{font-size:14px;fill:#657078}.note{font-size:13px;fill:#657078}</style><text x="%.0f" y="49" class="title">%s</text><text x="%.0f" y="78" class="sub">%s · January 2013–July 2026</text>`, W, H, W, H, l, html.EscapeString(title), l, html.EscapeString(subtitle))
	for i := 0; i <= 5; i++ {
		v := ymax * float64(i) / 5
		yy := t + ph - ph*float64(i)/5
		s += fmt.Sprintf(`<line x1="%.0f" y1="%.1f" x2="%.0f" y2="%.1f" stroke="#d8d5cf"/><text x="%.0f" y="%.1f" text-anchor="end" class="axis">%.0fk</text>`, l, yy, l+pw, yy, l-12, yy+5, v/1000)
	}
	for year := 2013; year <= 2026; year++ {
		i := (year - 2013) * 12
		if i >= len(p) {
			break
		}
		s += fmt.Sprintf(`<text x="%.1f" y="%.0f" text-anchor="middle" class="axis">%d</text>`, x(i), t+ph+30, year)
	}
	d := ""
	for i, q := range p {
		c := "L"
		if i == 0 {
			c = "M"
		}
		d += fmt.Sprintf("%s%.1f %.1f", c, x(i), y(q.Value))
	}
	s += fmt.Sprintf(`<path d="%s" fill="none" stroke="#d85b36" stroke-width="3" stroke-linejoin="round"/><circle cx="%.1f" cy="%.1f" r="5" fill="#d85b36"/><text x="%.1f" y="%.1f" font-size="17" font-weight="700">%s</text><text x="%.0f" y="%.0f" class="note">Source: TÜİK SDMX. “Car sales” is a transfer-count proxy, not an official sales measure.</text></svg>`, d, x(len(p)-1), y(p[len(p)-1].Value), x(len(p)-1)-8, y(p[len(p)-1].Value)-13, comma(p[len(p)-1].Value), l, float64(H-20))
	return os.WriteFile(path, []byte(s), 0644)
}

func comma(n int64) string {
	s := strconv.FormatInt(n, 10)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func writeComparison(path string, housing, cars []point, rates []ratePoint) error {
	if len(housing) != len(cars) || len(housing) != len(rates) || len(housing) < 2 {
		return errors.New("comparison series are not aligned")
	}
	const W, H = 1200, 650
	const l, r, t, b = 92., 45., 125., 88.
	pw, ph := W-l-r, H-t-b
	h0, c0 := float64(housing[0].Value), float64(cars[0].Value)
	max := 0.
	for i := range housing {
		if housing[i].Period != cars[i].Period {
			return fmt.Errorf("comparison period mismatch at %d", i)
		}
		max = math.Max(max, math.Max(float64(housing[i].Value)/h0*100, float64(cars[i].Value)/c0*100))
	}
	ymax := math.Ceil(max/50) * 50
	x := func(i int) float64 { return l + float64(i)*pw/float64(len(housing)-1) }
	y := func(v float64) float64 { return t + ph - v/ymax*ph }
	ry := func(v float64) float64 { return t + ph - v/50*ph }
	pathFor := func(p []point, base float64) string {
		d := ""
		for i, q := range p {
			cmd := "L"
			if i == 0 {
				cmd = "M"
			}
			d += fmt.Sprintf("%s%.1f %.1f", cmd, x(i), y(float64(q.Value)/base*100))
		}
		return d
	}
	s := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d"><rect width="100%%" height="100%%" fill="#f7f4ee"/><style>text{font-family:Inter,Arial,sans-serif;fill:#17232c}.title{font-size:32px;font-weight:700}.sub{font-size:17px;fill:#56636c}.axis{font-size:14px;fill:#657078}.note{font-size:13px;fill:#657078}.legend{font-size:15px;font-weight:600}</style><text x="%.0f" y="49" class="title">Housing sales and car-market activity</text><text x="%.0f" y="78" class="sub">Indexed monthly counts · January 2013 = 100</text>`, W, H, W, H, l, l)
	for i := 0.; i <= ymax; i += 50 {
		yy := y(i)
		s += fmt.Sprintf(`<line x1="%.0f" y1="%.1f" x2="%.0f" y2="%.1f" stroke="#d8d5cf"/><text x="%.0f" y="%.1f" text-anchor="end" class="axis">%.0f</text>`, l, yy, l+pw, yy, l-12, yy+5, i)
	}
	for year := 2013; year <= 2026; year++ {
		i := (year - 2013) * 12
		if i >= len(housing) {
			break
		}
		s += fmt.Sprintf(`<text x="%.1f" y="%.0f" text-anchor="middle" class="axis">%d</text>`, x(i), t+ph+30, year)
	}
	s += fmt.Sprintf(`<path d="%s" fill="none" stroke="#d85b36" stroke-width="3" stroke-linejoin="round"/><path d="%s" fill="none" stroke="#247b88" stroke-width="3" stroke-linejoin="round"/>`, pathFor(housing, h0), pathFor(cars, c0))
	rd := ""
	for i, q := range rates {
		cmd := "L"
		if i == 0 {
			cmd = "M"
		}
		rd += fmt.Sprintf("%s%.1f %.1f", cmd, x(i), ry(q.Value))
		if i+1 < len(rates) {
			rd += fmt.Sprintf("L%.1f %.1f", x(i+1), ry(q.Value))
		}
	}
	s += fmt.Sprintf(`<path d="%s" fill="none" stroke="#7157a5" stroke-width="3" stroke-linejoin="round"/>`, rd)
	for i := 0; i <= 5; i++ {
		v := float64(i * 10)
		yy := ry(v)
		s += fmt.Sprintf(`<text x="%.0f" y="%.1f" class="axis">%.0f%%</text>`, l+pw+9, yy+5, v)
	}
	s += fmt.Sprintf(`<line x1="%.0f" y1="101" x2="%.0f" y2="101" stroke="#d85b36" stroke-width="4"/><text x="%.0f" y="106" class="legend">Housing</text><line x1="%.0f" y1="101" x2="%.0f" y2="101" stroke="#247b88" stroke-width="4"/><text x="%.0f" y="106" class="legend">Car transfers</text><line x1="%.0f" y1="101" x2="%.0f" y2="101" stroke="#7157a5" stroke-width="4"/><text x="%.0f" y="106" class="legend">Policy rate (right)</text>`, l, l+28, l+38, l+145, l+173, l+183, l+305, l+333, l+343)
	s += fmt.Sprintf(`<text x="%.0f" y="%.0f" class="note">Indexed counts compare relative change; policy rate is the month-end one-week repo rate.</text><text x="%.0f" y="%.0f" class="note">Sources: TÜİK SDMX (counts); TCMB (policy rate) · Counts are unadjusted</text></svg>`, l, float64(H-35), l, float64(H-16))
	return os.WriteFile(path, []byte(s), 0644)
}

func writeRatesCSV(path string, p []ratePoint) error {
	f, e := os.Create(path)
	if e != nil {
		return e
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{"period", "one_week_repo_rate_percent"})
	for _, q := range p {
		_ = w.Write([]string{q.Period, strconv.FormatFloat(q.Value, 'f', 2, 64)})
	}
	return w.Error()
}

func monthlyPolicyRates() []ratePoint {
	changes := []struct {
		date string
		rate float64
	}{
		{"2012-12-19", 5.50}, {"2013-04-17", 5.00}, {"2013-05-17", 4.50}, {"2014-01-29", 10.00}, {"2014-05-23", 9.50}, {"2014-06-25", 8.75}, {"2014-07-18", 8.25}, {"2015-01-21", 7.75}, {"2015-02-25", 7.50}, {"2016-11-25", 8.00}, {"2018-06-01", 16.50}, {"2018-06-08", 17.75}, {"2018-09-14", 24.00}, {"2019-07-26", 19.75}, {"2019-09-13", 16.50}, {"2019-10-25", 14.00}, {"2019-12-13", 12.00}, {"2020-01-17", 11.25}, {"2020-02-20", 10.75}, {"2020-03-18", 9.75}, {"2020-04-23", 8.75}, {"2020-05-22", 8.25}, {"2020-09-25", 10.25}, {"2020-11-20", 15.00}, {"2020-12-25", 17.00}, {"2021-03-19", 19.00}, {"2021-09-24", 18.00}, {"2021-10-22", 16.00}, {"2021-11-19", 15.00}, {"2021-12-17", 14.00}, {"2022-08-19", 13.00}, {"2022-09-23", 12.00}, {"2022-10-21", 10.50}, {"2022-11-25", 9.00}, {"2023-02-24", 8.50}, {"2023-06-23", 15.00}, {"2023-07-21", 17.50}, {"2023-08-25", 25.00}, {"2023-09-22", 30.00}, {"2023-10-27", 35.00}, {"2023-11-24", 40.00}, {"2023-12-22", 42.50}, {"2024-01-26", 45.00}, {"2024-03-22", 50.00}, {"2024-12-27", 47.50}, {"2025-01-24", 45.00}, {"2025-03-07", 42.50}, {"2025-04-18", 46.00}, {"2025-07-25", 43.00}, {"2025-09-12", 40.50}, {"2025-10-24", 39.50}, {"2025-12-12", 38.00}, {"2026-01-23", 37.00},
	}
	var out []ratePoint
	ci := 0
	for i := 0; i < 163; i++ {
		d := time.Date(2013, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, i+1, 0).AddDate(0, 0, -1)
		for ci+1 < len(changes) && changes[ci+1].date <= d.Format("2006-01-02") {
			ci++
		}
		out = append(out, ratePoint{d.Format("2006-01"), changes[ci].rate})
	}
	return out
}
