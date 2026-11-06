// Command nsrecord records real NS API responses into testdata/ns, for parser tests and fixture
// mode. It reads NS_API_KEY from the environment or from a .env file and never prints it.
// The recordings are committed (the NS terms allow it); responses carry no key or personal data.
//
//	go run ./cmd/nsrecord -envfile ../.env [-train 3045] [-station UT] [-out testdata/ns]
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/PimVanLeeuwen/dezetrein/backend/internal/envfile"
	"github.com/PimVanLeeuwen/dezetrein/backend/internal/ns"
)

func main() {
	envPath := flag.String("envfile", "../.env", "file with NS_API_KEY=...")
	out := flag.String("out", "testdata/ns", "output folder")
	train := flag.String("train", "", "train number to record (default: a moving intercity from the vehicle list)")
	station := flag.String("station", "UT", "station code for the departures recording")
	stationsOnly := flag.Bool("stations", false, "record only the station list, added to the existing index.json")
	flag.Parse()

	if err := envfile.Load(*envPath); err != nil {
		fmt.Fprintln(os.Stderr, "reading env file:", err)
		os.Exit(1)
	}
	key := os.Getenv("NS_API_KEY")
	if key == "" {
		fmt.Fprintln(os.Stderr, "NS_API_KEY is not set (environment or", *envPath+")")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	var (
		idx *Index
		err error
	)
	if *stationsOnly {
		idx, err = RecordStations(ctx, ns.NewClient(key), *out)
	} else {
		idx, err = Record(ctx, ns.NewClient(key), *out, *train, *station, time.Now())
	}
	if idx != nil {
		fmt.Printf("train %q, station %q\n", idx.Train, idx.Station)
		for _, e := range idx.Entries {
			fmt.Printf("  %-12s %-45s HTTP %3d %8d bytes  %s\n", e.Name, e.Path, e.Status, e.Bytes, e.Error)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
