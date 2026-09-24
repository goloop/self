[« Життєвий цикл сервісу](08-lifecycle.uk.md) · [Зміст](../main.uk.md) · [Складаємо все разом »](10-whole-stack.uk.md)

---

# 09. Реальний час на WebSocket

**Задача.** Тримати з'єднання відкритим і слати повідомлення обома напрямками:
відлунити повідомлення, зробити маленький запит/відповідь, пушити потік оновлень
із сервера до клієнта, розіслати одне повідомлення всім під'єднаним клієнтам і
охайно закрити з'єднання зі статус-кодом.

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

## Приклад D - broadcast у hub

Причина брати сокет замість polling - це фан-аут: одна подія, доставлена кожному
під'єднаному клієнту. Hub - це захищений набір з'єднань плюс broadcast-запис.
Обробник кожного клієнта реєструє його, тоді читає в циклі; повідомлення від
будь-якого клієнта пишеться всім:

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
			delete(h.conns, c) // прибрати мертве з'єднання
		}
	}
}

// в обробнику /hub: h.add(conn); тоді на кожне повідомлення h.broadcast(data)
```

Два клієнти приєднуються, один шле `"hello all"`, і обидва отримують його.
Додайте кімнати (hub на ключ) - і маєте чат чи живу стрічку.

## Приклад E - граційне закриття зі статус-кодом

Чисте завершення - це не просто впустити сокет. `CloseWithStatus` шле close-кадр
із кодом і причиною; наступне читання клієнта повідомляє цей код, тож він
відрізнить нормальне закриття від краху. `IsCloseError` його перевіряє:

```go
// сервер: надіслати останнє повідомлення, тоді закрити чисто
_ = conn.WriteMessage(websocket.TextMessage, []byte("closing now"))
_ = conn.CloseWithStatus(websocket.CloseNormalClosure, "bye")

// клієнт: прочитати повідомлення, тоді побачити close-код на наступному читанні
_, msg, _ := c.ReadMessage() // "closing now"
_, _, err := c.ReadMessage()
websocket.IsCloseError(err, websocket.CloseNormalClosure) // true
```

## Приклад F - раптовий обрив

Друга половина «нормальне закриття чи крах» - це власне крах. Пір, який просто
зник (обірвана мережа, вбитий процес), close-кадру не надсилає взагалі, і це
завершення повідомляється так само: `*CloseError` із кодом `1006`, а початкова
помилка лишається його причиною. Тож одна перевірка покриває обидва випадки, і
помилка, яку ви ловили б руками, нікуди не поділася:

```go
_, _, err := c.ReadMessage()
if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure,
    websocket.CloseGoingAway) {
    // пір упав або з'єднання обірвалося
    log.Printf("втрачено клієнта: %v", err) // "websocket: close 1006"
}
errors.Is(err, io.EOF) // true, коли потік просто скінчився
```

Те, що насправді сталося, лишається причиною, тож його й далі можна перевірити:
`io.EOF`, коли потік скінчився, або мережева помилка на кшталт
`syscall.ECONNRESET`, коли з'єднання скинули. Дедлайн читання - інша річ, він
лишається timeout: з'єднання відкрите, а дедлайн ставили ви.

Код `1006` описує те, що сталося локально, і на дріт не йде ніколи - тож ви
побачите його з читання, але ніколи від піра.

## Звіт виконання

Програма хостить ендпоінти й під'єднується до них як клієнт:

```text
$ go test ./...
ok  	goloop.one/handbook/009-websocket	0.006s

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
F. an abrupt drop, with no close frame:
   unexpected-close=true code=1006 cause=true
```

Відлуння повернуло точні байти; JSON-RPC порахував суму на сервері й прочитав її
на клієнті; потік доставив три ініційовані сервером повідомлення по одному
з'єднанню; hub розіслав повідомлення одного клієнта обом з'єднанням; а граційне
закриття дало клієнту прочитати код нормального закриття замість голої помилки.
Рецепт проходить під детектором гонок.

## Що ви дізналися

- `websocket.Upgrade(w, r)` апгрейдить на сервері; `websocket.Dial(ctx, url)`
  під'єднує з клієнта; обидва дають `*Conn`.
- `ReadMessage`/`WriteMessage` рухають сирі кадри; `ReadJSON`/`WriteJSON`
  маршалять за вас.
- Сокет дає серверу пушити без запиту - у цьому й сенс брати його замість
  polling. Hub (захищений набір з'єднань плюс broadcast) розсилає одне
  повідомлення кожному клієнту; додайте кімнати для чату чи живої стрічки.
- `CloseWithStatus` закриває з кодом і причиною; `IsCloseError` дає іншому кінцю
  відрізнити нормальне закриття від краху, а `IsUnexpectedCloseError` ловить сам
  крах - обірване з'єднання приходить як close `1006`.
- `SetReadLimit` обмежує отримане повідомлення, `SetWriteLimit` - надіслане; а
  повідомлення, записане через `NextWriter`, виходить фрагментами в міру запису,
  тож надіслати велике не означає тримати його в пам'яті.
- Body-cap чи буферизувальний timeout-middleware треба пропускати для апгрейду;
  глава про весь стек показує патерн.

Далі: складаємо шматки в один сервіс.

---

[« Життєвий цикл сервісу](08-lifecycle.uk.md) · [Зміст](../main.uk.md) · [Складаємо все разом »](10-whole-stack.uk.md)
