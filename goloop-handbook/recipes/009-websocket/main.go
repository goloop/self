// Recipe 009: real-time updates over a WebSocket.
//
// The task: keep a connection open and push messages over it, both ways. One
// module, on both ends: websocket upgrades a request on the server and dials
// one from a client, over plain net/http. Three examples:
//
//	A. echo    - the server reads a text message and writes it back;
//	B. rpc     - a JSON request in, a JSON reply out (WriteJSON/ReadJSON);
//	C. stream  - the server pushes a sequence of messages the client reads;
//	D. hub     - one message fanned out to every connected client (broadcast);
//	E. close   - the server closes with a status code the client can read.
package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/goloop/websocket"
)

// hub is a set of live connections with a broadcast. A real chat or live feed
// is this plus rooms; the core is a guarded map and a fan-out write.
type hub struct {
	mu    sync.Mutex
	conns map[*websocket.Conn]bool
}

func newHub() *hub { return &hub{conns: map[*websocket.Conn]bool{}} }

func (h *hub) add(c *websocket.Conn) {
	h.mu.Lock()
	h.conns[c] = true
	h.mu.Unlock()
}

func (h *hub) remove(c *websocket.Conn) {
	h.mu.Lock()
	delete(h.conns, c)
	h.mu.Unlock()
}

// broadcast writes data to every connection, dropping any that error.
func (h *hub) broadcast(data []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.conns {
		if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
			delete(h.conns, c)
		}
	}
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "recipe:", err)
		os.Exit(1)
	}
}

func run() error {
	// A test server hosts three WebSocket endpoints.
	srv := httptest.NewServer(handler())
	defer srv.Close()
	base := "ws" + strings.TrimPrefix(srv.URL, "http")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Example A: echo. Dial, send a message, read it back.
	fmt.Println("A. echo (WriteMessage / ReadMessage):")
	c, _, err := websocket.Dial(ctx, base+"/echo")
	if err != nil {
		return err
	}
	_ = c.WriteMessage(websocket.TextMessage, []byte("hello"))
	_, msg, _ := c.ReadMessage()
	fmt.Printf("   sent %q, received %q\n", "hello", msg)
	c.Close()

	// Example B: rpc. Send a JSON request, read a JSON reply.
	fmt.Println("B. json rpc (WriteJSON / ReadJSON):")
	c, _, err = websocket.Dial(ctx, base+"/rpc")
	if err != nil {
		return err
	}
	_ = c.WriteJSON(map[string]int{"a": 2, "b": 3})
	var reply struct {
		Sum int `json:"sum"`
	}
	_ = c.ReadJSON(&reply)
	fmt.Printf("   a=2 b=3 -> sum=%d\n", reply.Sum)
	c.Close()

	// Example C: server push. The server streams a few messages; the client
	// reads them until the connection closes.
	fmt.Println("C. server push (a stream of messages):")
	c, _, err = websocket.Dial(ctx, base+"/stream")
	if err != nil {
		return err
	}
	for {
		_, m, err := c.ReadMessage()
		if err != nil {
			break
		}
		fmt.Printf("   push: %s\n", m)
	}
	c.Close()

	// Example D: broadcast. Two clients join the hub; one sends a message and
	// the server fans it out to every connection, so both receive it.
	fmt.Println("D. broadcast to a hub (fan-out to every client):")
	a, _, err := websocket.Dial(ctx, base+"/hub")
	if err != nil {
		return err
	}
	defer a.Close()
	b, _, err := websocket.Dial(ctx, base+"/hub")
	if err != nil {
		return err
	}
	defer b.Close()
	_ = a.WriteMessage(websocket.TextMessage, []byte("hello all"))
	_, ma, _ := a.ReadMessage()
	_, mb, _ := b.ReadMessage()
	fmt.Printf("   client A sent %q; A got %q, B got %q\n", "hello all", ma, mb)

	// Example E: a graceful close with a status code. The server sends one
	// message, then closes with CloseNormalClosure; the client reads the
	// message, and the next read reports the close code instead of a raw EOF.
	fmt.Println("E. graceful close with a status code:")
	c, _, err = websocket.Dial(ctx, base+"/bye")
	if err != nil {
		return err
	}
	_, first, _ := c.ReadMessage()
	_, _, closeErr := c.ReadMessage()
	fmt.Printf("   message %q, then normal-closure=%v\n",
		first, websocket.IsCloseError(closeErr, websocket.CloseNormalClosure))
	c.Close()
	return nil
}

// handler serves the WebSocket endpoints.
func handler() http.Handler {
	mux := http.NewServeMux()
	h := newHub()

	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Upgrade(w, r)
		if err != nil {
			return
		}
		defer conn.Close()
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		_ = conn.WriteMessage(websocket.TextMessage, data)
	})

	mux.HandleFunc("/rpc", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Upgrade(w, r)
		if err != nil {
			return
		}
		defer conn.Close()
		var req struct{ A, B int }
		if err := conn.ReadJSON(&req); err != nil {
			return
		}
		_ = conn.WriteJSON(map[string]int{"sum": req.A + req.B})
	})

	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Upgrade(w, r)
		if err != nil {
			return
		}
		defer conn.Close()
		for i := 1; i <= 3; i++ {
			if err := conn.WriteMessage(websocket.TextMessage,
				[]byte(fmt.Sprintf("tick %d", i))); err != nil {
				return
			}
		}
	})

	// /hub: every connection joins the hub; each message it sends is broadcast
	// to all connected clients, the sender included.
	mux.HandleFunc("/hub", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Upgrade(w, r)
		if err != nil {
			return
		}
		h.add(conn)
		defer func() {
			h.remove(conn)
			conn.Close()
		}()
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			h.broadcast(data)
		}
	})

	// /bye: send one message, then close cleanly with a status code the client
	// can distinguish from an abrupt disconnect.
	mux.HandleFunc("/bye", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Upgrade(w, r)
		if err != nil {
			return
		}
		defer conn.Close()
		if err := conn.WriteMessage(websocket.TextMessage, []byte("closing now")); err != nil {
			return
		}
		_ = conn.CloseWithStatus(websocket.CloseNormalClosure, "bye")
	})

	return mux
}
