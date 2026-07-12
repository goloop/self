[« Конфігурація](01-configuration.uk.md) · [Зміст](../main.uk.md) · [Валідація й чистка »](03-validate-and-clean.uk.md)

---

# 02. JSON HTTP API без фреймворка

**Задача.** Віддавати JSON по HTTP: маршрутизувати запити, читати параметри
шляху й параметри запиту, повертати правильні статус-коди для читань і записів,
і загорнути все у звичні наскрізні турботи (request id, відновлення після
паніки, логування, security-заголовки) плюс одну свою - не беручи веб-фреймворк.

**Модулі.** [`mux`](https://github.com/goloop/mux) маршрутизує (тонкий шар над
`net/http.ServeMux`), [`resp`](https://github.com/goloop/resp) пише JSON- і
error-відповіді, [`qp`](https://github.com/goloop/qp) читає й валідує параметри
запиту, [`middlewares`](https://github.com/goloop/middlewares) - це ланцюжок
обгорток `func(http.Handler) http.Handler`.

**Рецепт.** [`recipes/002-http-json-api`](../recipes/002-http-json-api/)

## Приклад A - маршрутизація і JSON

`net/http` у Go вже маршрутизує, і з Go 1.22 його `ServeMux` розуміє патерни
`GET /users/{id}`. `mux` додає малу ергономіку згори; `resp` пише JSON:

```go
r := mux.New()

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
```

`resp.JSON(w, v)` кодує й пише; `resp.R` - скорочення для `map[string]any`;
`resp.Error(w, status, message)` пише узгоджене тіло помилки; `mux.Param(req, "id")`
читає шаблон. Сам `Router` є `http.Handler`.

## Приклад B - правильний статус для запису

Читання - це `200`. Створення має бути `201` із `Location`; видалення - `204`
з порожнім тілом. `resp` має обидва:

```go
r.Post("/users", func(w http.ResponseWriter, req *http.Request) {
	var in user
	if err := json.NewDecoder(req.Body).Decode(&in); err != nil {
		_ = resp.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	u := s.add(in)
	_ = resp.Created(w, "/users/"+strconv.Itoa(u.ID), u) // 201 + Location
})

r.Delete("/users/{id}", func(w http.ResponseWriter, req *http.Request) {
	if !s.del(atoi(mux.Param(req, "id"))) {
		_ = resp.Error(w, http.StatusNotFound, "user not found")
		return
	}
	_ = resp.NoContent(w) // 204, порожнє тіло
})
```

## Приклад C - власний middleware у ланцюжку

Наскрізні турботи - це звичайні обгортки, застосовані від зовнішньої. Власній не
потрібен особливий інтерфейс - це просто `func(http.Handler) http.Handler`:

```go
func apiVersion(v string) middlewares.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-API-Version", v)
			next.ServeHTTP(w, r)
		})
	}
}

return middlewares.Handler(r,
	middlewares.RequestID(),
	middlewares.Recoverer(),
	middlewares.Logger(),
	middlewares.SecurityHeaders(),
	apiVersion("v1"), // ваша, у тому самому списку
)
```

## Приклад D - параметри запиту, читання й валідація

Ендпоінту-списку потрібні пагінація й фільтр, а вони приходять у query-рядку:
`GET /users?limit=2&offset=0&q=ada`. `qp` читає кожен параметр із типом,
значенням за замовчуванням і обмеженням, тож відсутнє чи хибне значення ніколи
не доходить до вашого обробника:

```go
r.Get("/users", func(w http.ResponseWriter, req *http.Request) {
	q := qp.New(req.URL)
	limit := q.Int("limit", qp.Default(20), qp.Between(1, 100))
	offset := q.Int("offset", qp.Default(0), qp.Min(0))
	name := q.String("q")
	_ = resp.JSON(w, resp.R{
		"users":  s.list(limit, offset, name),
		"limit":  limit,
		"offset": offset,
	})
})
```

`qp.Default(v)` - запасне значення, коли параметр відсутній, порожній або
некоректний. `qp.Between(1, 100)` і `qp.Min(0)` - обмеження: значення поза
діапазоном *відкидається*, а не обрізається - `qp` прибирає його й повертає
default. Тож `limit=999` не стає `100`; він стає default `20`. Ваш обробник
отримує значення, якому можна довіряти, і йому не треба перевіряти межі знову.

## Звіт виконання

Протестовано через `httptest`, потім розгорнуто й перевірено `curl`:

```text
$ go test ./...
ok  	goloop.one/handbook/002-http-json-api	0.004s

$ ./api &                        # розгорнуто на :8081

$ curl -D - -o /dev/null localhost:8081/health
HTTP/1.1 200 OK
X-Api-Version: v1
X-Content-Type-Options: nosniff
X-Request-Id: 5fdbdc2b60e7ee66b91294605c3e0a2b

$ curl -D - -X POST localhost:8081/users -d '{"name":"Grace","email":"grace@example.com"}'
HTTP/1.1 201 Created
Location: /users/2
{"id":2,"name":"Grace","email":"grace@example.com"}

$ curl -w 'HTTP %{http_code} bytes=%{size_download}\n' -X DELETE localhost:8081/users/1
HTTP 204 bytes=0

$ curl -w 'HTTP %{http_code}\n' localhost:8081/users/1   # після видалення
HTTP 404

$ curl -s 'localhost:8081/users?limit=2'
{"limit":2,"offset":0,"users":[{"id":1,...},{"id":2,...}]}

$ curl -s 'localhost:8081/users?q=gra'                   # фільтр за іменем
{"limit":20,"offset":0,"users":[{"id":2,"name":"Grace",...}]}

$ curl -s 'localhost:8081/users?limit=999'               # поза діапазоном
{"limit":20,...}                                         # відкинуто -> default 20
```

Кожна відповідь несе security-заголовок, request id і ваш `X-Api-Version`.
Створення повертає `201` із `Location: /users/2`; видалення - `204` без тіла; а
видалений користувач потім `404`. Список шанує `limit=2` і фільтр `q`, а
`limit=999` відкидає назад до default `20`, а не обрізає.

## Що ви дізналися

- `mux` маршрутизує стандартними патернами `GET /path/{id}`; роутер - це
  `http.Handler`.
- `resp.JSON`/`resp.R`/`resp.Error` покривають читання; `resp.Created` (201 +
  Location) і `resp.NoContent` (204) покривають записи.
- `qp.New(req.URL)` читає параметри запиту з типом, `Default` і обмеженнями
  (`Between`, `Min`); значення поза діапазоном відкидається й замінюється на
  default, а не обрізається.
- `middlewares.Handler(h, ...)` складає турботи; ваш власний middleware - це
  просто `func(http.Handler) http.Handler` у тому самому списку.
- Тестуйте через `httptest`; рецепт робить це й живий `curl`.

Далі: переконаймося, що дані, які приходять *усередину*, чисті, перш ніж їм
довіряти.

---

[« Конфігурація](01-configuration.uk.md) · [Зміст](../main.uk.md) · [Валідація й чистка »](03-validate-and-clean.uk.md)
