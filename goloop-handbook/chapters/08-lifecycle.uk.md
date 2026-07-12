[« Слаги й регістри](07-slug.uk.md) · [Зміст](../main.uk.md) · [Реальний час на WebSocket »](09-websocket.uk.md)

---

# 08. Життєвий цикл сервісу

**Задача.** Запустити HTTP-сервер як справжній сервіс: стартувати його, виставити
проби liveness і readiness, логувати структуровано й чисто завершитися по
сигналу - зупиняючи компоненти по порядку, не лишаючи з'єднань висіти.

**Модулі.** [`app`](https://github.com/goloop/app) - це впорядкований
start/stop-цикл, [`observe`](https://github.com/goloop/observe) дає `/healthz` і
`/readyz`, а [`log`](https://github.com/goloop/log) дає спільний структурований
`slog`-логер.

**Рецепт.** [`recipes/008-lifecycle`](../recipes/008-lifecycle/)

## Приклад A - життєвий цикл

`app` запускає набір компонентів із упорядкованим стартом, signal-обізнаним
очікуванням і граційним зворотним завершенням. `http.Server` - готовий
компонент:

```go
a := app.New("handbook-svc", app.WithLogger(logger))
a.Use(app.HTTPServer("http", &http.Server{Addr: ":8085", Handler: mux}))
a.OnStart(func(context.Context) error { logger.Info("service up"); return nil })
a.OnStop(func(context.Context) error { logger.Info("service stopping"); return nil })
return a.Run(ctx) // блокує до скасування ctx або SIGINT/SIGTERM, потім чисто зупиняє
```

`Run` повертає `nil` при чистому завершенні, тож `main`, що повертає свою
помилку, робить правильно і на сигнал, і на фатальну помилку компонента.

## Приклад B - liveness і readiness

`observe` розділяє два питання. Liveness (`/healthz`) питає «чи процес живий?» і
завжди ok. Readiness (`/readyz`) питає «чи залежності придатні?» і проганяє
кожну зареєстровану перевірку:

```go
reg := observe.New(observe.WithService("handbook-svc"),
	observe.WithBuildInfo(observe.BuildInfo{Version: "1.0.0"}))
reg.Check("clock", func(context.Context) error { return nil }) // ваша реальна перевірка

mux.Handle("GET /healthz", reg.HealthHandler())
mux.Handle("GET /readyz", reg.ReadyHandler())
```

## Приклад C - один спільний логер

`log.NewSlog` повертає стандартний `*slog.Logger`, тож кожна частина сервісу -
ваш код і сам `app` - логує через той самий структурований логер:

```go
logger := log.NewSlog("svc")
```

## Приклад D - readiness, що йде за залежністю

Справжня readiness-перевірка відображає стан залежності, а не константу.
Зареєструйте перевірку, що повертає помилку, поки залежність лежить; тоді
`/readyz` відповідає `503`, доки вона не підніметься, і `200` після - ніколи не
чіпаючи liveness. Тут `atomic.Bool` заміняє з'єднання з базою, яке перемикає
адмін-маршрут:

```go
var dbUp atomic.Bool
reg.Check("database", func(context.Context) error {
	if !dbUp.Load() {
		return errors.New("database: not ready")
	}
	return nil
})

mux.HandleFunc("POST /admin/db/{state}", func(w http.ResponseWriter, r *http.Request) {
	dbUp.Store(r.PathValue("state") == "up") // у проді - за справжнім з'єднанням
	w.WriteHeader(http.StatusNoContent)
})
```

Оркестратор, що опитує `/readyz`, тримає трафік від інстансу, доки залежність не
готова, а тоді спрямовує його, щойно перевірка проходить.

## Приклад E - запит-скоупний логер

`log.NewSlog` повертає стандартний `*slog.Logger`, тож `With` дає дочірній
логер, що приколює поля до кожного рядка, який він пише. Використовуйте його, щоб
причепити контекст обробника один раз замість повторювати на кожному виклику:

```go
reqLog := logger.With("route", "admin.db", "state", r.PathValue("state"))
reqLog.Info("dependency state changed") // ...route=admin.db state=up
```

## Звіт виконання

Протестовано (зокрема справжній старт-і-завершення), потім розгорнуто,
опробувано пробами й надіслано сигнал (шляхи в логах обрізано):

```text
$ go test ./...
ok  	goloop.one/handbook/008-lifecycle	0.307s

$ ./svc &                        # віддає на :8085

$ curl localhost:8085/healthz
{"status":"ok","service":"handbook-svc","version":"1.0.0"}

$ curl -o /dev/null -w 'HTTP %{http_code}\n' localhost:8085/readyz   # база ще лежить
HTTP 503
{"status":"error", ... "checks":{"clock":{"status":"ok"},
                                 "database":{"status":"error"}}}

$ curl -X POST localhost:8085/admin/db/up      # залежність піднімається
INFO ... dependency state changed route=admin.db state=up

$ curl -o /dev/null -w 'HTTP %{http_code}\n' localhost:8085/readyz   # тепер готово
HTTP 200
{"status":"ok", ... "checks":{"clock":{"status":"ok"},
                              "database":{"status":"ok"}}}

$ kill -INT %1                   # граційне завершення
INFO service up addr=:8085
INFO starting component component=http
INFO running components=1
INFO shutdown signal received
INFO stopping component component=http
INFO service stopping
```

Liveness відповів одразу; readiness прогнав обидві перевірки й повідомив `503`,
поки перевірка `database` падала, тоді `200`, щойно залежність піднялась;
запит-скоупний логер проштампував `route` і `state` у своєму рядку; а сигнал
пройшов життєвим циклом назад - зупинити компонент, виконати stop-хук - замість
того, щоб просто впустити процес.

## Що ви дізналися

- `app` запускає компоненти з упорядкованим стартом і граційним зворотним
  завершенням; `app.HTTPServer` обгортає `http.Server`, а `Run(ctx)` блокує до
  сигналу чи скасування.
- `observe` розділяє liveness (`/healthz`, завжди ok) і readiness (`/readyz`,
  проганяє ваші `Check`), як і очікують оркестратори.
- Readiness-`Check`, прив'язаний до залежності, змушує `/readyz` повертати `503`,
  доки залежність не піднялась, тож трафік чекає на справжній стан, не константу.
- `log.NewSlog` дає один структурований `*slog.Logger` на весь сервіс, зокрема
  й для власних lifecycle-логів `app`; `logger.With(...)` приколює запит-скоупні
  поля до дочірнього логера.

Далі: тримати з'єднання відкритим для оновлень у реальному часі.

---

[« Слаги й регістри](07-slug.uk.md) · [Зміст](../main.uk.md) · [Реальний час на WebSocket »](09-websocket.uk.md)
