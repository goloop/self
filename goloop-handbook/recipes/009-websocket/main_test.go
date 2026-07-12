package main

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/goloop/websocket"
)

func TestEchoAndRPC(t *testing.T) {
	srv := httptest.NewServer(handler())
	defer srv.Close()
	base := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c, _, err := websocket.Dial(ctx, base+"/echo")
	if err != nil {
		t.Fatal(err)
	}
	_ = c.WriteMessage(websocket.TextMessage, []byte("hi"))
	_, msg, _ := c.ReadMessage()
	if string(msg) != "hi" {
		t.Errorf("echo = %q", msg)
	}
	c.Close()

	c, _, err = websocket.Dial(ctx, base+"/rpc")
	if err != nil {
		t.Fatal(err)
	}
	_ = c.WriteJSON(map[string]int{"a": 4, "b": 5})
	var reply struct {
		Sum int `json:"sum"`
	}
	_ = c.ReadJSON(&reply)
	if reply.Sum != 9 {
		t.Errorf("sum = %d", reply.Sum)
	}
	c.Close()
}

func TestStream(t *testing.T) {
	srv := httptest.NewServer(handler())
	defer srv.Close()
	base := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, base+"/stream")
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for {
		if _, _, err := c.ReadMessage(); err != nil {
			break
		}
		n++
	}
	if n != 3 {
		t.Errorf("stream messages = %d, want 3", n)
	}
}

// TestBroadcast covers example D: a message from one client reaches every
// connected client.
func TestBroadcast(t *testing.T) {
	srv := httptest.NewServer(handler())
	defer srv.Close()
	base := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	a, _, err := websocket.Dial(ctx, base+"/hub")
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, _, err := websocket.Dial(ctx, base+"/hub")
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	_ = a.WriteMessage(websocket.TextMessage, []byte("ping all"))
	_, ma, _ := a.ReadMessage()
	_, mb, _ := b.ReadMessage()
	if string(ma) != "ping all" || string(mb) != "ping all" {
		t.Errorf("broadcast: A=%q B=%q, want both \"ping all\"", ma, mb)
	}
}

// TestGracefulClose covers example E: after the last message the client sees a
// normal-closure close code, not a bare read error.
func TestGracefulClose(t *testing.T) {
	srv := httptest.NewServer(handler())
	defer srv.Close()
	base := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c, _, err := websocket.Dial(ctx, base+"/bye")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, msg, _ := c.ReadMessage(); string(msg) != "closing now" {
		t.Errorf("first message = %q", msg)
	}
	if _, _, err := c.ReadMessage(); !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		t.Errorf("close error = %v, want normal closure", err)
	}
}
