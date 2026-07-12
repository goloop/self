// Recipe 008: run a service with a lifecycle.
//
// The task: start an HTTP server, expose liveness and readiness probes, log in
// a structured way, and shut down cleanly on a signal. Three small modules:
//
//	A. app     - an ordered start/stop lifecycle around your components;
//	B. observe - /healthz (liveness) and /readyz (readiness with checks);
//	C. log     - a shared, structured slog logger.
package main

import (
	"context"
	"net/http"
	"os"

	"github.com/goloop/app"
	"github.com/goloop/log/v2"
	"github.com/goloop/observe"
)

func main() {
	if err := run(context.Background()); err != nil {
		os.Exit(1)
	}
}

// run wires the service and blocks until the context is cancelled or a signal
// arrives, then shuts down gracefully.
func run(ctx context.Context) error {
	// C. one shared structured logger, backed by goloop/log.
	logger := log.NewSlog("svc")

	// B. observability: liveness is always ok; readiness runs its checks.
	reg := observe.New(
		observe.WithService("handbook-svc"),
		observe.WithBuildInfo(observe.BuildInfo{Version: "1.0.0"}),
	)
	reg.Check("clock", func(context.Context) error { return nil }) // a trivial dependency check

	mux := http.NewServeMux()
	mux.Handle("GET /healthz", reg.HealthHandler())
	mux.Handle("GET /readyz", reg.ReadyHandler())

	// A. the lifecycle: an HTTPServer component started and stopped by app.
	a := app.New("handbook-svc", app.WithLogger(logger))
	a.Use(app.HTTPServer("http", &http.Server{Addr: ":8085", Handler: mux}))
	a.OnStart(func(context.Context) error {
		logger.Info("service up", "addr", ":8085")
		return nil
	})
	a.OnStop(func(context.Context) error {
		logger.Info("service stopping")
		return nil
	})

	// Run blocks until ctx is cancelled or SIGINT/SIGTERM, then stops the
	// components in reverse order within the shutdown timeout.
	return a.Run(ctx)
}
