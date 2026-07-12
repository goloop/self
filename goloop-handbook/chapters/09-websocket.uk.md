[« Життєвий цикл сервісу](08-lifecycle.uk.md) · [Зміст](../main.uk.md) · [Складаємо все разом »](10-whole-stack.uk.md)

---

# 09. Реальний час на WebSocket

**Задача.** Тримати з'єднання відкритим і слати повідомлення обома напрямками:
відлунити повідомлення, зробити маленький запит/відповідь і пушити потік
оновлень із сервера до клієнта.

**Модуль.** [`websocket`](https://github.com/goloop/websocket) - реалізація
RFC 6455 з нуля на стандартній бібліотеці, з обома кінцями: `Upgrade` на сервері
й `Dial` на клієнті.

**Рецепт.** [`recipes/009-websocket`](../recipes/009-websocket/)

## Приклад A - відлуння

На сервері `websocket.Upgrade` перетворює HTTP-запит на `*Conn`; далі
`ReadMessage` і `WriteMessage` рухають кадри. Клієнт під'єднується через
`websocket.Dial`:

```go
// сервер:
conn, _ := websocket.Upgrade(w, r)
_, data, _ := conn.ReadMessage()
_ = conn.WriteMessage(websocket.TextMessage, data)

// клієнт:
c, _, _ := websocket.Dial(ctx, "ws://host/echo")
_ = c.WriteMessage(websocket.TextMessage, []byte("hello"))
_, msg, _ := c.ReadMessage() // "hello"
```

## Приклад B - JSON-запит/відповідь

`WriteJSON` і `ReadJSON` маршалять і розмаршалюють за вас, тож маленький RPC по
сокету - це кілька рядків:

```go
// сервер:
var req struct{ A, B int }
_ = conn.ReadJSON(&req)
_ = conn.WriteJSON(map[string]int{"sum": req.A + req.B})

// клієнт:
_ = c.WriteJSON(map[string]int{"a": 2, "b": 3})
var reply struct{ Sum int `json:"sum"` }
_ = c.ReadJSON(&reply) // reply.Sum == 5
```

## Приклад C - сервер пушить потік

Цінність сокета в тому, що сервер може слати, коли його не просили. Тут він пише
послідовність і закриває; клієнт читає, доки з'єднання не завершиться:

```go
// сервер:
for i := 1; i <= 3; i++ {
	_ = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("tick %d", i)))
}

// клієнт:
for {
	_, m, err := c.ReadMessage()
	if err != nil {
		break
	}
	// m == "tick 1", "tick 2", "tick 3"
}
```

## Звіт виконання

Програма хостить три ендпоінти й під'єднується до них як клієнт:

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
```

Відлуння повернуло точні байти; JSON-RPC порахував суму на сервері й прочитав її
на клієнті; а потік доставив три ініційовані сервером повідомлення по одному
з'єднанню.

## Що ви дізналися

- `websocket.Upgrade(w, r)` апгрейдить на сервері; `websocket.Dial(ctx, url)`
  під'єднує з клієнта; обидва дають `*Conn`.
- `ReadMessage`/`WriteMessage` рухають сирі кадри; `ReadJSON`/`WriteJSON`
  маршалять за вас.
- Сокет дає серверу пушити без запиту - у цьому й сенс брати його замість
  polling. Загорніть набір з'єднань у hub, щоб розсилати багатьом клієнтам.
- Body-cap чи буферизувальний timeout-middleware треба пропускати для апгрейду;
  главу про весь стек показує патерн.

Далі: складаємо шматки в один сервіс.

---

[« Життєвий цикл сервісу](08-lifecycle.uk.md) · [Зміст](../main.uk.md) · [Складаємо все разом »](10-whole-stack.uk.md)
