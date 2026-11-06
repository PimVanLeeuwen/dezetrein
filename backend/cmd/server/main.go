// Command server runs the dezetrein API.
//
// Environment:
//
//	LISTEN_ADDR  listen address (default :8080)
//	NS_MODE      "live" (default) calls NS with NS_API_KEY; without a key the NS routes answer
//	             503 and only the health check works. "fixture" replays the recordings in
//	             NS_FIXTURES (default testdata/ns) and needs no key.
//
// The version is set at build time with -ldflags "-X main.version=...".
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PimVanLeeuwen/dezetrein/backend/internal/api"
	"github.com/PimVanLeeuwen/dezetrein/backend/internal/live"
	"github.com/PimVanLeeuwen/dezetrein/backend/internal/ns"
)

var version = "dev"

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app := &api.Server{Version: version, Log: log}
	client, err := nsClient(os.Getenv)
	if err != nil {
		return err
	}
	if client != nil {
		poller := &live.Poller{
			Fetch: func(ctx context.Context) ([]ns.Vehicle, error) {
				v, _, err := client.Vehicles(ctx)
				return v, err
			},
			Log: log,
		}
		go poller.Run(ctx)
		app.Vehicles = poller
		app.NS = client
		log.Info("ns feed on", "mode", nsMode(os.Getenv))
	} else {
		log.Warn("NS_API_KEY is not set: the NS feed is off")
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           app.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	log.Info("server listening", "addr", addr, "version", version)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func nsMode(getenv func(string) string) string {
	if m := getenv("NS_MODE"); m != "" {
		return m
	}
	return "live"
}

// nsClient builds the NS client for NS_MODE. It returns nil without error in live mode when no
// key is set, so the server can still start (CI, local runs without a key).
func nsClient(getenv func(string) string) (*ns.Client, error) {
	switch mode := nsMode(getenv); mode {
	case "live":
		if key := getenv("NS_API_KEY"); key != "" {
			return ns.NewClient(key), nil
		}
		return nil, nil
	case "fixture":
		dir := getenv("NS_FIXTURES")
		if dir == "" {
			dir = "testdata/ns"
		}
		return ns.NewFixtureClient(dir)
	default:
		return nil, fmt.Errorf("NS_MODE %q: want live or fixture", mode)
	}
}
