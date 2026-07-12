// Recipe 002: a small JSON HTTP API.
//
// Three examples of the same three modules:
//
//	A. routing + JSON  - GET /health and GET /users/{id}, with a clean 404;
//	B. CRUD statuses   - POST returns 201 Created + Location, DELETE returns 204;
//	C. your middleware - a custom wrapper dropped into the standard chain.
//
// The modules: mux routes (net/http patterns), resp writes JSON and errors,
// middlewares is the chain of func(http.Handler) http.Handler wrappers.
package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
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

// store is a trivial concurrency-safe map, so POST and DELETE are honest.
type store struct {
	mu   sync.Mutex
	next int
	rows map[int]user
}

func newStore() *store {
	return &store{next: 2, rows: map[int]user{1: {ID: 1, Name: "Ada", Email: "ada@example.com"}}}
}

// newRouter builds the routes. mux is a thin, standard-library router: the
// patterns are exactly net/http.ServeMux patterns ("GET /path/{id}"), and the
// Router is itself an http.Handler.
func newRouter(s *store) http.Handler {
	r := mux.New()

	// Example A: routing + JSON responses + a path parameter + a clean 404.
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		_ = resp.JSON(w, resp.R{"status": "ok"})
	})
	r.Get("/users/{id}", func(w http.ResponseWriter, req *http.Request) {
		u, ok := s.get(atoi(mux.Param(req, "id")))
		if !ok {
			_ = resp.Error(w, http.StatusNotFound, "user not found")
			return
		}
		_ = resp.JSON(w, u)
	})

	// Example B: the right status code for each write. resp.Created sends 201
	// with a Location header; resp.NoContent sends 204 with an empty body.
	r.Post("/users", func(w http.ResponseWriter, req *http.Request) {
		var in user
		if err := json.NewDecoder(req.Body).Decode(&in); err != nil {
			_ = resp.Error(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		u := s.add(in)
		_ = resp.Created(w, "/users/"+strconv.Itoa(u.ID), u)
	})
	r.Delete("/users/{id}", func(w http.ResponseWriter, req *http.Request) {
		if !s.del(atoi(mux.Param(req, "id"))) {
			_ = resp.Error(w, http.StatusNotFound, "user not found")
			return
		}
		_ = resp.NoContent(w)
	})

	// Example C: wrap the router in a middleware chain. The order is
	// outermost-first, and a custom middleware (apiVersion) is just another
	// func(http.Handler) http.Handler in the same list as the built-ins.
	return middlewares.Handler(r,
		middlewares.RequestID(),
		middlewares.Recoverer(),
		middlewares.Logger(),
		middlewares.SecurityHeaders(),
		apiVersion("v1"),
	)
}

// apiVersion is a custom middleware: it stamps every response with an
// X-API-Version header. Your own middleware needs no special interface.
func apiVersion(v string) middlewares.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-API-Version", v)
			next.ServeHTTP(w, r)
		})
	}
}

func main() {
	srv := &http.Server{
		Addr:              ":8081",
		Handler:           newRouter(newStore()),
		ReadHeaderTimeout: 5 * time.Second,
	}
	_ = srv.ListenAndServe()
}

func (s *store) get(id int) (user, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.rows[id]
	return u, ok
}

func (s *store) add(in user) user {
	s.mu.Lock()
	defer s.mu.Unlock()
	in.ID = s.next
	s.next++
	s.rows[in.ID] = in
	return in
}

func (s *store) del(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rows[id]; !ok {
		return false
	}
	delete(s.rows, id)
	return true
}

func atoi(s string) int {
	if s == "" {
		return -1
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return -1
		}
		n = n*10 + int(r-'0')
	}
	return n
}
