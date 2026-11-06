// Command server runs the dezetrein API.
//
// Environment: LISTEN_ADDR (default :8080) and NS_API_KEY (without it the NS feed is off and
// only the health check works). The version is set at build time with
// -ldflags "-X main.version=...".
package main

import (
	"context"
	"errors"
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

	app := &api.Server{Version: version}
	if key := os.Getenv("NS_API_KEY"); key != "" {
		client := ns.NewClient(key)
		poller := &live.Poller{
			Fetch: func(ctx context.Context) ([]ns.Vehicle, error) {
				v, _, err := client.Vehicles(ctx)
				return v, err
			},
			Log: log,
		}
		go poller.Run(ctx)
		app.Vehicles = poller
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
