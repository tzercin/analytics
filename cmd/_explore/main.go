package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/tzercin/analytics/tuik"
)

func main() {
	mode := flag.String("mode", "data", "data or structure")
	dataflow := flag.String("dataflow", "", "dataflow id (data mode)")
	version := flag.String("version", "1.0", "version")
	key := flag.String("key", "", "series key (data mode)")
	start := flag.String("start", "", "startPeriod")
	end := flag.String("end", "", "endPeriod")
	resource := flag.String("resource", "datastructure", "structure resource")
	id := flag.String("id", "", "structure id")
	refs := flag.String("refs", "all", "structure references")
	out := flag.String("out", "", "write raw response bytes to this file")
	flag.Parse()

	apiKey := os.Getenv("TUIK_API_KEY")
	if apiKey == "" {
		log.Fatal("TUIK_API_KEY not set")
	}
	tokens, err := tuik.NewAPIKeyTokenSource(apiKey)
	if err != nil {
		log.Fatal(err)
	}
	client, err := tuik.NewClient(tokens)
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	if *mode == "structure" {
		resp, err := client.Structure(ctx, tuik.StructureQuery{
			Resource: tuik.StructureResource(*resource), Agency: "TR", ID: *id,
			Version: *version, Detail: "full", References: *refs,
		})
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		if *out != "" {
			if err := os.WriteFile(*out, body, 0o644); err != nil {
				log.Fatal(err)
			}
		}
		log.Printf("structure %s/%s: %d bytes -> %s", *resource, *id, len(body), *out)
		return
	}

	resp, err := client.Data(ctx, tuik.DataQuery{
		Agency: "TR", Dataflow: *dataflow, Version: *version,
		Key: *key, StartPeriod: *start, EndPeriod: *end,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	if *out != "" {
		if err := os.WriteFile(*out, body, 0o644); err != nil {
			log.Fatal(err)
		}
	}
	msg, err := tuik.DecodeGenericData(bytes.NewReader(body))
	if err != nil {
		log.Fatalf("decode: %v (raw %d bytes written to %s)", err, len(body), *out)
	}
	fmt.Printf("RAW_BYTES=%d DATASETS=%d\n", len(body), len(msg.DataSets))
	for di, ds := range msg.DataSets {
		fmt.Printf("DATASET[%d] attrs=%v series=%d\n", di, ds.Attributes, len(ds.Series))
		for _, s := range ds.Series {
			ks := make([]string, 0, len(s.Key))
			for k := range s.Key {
				ks = append(ks, k)
			}
			sort.Strings(ks)
			var sb strings.Builder
			for _, k := range ks {
				fmt.Fprintf(&sb, "%s=%s ", k, s.Key[k])
			}
			obs := make([]tuik.GenericObservation, len(s.Observations))
			copy(obs, s.Observations)
			sort.Slice(obs, func(i, j int) bool { return obs[i].Dimension < obs[j].Dimension })
			n := len(obs)
			first, last := "-", "-"
			fv, lv := "", ""
			if n > 0 {
				first, fv = obs[0].Dimension, obs[0].Value
				last, lv = obs[n-1].Dimension, obs[n-1].Value
			}
			fmt.Printf("  SERIES n=%d [%s] first=%s:%s last=%s:%s\n", n, strings.TrimSpace(sb.String()), first, fv, last, lv)
		}
	}
}
