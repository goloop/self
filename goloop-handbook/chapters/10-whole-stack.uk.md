[« Реальний час на WebSocket](09-websocket.uk.md) · [Зміст](../main.uk.md) · [Зміст »](../main.uk.md)

---

# 10. Складаємо все разом - один API на цілому стеку

**Задача.** Збудувати малий, але повний сервіс: API нотаток, де людина
реєструється, входить, створює й перелічує нотатки за bearer-токеном. Ця глава
не вводить нового модуля; вона збирає ті, що з попередніх глав, в одну програму,
щоб ви побачили, як вони компонуються.

**Модулі.** Усе дотепер: `env`/`opt` (конфіг), `app`/`observe`/`log` (цикл,
проби, логування), `mux`/`resp`/`middlewares` (HTTP), `norm` (валідація), `auth`
(паролі й токени) і `pgc`-згенероване сховище (база).

**Рецепт.** [`recipes/010-whole-stack`](../recipes/010-whole-stack/)

## Як це складається докупи

Прочитайте `main.go` згори донизу - і кожна глава з'являється на своєму місці:

- **Конфіг** з оточення й прапорців заповнює структуру `Config`
  (`env.LoadSafe`, `env.Unmarshal`, `opt.UnmarshalArgs`).
- **Життєвий цикл**: `app.New` запускає компонент `HTTPServer` і сам `api` як
  компонент, тож пул бази відкривається у `Start` і закривається у `Stop`.
- **Спостережуваність**: `observe` віддає `/healthz` і `/readyz`, а
  readiness-перевірка пінгує базу.
- **HTTP**: `mux` маршрутизує, `resp` пише JSON і помилки, `middlewares` додає
  request id, відновлення, логування й security-заголовки.
- **Валідація**: `norm.EmailFold` згортає email в ідентичність; `norm.Clean`
  чистить заголовок нотатки.
- **Auth**: `auth` хешує пароль і видає токен; `tm.Protect` захищає маршрути
  нотаток, тож неавтентифікований запит - це `401`.
- **База**: `pgc`-згенероване `store` дає типізовані `CreateUser`,
  `UserByEmail`, `CreateNote` і `NotesByUser`.

Захищені маршрути - це суть: одна обгортка робить будь-який хендлер
автентифікованим:

```go
r.Post("/v1/signup", a.signup)
r.Post("/v1/login", a.login)
r.Handle("POST /v1/notes", a.tm.Protect(http.HandlerFunc(a.createNote)))
r.Handle("GET /v1/notes", a.tm.Protect(http.HandlerFunc(a.listNotes)))
```

## Звіт виконання

Змігровано, протестовано проти справжнього PostgreSQL, потім розгорнуто й
прогнано наскрізь:

```text
$ pgc migrate
applied 001_schema.sql

$ go test ./...            # signup, захищене створення, 401 без токена
ok  	goloop.one/handbook/010-whole-stack	0.118s

$ ./notes &               # віддає на :8087

$ curl localhost:8087/readyz
{"status":"ok","service":"notes","checks":{"database":{"status":"ok","latency_ms":0}}}

$ curl -X POST localhost:8087/v1/signup -d '{"Email":"Ada@Example.com","Password":"password1"}'
201 -> {"token":"eyJhbGci...","email":"ada@example.com"}

$ curl -X POST localhost:8087/v1/notes -d '{"Title":"x"}'          # без токена
HTTP 401

$ curl -X POST localhost:8087/v1/notes -H "Authorization: Bearer $TOKEN" -d '{"Title":"Read the handbook"}'
{"id":1,"user_id":1,"title":"Read the handbook","created_at":"..."}

$ curl localhost:8087/v1/login -d '{"Email":"ada@example.com","Password":"password1"}'   # регістронезалежно
$ curl localhost:8087/v1/notes -H "Authorization: Bearer $TOKEN"
   notes: ["Read the handbook"]
```

Signup згорнув `Ada@Example.com` в ідентичність у нижньому регістрі й повернув
токен; маршрут нотатки відхилив запит без токена й прийняв із ним; login знайшов
користувача регістронезалежно; а readiness підтвердив базу. Кожен ярус із
попередніх глав робить свою одну роботу.

## Що ви дізналися

- Модулі компонуються без клею: конфіг заповнює структуру, `app` запускає сервер
  і пул, `observe` звітує про здоров'я, `mux`/`resp`/`middlewares` віддають HTTP,
  `auth` захищає маршрути, а `pgc`-сховище - це база.
- `Protect` у `auth` робить будь-який хендлер автентифікованим; id суб'єкта з
  токена - це користувач.
- Оскільки кожен шматок - окремий модуль, цей сервіс - набір рішень, а не
  фреймворк, який довелося прийняти цілком. Міняйте драйвер бази, транспорт пошти
  чи провайдера моделі, не чіпаючи решти.

Це і є весь цикл: від конфігурації всередину - до запиту, який змаршрутизовано,
провалідовано, автентифіковано, збережено й на який відповіли. Далі будуйте своє.

---

[« Реальний час на WebSocket](09-websocket.uk.md) · [Зміст](../main.uk.md) · [Зміст »](../main.uk.md)
