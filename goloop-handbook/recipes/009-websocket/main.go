// Recipe 009: real-time updates over a WebSocket.
//
// The task: keep a connection open and push messages over it, both ways. One
// module, on both ends: websocket upgrades a request on the server and dials
// one from a client, over plain net/http. Three examples:
//
//	A. echo    - the server reads a text message and writes it back;
//	B. rpc     - a JSON request in, a JSON reply out (WriteJSON/ReadJSON);
//	C. stream  - the server pushes a sequence of messages the client reads.
package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	"github.com/goloop/websocket"
)

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
	return nil
}

// handler serves the three WebSocket endpoints.
func handler() http.Handler {
	mux := http.NewServeMux()

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

	return mux
}
