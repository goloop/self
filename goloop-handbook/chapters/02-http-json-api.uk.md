[« Конфігурація](01-configuration.uk.md) · [Зміст](../main.uk.md) · [Валідація й чистка »](03-validate-and-clean.uk.md)

---

# 02. JSON HTTP API без фреймворка

**Задача.** Віддавати JSON по HTTP: маршрутизувати запити, читати параметри
шляху, повертати правильні статус-коди й загорнути все у звичні наскрізні
турботи (request id, відновлення після паніки, логування, security-заголовки) -
не беручи веб-фреймворк.

**Модулі.** [`mux`](https://github.com/goloop/mux) маршрутизує (тонкий шар над
`net/http.ServeMux`), [`resp`](https://github.com/goloop/resp) пише JSON- і
error-відповіді, [`middlewares`](https://github.com/goloop/middlewares) - це
ланцюжок обгорток `func(http.Handler) http.Handler`.

**Рецепт.** [`recipes/002-http-json-api`](../recipes/002-http-json-api/)

## Ідея

`net/http` у Go вже маршрутизує, і з Go 1.22 його `ServeMux` розуміє патерни з
методом і шаблоном на кшталт `GET /users/{id}`. Чого він не дає - це малої
ергономіки згори: однорядковика для запису JSON, чистого тіла помилки й
охайного способу складати middleware. GoLoop додає саме це - і нічого, що
ховало б `net/http` під собою.

Маршрутизація через `mux` читається як стандартні патерни, бо це *і є*
стандартні патерни:

```go
r := mux.New()

r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
	_ = resp.JSON(w, resp.R{"status": "ok"})
})

r.Get("/users/{id}", func(w http.ResponseWriter, req *http.Request) {
	u, ok := users[atoi(mux.Param(req, "id"))]
	if !ok {
		_ = resp.Error(w, http.StatusNotFound, "user not found")
		return
	}
	_ = resp.JSON(w, u)
})
```

`resp.JSON(w, v)` кодує й пише; `resp.R` - скорочення для `map[string]any` під
довільні об'єкти; `resp.Error(w, status, message)` пише узгоджене тіло помилки.
`mux.Param(req, "id")` читає шаблон. Сам `Router` є `http.Handler`.

## Ланцюжок middleware

Наскрізні турботи - це звичайні обгортки, застосовані від зовнішньої:

```go
return middlewares.Handler(r,
	middlewares.RequestID(),       // позначити кожен запит id
	middlewares.Recoverer(),       // перетворити паніку на 500, не на крах
	middlewares.Logger(),          // один структурований рядок логу на запит
	middlewares.SecurityHeaders(), // nosniff, frame options тощо
)
```

Оскільки кожен middleware - це просто `func(http.Handler) http.Handler`, твої
власні лягають у той самий список, і тут немає GoLoop-специфічної сантехніки.

## Звіт виконання

Протестовано через `httptest` (без мережі), потім розгорнуто й перевірено
`curl`:

```text
$ go test ./...
INFO http request method=GET path=/health   status=200 bytes=16 ...
INFO http request method=GET path=/users/1   status=200 bytes=48 ...
INFO http request method=GET path=/users/999 status=404 bytes=40 ...
ok  	goloop.one/handbook/002-http-json-api	0.003s

$ ./api &                        # розгорнуто на :8081

$ curl -i localhost:8081/health
HTTP/1.1 200 OK
X-Content-Type-Options: nosniff
X-Request-Id: 53941dbe3ef91b6ca0c52360342fc109
{"status":"ok"}

$ curl localhost:8081/users/1
{"id":1,"name":"Ada","email":"ada@example.com"}

$ curl -w 'HTTP %{http_code}\n' localhost:8081/users/999
{"code":404,"message":"user not found"}
HTTP 404
```

Кожна відповідь несе security-заголовок і request id, параметр шляху
розв'язується, а відсутній користувач повертає чисте `404` JSON-тіло - тести
перевіряють усі три, а живий `curl` це підтверджує.

## Що ти дізнався

- `mux` маршрутизує стандартними патернами `GET /path/{id}`; роутер - це
  `http.Handler`.
- `resp.JSON`, `resp.R` і `resp.Error` покривають JSON- і error-відповіді, які
  пишеш раз за разом.
- `middlewares.Handler(h, ...)` складає турботи як звичайні `net/http`-обгортки;
  твій власний middleware лягає в той самий список.
- Тестуй хендлери через `httptest`; рецепт робить і це, і живий `curl`.

Далі: переконаймося, що дані, які приходять *усередину*, чисті, перш ніж їм
довіряти.

---

[« Конфігурація](01-configuration.uk.md) · [Зміст](../main.uk.md) · [Валідація й чистка »](03-validate-and-clean.uk.md)
