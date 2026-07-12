// Recipe 002: a small JSON HTTP API.
//
// The task: serve JSON over HTTP with clean routing, path parameters, correct
// status codes and a stack of cross-cutting concerns (request id, panic
// recovery, logging, security headers) - without a framework. goloop composes
// this from three small pieces built on net/http: mux (routing), resp
// (responses) and middlewares (the chain).
package main

import (
	"net/http"
	"time"

	"github.com/goloop/middlewares"
	"github.com/goloop/mux"
	"github.com/goloop/resp/v2"
)

// user is a tiny in-memory model, enough to show JSON in and out.
type user struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var users = map[int]user{
	1: {ID: 1, Name: "Ada", Email: "ada@example.com"},
}

// newRouter builds the routes. mux is a thin, standard-library router: the
// patterns are exactly net/http.ServeMux patterns ("GET /path/{id}"), and the
// Router is itself an http.Handler.
func newRouter() http.Handler {
	r := mux.New()

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		_ = resp.JSON(w, resp.R{"status": "ok"})
	})

	// A path parameter, read with mux.Param, plus a 404 with a clean body.
	r.Get("/users/{id}", func(w http.ResponseWriter, req *http.Request) {
		id := atoi(mux.Param(req, "id"))
		u, ok := users[id]
		if !ok {
			_ = resp.Error(w, http.StatusNotFound, "user not found")
			return
		}
		_ = resp.JSON(w, u)
	})

	// Wrap the router in a middleware chain. The order is outermost-first:
	// a request id, then panic recovery, then request logging, then security
	// headers - each a plain func(http.Handler) http.Handler.
	return middlewares.Handler(r,
		middlewares.RequestID(),
		middlewares.Recoverer(),
		middlewares.Logger(),
		middlewares.SecurityHeaders(),
	)
}

func main() {
	srv := &http.Server{
		Addr:              ":8081",
		Handler:           newRouter(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	_ = srv.ListenAndServe()
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return -1
		}
		n = n*10 + int(r-'0')
	}
	if s == "" {
		return -1
	}
	return n
}
