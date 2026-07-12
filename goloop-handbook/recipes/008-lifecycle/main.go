// Recipe 008: run a service with a lifecycle.
//
// The task: start an HTTP server, expose liveness and readiness probes, log in
// a structured way, and shut down cleanly on a signal. Three small modules:
//
//	A. app     - an ordered start/stop lifecycle around your components;
//	B. observe - /healthz (liveness) and /readyz (readiness with checks);
//	C. log     - a shared, structured slog logger;
//	D. observe - a readiness check tied to a dependency: /readyz is 503 until
//	             the dependency is up, then 200;
//	E. log     - a request-scoped child logger via logger.With.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"sync/atomic"

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

	// D. a readiness check tied to a real dependency. dbUp starts false, so
	// /readyz reports 503 until the dependency is up; a check returning an error
	// fails readiness without touching liveness. An admin route flips the state
	// here; in production the flag follows a real connection.
	var dbUp atomic.Bool
	reg.Check("database", func(context.Context) error {
		if !dbUp.Load() {
			return errDependencyDown
		}
		return nil
	})

	mux := http.NewServeMux()
	mux.Handle("GET /healthz", reg.HealthHandler())
	mux.Handle("GET /readyz", reg.ReadyHandler())
	mux.HandleFunc("POST /admin/db/{state}", func(w http.ResponseWriter, r *http.Request) {
		dbUp.Store(r.PathValue("state") == "up")
		// E. a request-scoped child logger: With pins fields onto every line it
		// writes, so a handler's logs carry their own context.
		logger.With("route", "admin.db", "state", r.PathValue("state")).
			Info("dependency state changed")
		w.WriteHeader(http.StatusNoContent)
	})

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

// errDependencyDown is what the database readiness check returns while the
// dependency is not yet up, turning /readyz into a 503.
var errDependencyDown = errors.New("database: not ready")
