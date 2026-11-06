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
//
// "server -healthcheck" asks a running server on LISTEN_ADDR for /healthz and exits 0 when it
// answers, else 1: the Docker HEALTHCHECK, as the scratch image has no shell or curl.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
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
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		if err := healthcheck(listenAddr(os.Getenv), &http.Client{Timeout: 3 * time.Second}); err != nil {
			fmt.Fprintln(os.Stderr, "unhealthy:", err)
			os.Exit(1)
		}
		return
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	addr := listenAddr(os.Getenv)
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

func listenAddr(getenv func(string) string) string {
	if a := getenv("LISTEN_ADDR"); a != "" {
		return a
	}
	return ":8080"
}

// healthcheck asks the server listening on addr for /healthz. It checks liveness only: a stale
// NS feed is reported in the body for the monitor, not treated as unhealthy here, so a slow NS
// never gets the container flagged or restarted.
func healthcheck(addr string, client *http.Client) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("LISTEN_ADDR %q: %w", addr, err)
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	res, err := client.Get("http://" + net.JoinHostPort(host, port) + "/healthz")
	if err != nil {
		return err
	}
	defer res.Body.Close()
	var body struct {
		OK bool `json:"ok"`
	}
	if res.StatusCode != http.StatusOK || json.NewDecoder(res.Body).Decode(&body) != nil || !body.OK {
		return fmt.Errorf("/healthz answered HTTP %d", res.StatusCode)
	}
	return nil
}
