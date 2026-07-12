[« A service lifecycle](08-lifecycle.md) · [Contents](../main.md) · [Putting it together »](10-whole-stack.md)

---

# 09. Real-time with WebSockets

**Task.** Keep a connection open and send messages over it both ways: echo a
message, do a small request/response, push a stream of updates from the server
to the client, fan one message out to every connected client, and close a
connection cleanly with a status code.

**Module.** [`websocket`](https://github.com/goloop/websocket) is a from-scratch
RFC 6455 implementation on the standard library, with both ends: `Upgrade` on
the server and `Dial` on the client.

**Recipe.** [`recipes/009-websocket`](../recipes/009-websocket/)

## Example A - echo

On the server, `websocket.Upgrade` turns an HTTP request into a `*Conn`; then
`ReadMessage` and `WriteMessage` move frames. The client dials with
`websocket.Dial`:

```go
// server:
conn, _ := websocket.Upgrade(w, r)
_, data, _ := conn.ReadMessage()
_ = conn.WriteMessage(websocket.TextMessage, data)

// client:
c, _, _ := websocket.Dial(ctx, "ws://host/echo")
_ = c.WriteMessage(websocket.TextMessage, []byte("hello"))
_, msg, _ := c.ReadMessage() // "hello"
```

## Example B - a JSON request/response

`WriteJSON` and `ReadJSON` marshal and unmarshal for you, so a small RPC over
the socket is a few lines:

```go
// server:
var req struct{ A, B int }
_ = conn.ReadJSON(&req)
_ = conn.WriteJSON(map[string]int{"sum": req.A + req.B})

// client:
_ = c.WriteJSON(map[string]int{"a": 2, "b": 3})
var reply struct{ Sum int `json:"sum"` }
_ = c.ReadJSON(&reply) // reply.Sum == 5
```

## Example C - the server pushes a stream

The value of a socket is that the server can send without being asked. Here it
writes a sequence and closes; the client reads until the connection ends:

```go
// server:
for i := 1; i <= 3; i++ {
	_ = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("tick %d", i)))
}

// client:
for {
	_, m, err := c.ReadMessage()
	if err != nil {
		break
	}
	// m == "tick 1", "tick 2", "tick 3"
}
```

## Example D - broadcast to a hub

The reason to reach for a socket over polling is fan-out: one event delivered to
every connected client. A hub is a guarded set of connections plus a broadcast
write. Each client's handler registers it, then loops reading; a message from
any client is written to all of them:

```go
type hub struct {
	mu    sync.Mutex
	conns map[*websocket.Conn]bool
}

func (h *hub) broadcast(data []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.conns {
		if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
			delete(h.conns, c) // drop a dead connection
		}
	}
}

// in the /hub handler: h.add(conn); then for each message, h.broadcast(data)
```

Two clients join, one sends `"hello all"`, and both receive it. Add rooms (a
hub per key) and you have a chat or a live feed.

## Example E - a graceful close with a status code

A clean shutdown is not just dropping the socket. `CloseWithStatus` sends a
close frame with a code and reason; the client's next read reports that code, so
it can tell a normal close from a crash. `IsCloseError` inspects it:

```go
// server: send a last message, then close cleanly
_ = conn.WriteMessage(websocket.TextMessage, []byte("closing now"))
_ = conn.CloseWithStatus(websocket.CloseNormalClosure, "bye")

// client: read the message, then see the close code on the next read
_, msg, _ := c.ReadMessage() // "closing now"
_, _, err := c.ReadMessage()
websocket.IsCloseError(err, websocket.CloseNormalClosure) // true
```

## Execution report

The program hosts the endpoints and dials them as a client:

```text
$ go test ./...
ok  	goloop.one/handbook/009-websocket	0.005s

$ go run .
A. echo (WriteMessage / ReadMessage):
   sent "hello", received "hello"
B. json rpc (WriteJSON / ReadJSON):
   a=2 b=3 -> sum=5
C. server push (a stream of messages):
   push: tick 1
   push: tick 2
   push: tick 3
D. broadcast to a hub (fan-out to every client):
   client A sent "hello all"; A got "hello all", B got "hello all"
E. graceful close with a status code:
   message "closing now", then normal-closure=true
```

The echo returned the exact bytes; the JSON RPC computed the sum on the server
and read it back on the client; the stream delivered three server-initiated
messages over one connection; the hub fanned one client's message out to both
connections; and the graceful close let the client read a normal-closure code
instead of a bare error. The recipe passes under the race detector.

## What you learned

- `websocket.Upgrade(w, r)` upgrades on the server; `websocket.Dial(ctx, url)`
  connects from a client; both give a `*Conn`.
- `ReadMessage`/`WriteMessage` move raw frames; `ReadJSON`/`WriteJSON` marshal
  for you.
- A socket lets the server push without a request, which is the point of using
  one over polling. A hub (a guarded set of connections plus a broadcast) fans
  one message out to every client; add rooms for a chat or a live feed.
- `CloseWithStatus` closes with a code and reason; `IsCloseError` lets the other
  end tell a normal close from a crash.
- A body cap or a buffering timeout middleware must be skipped for an upgrade;
  the whole-stack chapter shows the pattern.

Next: put the pieces together into one service.

---

[« A service lifecycle](08-lifecycle.md) · [Contents](../main.md) · [Putting it together »](10-whole-stack.md)
