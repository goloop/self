[![Stay with Ukraine](https://img.shields.io/static/v1?label=Stay%20with&message=Ukraine%20♥&color=ffD700&labelColor=0057B8&style=flat)](https://u24.gov.ua/)

# goloop

📖 [English](README.md) · **Українська**

`goloop` - це група невеликих, сфокусованих Go-модулів для щоденної роботи з
конфігурацією, CLI, HTTP, роутингом і middleware, WebSocket-з'єднаннями, API
великих мовних моделей, типізованими PostgreSQL-запитами, валідацією,
логуванням, колекціями, ідентифікаторами, рядками, рефлексією типів та
тризначною логікою - плюс тонкий прикладний шар (життєвий цикл сервісу,
health-ендпоінти, автентифікація, сесії й пошта), що будується поверх цих
шматків. Модулі незалежні: ви підключаєте тільки той пакет, який потрібен
конкретному застосунку, а кожен пакет має власний versioned module path.

Група має два яруси. **Фундаментний ярус** незалежних, здебільшого
zero-dependency утиліт, і невеликий **прикладний ярус**, що збирає їх у
повторювані частини мережевого сервісу. Уперше з goloop? Спершу прочитайте
[Побудова сервісу](#побудова-сервісу) для загальної картини, а потім
занурюйтеся в довідник по кожному модулю нижче.

Поточна група:

- **Фундамент:** `ai` (з драйверами провайдерів `anthropic`, `openai`,
  `gemini`, `grok`, `deepseek`, `openrouter`, `ollama`, `mistral`, `cohere`),
  `env`, `g`, `is`, `key`, `kind`, `log`, `middlewares`, `mux`, `opt`, `pgc`,
  `qp`, `resp`, `scs`, `set`, `slug`, `t13n`, `trit`, `websocket`.
- **Прикладний ярус:** `app`, `observe`, `jwt`, `auth`, `session`, `mail`, `cli`.

Разом вони закривають “нудні”, але важливі краї прикладного коду: читання
конфігурації з `.env`, парсинг аргументів командного рядка, перевірку вхідних
даних, читання query-параметрів, роутинг запитів, побудову ланцюжків middleware,
запис HTTP-відповідей, роботу з протоколом WebSocket, логування, перетворення
стилів рядків, генерацію slug-ів, транслітерацію Unicode, множини, короткі
зворотні ключі, generic-хелпери і логіку з третім станом `Unknown` - а тоді
запуск сервісу з упорядкованим життєвим циклом start/stop, health- і
readiness-пробами, підписаними токенами й cookie-сесіями та вихідною поштою.

## Що означає goloop

GoLoop - це stdlib-first набір Go-інструментів. Він не намагається замінити
стандартну бібліотеку Go або сховати її за фреймворком. Кожен модуль лишає
stdlib базовою мовою проєкту й додає тільки той малий шар, який прикладний код
інакше переписував би в кожному застосунку.

Практичні правила дизайну:

- **Малі модулі, чітка відповідальність.** Один пакет має добре закривати одну
  звичайну задачу: environment config, CLI options, HTTP responses, routing,
  validation, logging, WebSocket, згенеровані PostgreSQL-запити тощо.
- **Явна поведінка.** Public API має показувати control flow, помилки й рух
  даних. Згенерований код має виглядати як код, який уважний Go-розробник міг
  би написати руками.
- **Stdlib first.** Спершу `net/http`, `database/sql`, `log/slog`, `context`,
  `encoding/json`, `iter`, `slices`, `maps`, `cmp` та інші стандартні пакети;
  власні runtime-абстракції - тільки коли вони справді потрібні.
- **Нуль залежностей за замовчуванням.** Модуль може залежати від іншого
  GoLoop-модуля, якщо це частина дизайну, але сторонні залежності мають бути
  винятком і чесно окупати свою ціну.
- **Без framework lock-in.** Можна взяти один модуль без прийняття всієї
  екосистеми. Згенерований код не має вимагати GoLoop runtime-пакет, якщо саме
  ця залежність не є суттю фічі.
- **Вимірювана якість.** Гарячі шляхи мають бути allocation-aware,
  race-clean і покриті тестами. Benchmark-и - інженерний інструмент, а не
  маркетинг.
- **Нудна поверхня, сильні нутрощі.** Найкращий GoLoop-пакет зовні виглядає
  очевидно, навіть якщо всередині є протокол, парсер, кешування або
  кодогенерація.

Коротко: GoLoop - для застосунків, яким потрібні явний Go-код, інтерфейси
стандартної бібліотеки, малі сфокусовані пакети й production-grade краї без
прийняття великого фреймворку.

Саме тому `pgc` генерує звичайний `database/sql` код замість runtime ORM,
`websocket` реалізує RFC 6455 без фреймворку, `mux` будується поверх
`net/http.ServeMux`, а provider-пакети на кшталт `anthropic` чи `openai`
показують нативні API й ділять тільки малий контракт `ai`, де це справді
корисно.

## Побудова сервісу

Це частина, яку варто прочитати першою. Модулі незалежні, але спроєктовані так,
щоб складатися разом у передбачуваному порядку. Типовий goloop-сервіс збирається
шарами, кожен закриває конкретну прогалину, яку лишає стандартна бібліотека:

| Етап | Модулі | Що роблять |
|------|--------|-----------|
| **Конфігурація** | `env`, `opt` | Читають `.env`/environment у типізовану структуру config; парсять CLI-прапорці. |
| **Життєвий цикл** | `app` | Володіє впорядкованою послідовністю start/stop і graceful shutdown, щоб `main` цього не робив. |
| **HTTP-край** | `mux`, `middlewares` | Роутять запити поверх `net/http.ServeMux`; додають request ID, real IP, recovery, логування, CORS. |
| **Handlers** | `qp`, `resp`, `is` | Читають типізовані query-параметри, валідують вхід, пишуть JSON/інші відповіді. |
| **Ідентичність** | `auth`, `jwt`, `session` | Хешують паролі, видають і перевіряють токени, захищають маршрути, тримають браузерні сесії. |
| **Дані** | `pgc` | Компілюють SQL у типобезпечний Go проти PostgreSQL. |
| **Realtime / AI** | `websocket`, `ai` | Говорять RFC 6455; звертаються до LLM-провайдерів за одним інтерфейсом. |
| **Спостережуваність** | `observe`, `log` | Виставляють `/healthz` і `/readyz`; продукують operational-логи. |
| **Скафолдинг** | `cli` | Командний інструмент, що генерує й перевіряє goloop-first проєкти. |

Напрям залежностей односторонній: прикладний ярус імпортує фундамент, ніколи
навпаки. `app` нічого не знає про HTTP чи health; `mux` нічого не знає про
життєвий цикл; `observe` ніколи не імпортує `app`. Ти зшиваєш їх у власному коді
обв'язки, і це тримає кожен модуль незалежно придатним до використання й
незалежно тестованим.

Мінімальний, але повний сервіс зшиває конфігурацію, життєвий цикл, роутинг,
middleware, health і відповіді разом:

```go
package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/goloop/app"
	"github.com/goloop/env/v2"
	"github.com/goloop/middlewares"
	"github.com/goloop/mux"
	"github.com/goloop/observe"
	"github.com/goloop/resp/v2"
)

type Config struct {
	Addr string `env:"ADDR" def:":8080"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// 1. Конфігурація з environment.
	var cfg Config
	if err := env.Unmarshal(&cfg); err != nil {
		return err
	}

	// 2. Health- і readiness-проби (observe не імпортує життєвий цикл).
	obs := observe.New(observe.WithService("api"))
	obs.Check("self", func(context.Context) error { return nil })

	// 3. Маршрути поверх net/http.ServeMux, тоді ланцюжок middleware.
	r := mux.New()
	r.Get("/healthz", obs.HealthHandler().ServeHTTP)
	r.Get("/readyz", obs.ReadyHandler().ServeHTTP)
	r.Get("/hello", func(w http.ResponseWriter, _ *http.Request) {
		_ = resp.JSON(w, resp.R{"message": "hello"})
	})
	handler := middlewares.Chain(
		middlewares.RequestID(),
		middlewares.RealIP(),
		middlewares.Recoverer(),
		middlewares.Logger(),
	)(r)

	// 4. Життєвий цикл: упорядкований старт, очікування сигналу, graceful shutdown.
	a := app.New("api",
		app.WithLogger(slog.New(slog.NewTextHandler(os.Stdout, nil))),
		app.WithShutdownTimeout(10*time.Second),
	)
	a.Use(app.HTTPServer("http", &http.Server{Addr: cfg.Addr, Handler: handler}))
	return a.Run(context.Background())
}
```

Додай ідентичність, видаючи токени через `auth` (який підписує HS256 JWT через
`jwt`) і захищаючи маршрути його bearer-middleware, або тримай браузерний стан у
підписаній cookie через `session`; надсилай транзакційну пошту через `mail`;
дістанься бази через `pgc`. Кожне - це окремий import, який ти додаєш лише за
потреби; ніщо вище не вимагає решти групи.

## Зміст

Перейдіть до потрібного пакета; кожен блок завершується посиланнями на
репозиторій і довідник. Як вони поєднуються - див.
[Побудова сервісу](#побудова-сервісу).

**Фундамент**

- [**ai** - один інтерфейс до LLM-API, з драйверами для основних провайдерів](#ai)
- [**env** - .env файли, process environment і struct mapping](#env)
- [**g** - generic-хелпери для слайсів, чисел, умов і конвертацій](#g)
- [**is** - перевірка форматів і значень](#is)
- [**key** - зворотні короткі ключі для uint64 ID](#key)
- [**kind** - кешована рефлексія для авторів парсерів і декодерів](#kind)
- [**log** - рівневе логування у кілька напрямків](#log)
- [**middlewares** - net/http middleware: request ID, real IP, recovery, логування тощо](#middlewares)
- [**mux** - ергономічний роутинг поверх net/http.ServeMux](#mux)
- [**norm** - нормалізація введених даних до канонічної форми з валідацією](#norm)
- [**opt** - парсинг CLI-аргументів у структури](#opt)
- [**pgc** - SQL-запити, скомпільовані в типобезпечний Go для PostgreSQL](#pgc)
- [**qp** - типізоване читання URL query-параметрів](#qp)
- [**resp** - HTTP response helpers поверх net/http](#resp)
- [**scs** - перетворення і визначення стилів рядків](#scs)
- [**set** - generic-множини для comparable типів](#set)
- [**slug** - URL-friendly slug-и з Unicode-тексту](#slug)
- [**t13n** - Unicode-to-ASCII транслітерація](#t13n)
- [**trit** - тризначна логіка: False, Unknown, True](#trit)
- [**websocket** - WebSocket (RFC 6455): клієнт і сервер](#websocket)

**Прикладний ярус**

- [**app** - життєвий цикл сервісу: упорядкований start/stop і graceful shutdown](#app)
- [**observe** - health-, readiness- і build-info-ендпоінти](#observe)
- [**jwt** - компактні HS256 JSON Web Tokens](#jwt)
- [**auth** - хешування паролів, access-токени і bearer-middleware](#auth)
- [**session** - підписані cookie-сесії для браузерних застосунків](#session)
- [**mail** - побудова й надсилання вихідної пошти через SMTP](#mail)
- [**cli** - скафолдинг і перевірка goloop-first проєктів](#cli)

## ai

`ai` - це один провайдер-незалежний інтерфейс до API великих мовних моделей плюс
спільні типи запиту й відповіді, якими «говорить» кожен драйвер. Як `database/sql`
зі стандартної бібліотеки з її драйверами, `ai` тримає спільний контракт -
`Generate` і стрімінговий `Stream`, повідомлення, інструменти й мультимодальні
частини - а окремий пакет на кожного провайдера його реалізує. Код проти
інтерфейсу працює з будь-яким провайдером; ендпоінти, яких провайдер не поділяє,
подаються власними нативними методами драйвера. Кожен драйвер залежить лише від
`ai`, тож увесь набір без сторонніх залежностей.

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/goloop/ai"
	"github.com/goloop/anthropic"
)

func main() {
	var client ai.Client = anthropic.New(os.Getenv("ANTHROPIC_API_KEY"))

	resp, err := client.Generate(context.Background(), &ai.Request{
		Model:    anthropic.ModelClaudeSonnet5,
		Messages: []ai.Message{ai.UserText("Привітайся одним словом.")},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(resp.Text())
}
```

Драйвери провайдерів, кожен реалізує `ai.Client` й імпортує лише `ai`:

- **anthropic** - Claude Messages API, батчі й підрахунок токенів - [repo](https://github.com/goloop/anthropic) · [довідник](https://pkg.go.dev/github.com/goloop/anthropic)
- **openai** - Chat Completions, Responses API, embeddings, зображення й аудіо - [repo](https://github.com/goloop/openai) · [довідник](https://pkg.go.dev/github.com/goloop/openai)
- **gemini** - Google Gemini `generateContent`, embeddings і підрахунок токенів - [repo](https://github.com/goloop/gemini) · [довідник](https://pkg.go.dev/github.com/goloop/gemini)
- **grok** - xAI Grok, сумісний із chat-completions, із генерацією зображень - [repo](https://github.com/goloop/grok) · [довідник](https://pkg.go.dev/github.com/goloop/grok)
- **deepseek** - DeepSeek chat і ланцюг міркувань reasoning-моделі - [repo](https://github.com/goloop/deepseek) · [довідник](https://pkg.go.dev/github.com/goloop/deepseek)
- **openrouter** - шлюз OpenRouter до багатьох моделей за одним ключем - [repo](https://github.com/goloop/openrouter) · [довідник](https://pkg.go.dev/github.com/goloop/openrouter)
- **ollama** - локальні моделі через нативний Ollama API - [repo](https://github.com/goloop/ollama) · [довідник](https://pkg.go.dev/github.com/goloop/ollama)
- **mistral** - Mistral chat, embeddings і fill-in-the-middle - [repo](https://github.com/goloop/mistral) · [довідник](https://pkg.go.dev/github.com/goloop/mistral)
- **cohere** - Cohere v2 chat, embeddings і rerank - [repo](https://github.com/goloop/cohere) · [довідник](https://pkg.go.dev/github.com/goloop/cohere)

**Детальніше:** [github.com/goloop/ai](https://github.com/goloop/ai) · [довідник](https://pkg.go.dev/github.com/goloop/ai)

## env

`env` з'єднує `.env` файли, process environment і Go-структури. Його варто
використовувати тоді, коли конфігурація живе в environment variables, але
застосунок хоче працювати з типізованим config-об'єктом: зі значеннями за
замовчуванням, обов'язковими полями, слайсами, масивами, вкладеними
структурами, `time.Duration`, `time.Time`, `url.URL` та іншими звичними Go
типами.

Пакет може завантажувати файли в process environment (`Load`, `Overload`),
парсити `.env` у map без побічних ефектів (`Read`, `Parse`), розкладати
environment у структуру і серіалізувати структуру назад у `.env` текст або
файл. Тому він корисний і в застосунках, і в тестах, де небажано чіпати
глобальне environment-середовище.

```go
package main

import (
	"log"
	"time"

	"github.com/goloop/env/v2"
)

type Config struct {
	Host    string        `env:"HOST" def:"127.0.0.1"`
	Port    int           `env:"PORT" def:"8080"`
	Timeout time.Duration `env:"TIMEOUT" def:"5s"`
}

func main() {
	var cfg Config
	if err := env.Unmarshal(&cfg); err != nil {
		log.Fatal(err)
	}
}
```

**Детальніше:** [github.com/goloop/env](https://github.com/goloop/env) · [довідник](https://pkg.go.dev/github.com/goloop/env/v2)

## g

`g` - це generic toolbox. Він збирає короткі функції, які в Go-проєктах часто
переписують локально: умовне значення, ліниву умовну гілку, min/max, clamp,
map/filter для слайсів, перевірку входження, конвертації, безпечну арифметику
та поширені числові утиліти.

Найкраще сприймати `g` як зручний шар поверх стандартних Go-семантик. У v2
гарячі місця делегуються сучасній стандартній бібліотеці (`slices`, `maps`,
`cmp`, `iter`, `math/rand/v2`) там, де це правильно, а для коду застосунку
залишається короткий фасад `g.*`.

```go
package main

import (
	"fmt"

	g "github.com/goloop/g/v2"
)

func main() {
	name := g.If(len("admin") > 0, "admin", "guest")
	page := g.Clamp(250, 1, 100)
	ids := g.Filter([]int{1, 2, 3, 4}, func(v int) bool { return v%2 == 0 })

	fmt.Println(name, page, ids) // admin 100 [2 4]
}
```

**Детальніше:** [github.com/goloop/g](https://github.com/goloop/g) · [довідник](https://pkg.go.dev/github.com/goloop/g/v2)

## is

`is` - пакет валідації. Кожна функція відповідає на одне питання про значення:
чи це email, IP-адреса, IBAN, UUID, телефон, hex color, JWT, numeric string,
latitude, variable name тощо.

Пакет перевіряє саме той input, який ви передали; він не є sanitizer-ом і не
нормалізує дані. Це важливо для HTTP forms і API: якщо застосунку потрібна
нормалізація, зробіть її окремо (див. [norm](#norm)), а потім викликайте `is.*`,
щоб перевірити очікуваний формат або правило.

```go
package main

import (
	"fmt"

	"github.com/goloop/is/v2"
)

func main() {
	fmt.Println(is.Email("user@example.com"))                    // true
	fmt.Println(is.UUID("550e8400-e29b-41d4-a716-446655440000")) // true
	fmt.Println(is.IP("2001:db8::1"))                            // true
}
```

**Детальніше:** [github.com/goloop/is](https://github.com/goloop/is) · [довідник](https://pkg.go.dev/github.com/goloop/is/v2)

## key

`key` перетворює `uint64` ідентифікатори у короткі зворотні текстові ключі на
основі вашого алфавіту. Це корисно для публічних ID, invite-кодів, номерів
тікетів, купонів і URL-safe представлення внутрішніх числових ID.

Головна абстракція - `Locksmith`: base-N encoder/decoder поверх заданого
алфавіту. Dynamic keys мають мінімально потрібну довжину, fixed keys завжди
мають рівно заданий розмір. Decode строгий: кожен валідний key має одну
canonical форму і один numeric ID.

```go
package main

import (
	"fmt"

	"github.com/goloop/key/v2"
)

func main() {
	ls := key.MustNewFixed(key.Base62, 8)

	s, _ := ls.Marshal(12345)
	id, _ := ls.Unmarshal(s)

	fmt.Println(s, id) // 000003D7 12345
}
```

**Детальніше:** [github.com/goloop/key](https://github.com/goloop/key) · [довідник](https://pkg.go.dev/github.com/goloop/key/v2)

## kind

`kind` відповідає на одне питання - «що це за тип і що він уміє?» - для коду,
який мусить обробляти довільні Go-типи під час виконання. Його рідна територія -
парсери, декодери й біндери: змінні оточення, файли конфігурації, CLI-прапорці,
query-параметри чи рядки БД, які вкладаються в структури, визначені
*викликачем*. Замість ручного `reflect` («це int? вказівник на структуру? чи
реалізує воно `TextUnmarshaler` на pointer-receiver?») ти питаєш один кешований
дескриптор пласким словником предикатів. Поза парсингом він покриває виявлення
здатностей (чи реалізує тип - або вказівник на нього - `sql.Scanner`,
`flag.Value`, метод `Set(string) error`?) та обхід полів і тегів структур для
валідаторів і генераторів; дескриптори кешуються за типом, тож гарячі цикли
парсингу платять за класифікацію один раз. Він навмисно вузький: якщо типи
відомі на етапі компіляції, якщо вистачить однієї дрібної перевірки через
`reflect`, або якщо треба *записувати* значення, а не класифікувати їх - `kind`
тобі не потрібен.

```go
package main

import (
	"fmt"

	"github.com/goloop/kind"
)

func main() {
	k := kind.Of([]int{1, 2, 3})

	fmt.Println(k.IsSlice())      // true
	fmt.Println(k.Elem().IsInt()) // true - тип елемента
	fmt.Println(k.IsAnyInt())     // true - leaf-aware: листок слайса - int
}
```

**Детальніше:** [github.com/goloop/kind](https://github.com/goloop/kind) · [довідник](https://pkg.go.dev/github.com/goloop/kind)

## log

`log` - рівневий logger із кількома outputs. Він може писати різні рівні в
різні writers, рендерити text або JSON, додавати timestamp і caller layout,
префікси, кольори для terminal output і передавати помилки запису в error
handler.

У сучасному Go цей пакет варто сприймати як практичний logging facade і output
router, а не як заміну всім сценаріям `log/slog`. Він корисний, коли
застосунок хоче просте multi-destination логування з level masks і контролем
формату, залишаючись близько до стандартної моделі логування Go.

```go
package main

import "github.com/goloop/log/v2"

func main() {
	logger := log.New("APP")

	logger.Info("service started")
	logger.Warnf("cache miss for key %q", "user:42")
	logger.Error("background job failed")
}
```

**Детальніше:** [github.com/goloop/log](https://github.com/goloop/log) · [довідник](https://pkg.go.dev/github.com/goloop/log/v2)

## middlewares

`middlewares` - набір HTTP middleware для стандартного `net/http`. Кожен
middleware має звичну форму `func(http.Handler) http.Handler`, тому працює з
будь-яким роутером: стандартним `http.ServeMux`, роутером `mux` чи рукописними
handlers.

Це не framework. Він закриває типові наскрізні потреби, які стандартна
бібліотека лишає осторонь - request identifiers, real client IP, panic recovery,
логування запитів, timeouts, стиснення відповідей, обмеження конкурентності,
CORS і security headers - і логує через стандартний `log/slog`.

```go
package main

import (
	"net/http"

	"github.com/goloop/middlewares"
)

func main() {
	mux := http.NewServeMux()
	// ... реєстрація handlers на mux ...

	h := middlewares.Chain(
		middlewares.RequestID(),
		middlewares.RealIP(),
		middlewares.Recoverer(),
		middlewares.Logger(),
		middlewares.Compress(),
	)(mux)

	http.ListenAndServe(":8080", h)
}
```

**Детальніше:** [github.com/goloop/middlewares](https://github.com/goloop/middlewares) · [довідник](https://pkg.go.dev/github.com/goloop/middlewares)

## mux

`mux` - невеликий шар роутингу поверх стандартного `net/http.ServeMux`. Починаючи
з Go 1.22 стандартний мультиплексер уже розуміє method patterns, wildcard-сегменти
і precedence, тому `mux` його не замінює: він додає ергономіку, якої стандартній
бібліотеці бракує - method helpers, prefix groups, ланцюжки middleware і
опційний handler, що повертає помилку.

Патерни - це звичайні патерни `net/http.ServeMux`, а не власний синтаксис, і сам
`Router` є `http.Handler`, тож він компонується з рештою `net/http`.

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/goloop/mux"
)

func main() {
	r := mux.New()

	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("ok"))
	})

	r.Route("/api/v1", func(r *mux.Router) {
		r.Get("/users/{id}", func(w http.ResponseWriter, req *http.Request) {
			fmt.Fprintf(w, "user %s", req.PathValue("id"))
		})
	})

	http.ListenAndServe(":8080", r)
}
```

**Детальніше:** [github.com/goloop/mux](https://github.com/goloop/mux) · [довідник](https://pkg.go.dev/github.com/goloop/mux)

## norm

`norm` - «пишучий» компаньйон до `is`: де `is` читає (чи це email?), `norm`
чистить значення до канонічної форми цього типу й перевіряє його. Кожна функція
повертає нормалізоване значення й чи воно валідне, тож обробник форми може
прийняти трохи неохайний ввід і зберегти чистий результат. Покриває email, URL,
UUID, MAC, IBAN, банківську картку, телефон, IP тощо, і додає Unicode-toolkit
для обрізання й видалення невидимих чи керівних символів.

```go
package main

import (
	"fmt"

	"github.com/goloop/norm"
)

func main() {
	fmt.Println(norm.Email("  Example @ Gmail.com ")) // Example@gmail.com true

	iban, ok := norm.IBAN("de89 3704 0044 0532 0130 00")
	fmt.Println(iban, ok) // DE89370400440532013000 true

	// Clean прибирає невидимі/керівні символи й стискає пробіли.
	fmt.Printf("%q\n", norm.Clean("  ragged   name  ")) // "ragged name"
}
```

**Детальніше:** [github.com/goloop/norm](https://github.com/goloop/norm) · [довідник](https://pkg.go.dev/github.com/goloop/norm)

## opt

`opt` парсить аргументи командного рядка у структуру. Це пакет для CLI-програм,
яким потрібен типізований config-об'єкт замість ручного обходу `os.Args` і
викликів `strconv`. Поля налаштовуються тегами: short flags, long aliases,
defaults, help text, separators, required values і positional arguments.

У v2 parser слідує звичним Unix/POSIX очікуванням: bool flags є switches,
`--no-name` вимикає bool, `--flag=value` підтримується, `--` завершує parsing
опцій, а від'ємні числа можуть бути values або positional arguments там, де це
доречно.

```go
package main

import (
	"log"

	"github.com/goloop/opt/v2"
)

type Args struct {
	Host    string   `opt:"H" alt:"host" def:"127.0.0.1"`
	Port    int      `opt:"p" alt:"port" def:"8080"`
	Verbose bool     `opt:"v" alt:"verbose"`
	Files   []string `opt:"[]"`
}

func main() {
	var args Args
	if err := opt.Unmarshal(&args); err != nil {
		log.Fatal(err)
	}
}
```

**Детальніше:** [github.com/goloop/opt](https://github.com/goloop/opt) · [довідник](https://pkg.go.dev/github.com/goloop/opt/v2)

## pgc

`pgc` компілює анотовані SQL-запити в типобезпечний Go-пакет для PostgreSQL.
Ти пишеш звичайний SQL поруч із міграціями; pgc питає твою базу розробки, чим
насправді є кожен параметр і кожна колонка - оракулом типів виступає сам
сервер, стейтменти готуються й описуються, але ніколи не виконуються - і
генерує код, який читається як написаний людиною: явні виклики `Scan`, godoc
на кожному символі, нуль рефлексії в рантаймі. Згенерований пакет імпортує
лише стандартну бібліотеку (`database/sql`, `context`, `time`), а сам pgc не
має жодної сторонньої залежності: він говорить wire-протоколом PostgreSQL
напряму.

```sql
-- queries/users.sql

-- name: GetUser :one
-- Returns a single user by primary key.
SELECT id, email, name, created_at
FROM users
WHERE id = $1;
```

```go
q := db.New(sqlDB) // згенерований пакет; *sql.DB або *sql.Tx

u, err := q.GetUser(ctx, 42)   // (User, error), типи з живої схеми
users, err := q.ListUsers(ctx, 10, 0)
err = q.WithTx(tx).DeleteUser(ctx, u.ID)
```

**Детальніше:** [github.com/goloop/pgc](https://github.com/goloop/pgc) · [довідник](https://pkg.go.dev/github.com/goloop/pgc)

## qp

`qp` читає URL query parameters у типізовані Go-значення. Він замінює
повторюваний код із `r.URL.Query().Get(...)`, `strconv.Atoi`, default handling
і range checks одним компактним API.

Використовуйте `qp.New(r.URL)`, коли handler читає кілька параметрів з одного
URL: query буде розібраний один раз, а далі доступні typed readers. Для
одноразового читання є top-level helpers. Опції покривають defaults, ranges,
allowed values, slice splitting і per-element validation.

```go
package main

import (
	"net/http"

	"github.com/goloop/qp/v2"
)

func handler(w http.ResponseWriter, r *http.Request) {
	q := qp.New(r.URL)

	page := q.Int("page", qp.Default(1), qp.Min(1))
	limit := q.Int("limit", qp.Default(20), qp.Between(1, 100))
	tags := q.StringSlice("tag")

	_, _, _ = page, limit, tags
}
```

**Детальніше:** [github.com/goloop/qp](https://github.com/goloop/qp) · [довідник](https://pkg.go.dev/github.com/goloop/qp/v2)

## resp

`resp` - тонкий helper-шар поверх `net/http` для типових HTTP-відповідей. Він
закриває JSON, JSONP, XML, HTML, strings, bytes, redirects, downloads, cookies,
status codes і headers, не перетворюючись на web framework.

Важлива деталь v2 - safe-by-default encoding: JSON/JSONP/XML спочатку
кодуються в pooled buffer, тому serialization error повертається до того, як
HTTP status буде відправлено клієнту. Для великих payloads можна явно увімкнути
direct streaming, якщо такий trade-off кращий.

```go
package main

import (
	"net/http"

	"github.com/goloop/resp/v2"
)

func handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("id") == "" {
		_ = resp.Error(w, http.StatusBadRequest, "missing id")
		return
	}

	_ = resp.JSON(w, resp.R{"ok": true}, resp.SecurityHeaders())
}
```

**Детальніше:** [github.com/goloop/resp](https://github.com/goloop/resp) · [довідник](https://pkg.go.dev/github.com/goloop/resp/v2)

## scs

`scs` означає String Case Style. Він перетворює identifiers між `camelCase`,
`PascalCase`, `snake_case`, `kebab-case`, `SCREAMING_SNAKE_CASE`, `dot.case`,
`Title Case` і `Sentence case`.

Усі converters використовують один tokenizer, тому не треба знати початковий
стиль рядка. Пакет також може визначити стиль, якщо відповідь однозначна,
розбити текст на слова, ітерувати слова через `iter.Seq` і використовувати
immutable `Caser` для opt-in acronyms на кшталт `ID`, `URL`, `HTTP`.

```go
package main

import (
	"fmt"

	"github.com/goloop/scs/v2"
)

func main() {
	fmt.Println(scs.ToSnake("HTTPServerID")) // http_server_id
	fmt.Println(scs.ToCamel("user_id"))      // userId

	c := scs.New(scs.WithAcronyms("ID", "URL"))
	fmt.Println(c.ToPascal("user_id")) // UserID
}
```

**Детальніше:** [github.com/goloop/scs](https://github.com/goloop/scs) · [довідник](https://pkg.go.dev/github.com/goloop/scs/v2)

## set

`set` - generic-множина для comparable Go-значень. Вона напряму побудована на
`map[T]struct{}`, тому identity - це рівно Go `==`: без reflection, без
custom hashing і без втрати елементів через collision.

Пакет варто використовувати для deduplication, membership checks, set algebra
і relation checks: union, intersection, difference, symmetric difference,
subset/superset і disjointness. Також є functional helpers, JSON support і
ітерація через `iter.Seq`.

```go
package main

import (
	"fmt"

	"github.com/goloop/set/v2"
)

func main() {
	a := set.New(1, 2, 3)
	b := set.New(3, 4)

	fmt.Println(set.Sorted(a.Union(b)))       // [1 2 3 4]
	fmt.Println(a.Contains(2), a.Contains(9)) // true false
}
```

**Детальніше:** [github.com/goloop/set](https://github.com/goloop/set) · [довідник](https://pkg.go.dev/github.com/goloop/set/v2)

## slug

`slug` генерує URL-friendly slug-и з Unicode-тексту. Він використовує `t13n`
для транслітерації, а потім нормалізує результат у слова, з'єднані
separator-ом. Пунктуація стає межею слова, а не зникає так, щоб склеїти слова.

Для простих випадків є package-level helpers, а для повного контролю можна
створити immutable `Slug` з options: мовні правила, custom separator, maximum
length і fallback для порожнього результату. Пакет також може перевірити, чи
рядок уже є canonical slug, і згенерувати unique slug через predicate вашого
storage.

```go
package main

import (
	"fmt"

	"github.com/goloop/slug/v2"
	"github.com/goloop/slug/v2/lang"
)

func main() {
	s := slug.New(slug.WithLang(lang.UK), slug.WithMaxLength(32))

	fmt.Println(slug.Lower("Hello, World!")) // hello-world
	fmt.Println(s.Make("Привіт, світ!"))    // Pryvit-svit
}
```

**Детальніше:** [github.com/goloop/slug](https://github.com/goloop/slug) · [довідник](https://pkg.go.dev/github.com/goloop/slug/v2)

## t13n

`t13n` означає transliteration. Він перетворює Unicode-текст в ASCII, за
потреби застосовуючи регіональні мовні правила. Це нижчий рівень текстового
перетворення для `slug`, але пакет корисний і сам по собі: для ASCII-only
search keys, filenames, identifiers або logs.

Пакет надає conversion для однієї руни, цілого рядка, language-specific
conversion і custom rendering rules. Базова таблиця вбудована компактно і
декодується ліниво, тому застосунок платить за неї лише коли реально виконує
транслітерацію.

```go
package main

import (
	"fmt"

	"github.com/goloop/t13n/v2"
	"github.com/goloop/t13n/v2/lang"
)

func main() {
	fmt.Println(t13n.Make("世界"))                       // Shi Jie
	fmt.Println(t13n.Trans(lang.UK, "Доброго вечора")) // Dobroho vechora

	s, ok := t13n.Rune('界')
	fmt.Println(s, ok) // Jie  true
}
```

**Детальніше:** [github.com/goloop/t13n](https://github.com/goloop/t13n) · [довідник](https://pkg.go.dev/github.com/goloop/t13n/v2)

## trit

`trit` реалізує тризначну логіку: `False`, `Unknown`, `True`. Він корисний там,
де значення не є простим yes/no: nullable database booleans, частково відома
конфігурація, feature flags зі спадкуванням defaults, policy decisions або
будь-який домен, де “unknown” не можна мовчки перетворити на `false`.

Zero value є `Unknown`, тому неініціалізовані значення мають осмислений стан.
Пакет надає truth-table operations, generic aggregate functions, parsing,
JSON/text/SQL integration і ordering (`False < Unknown < True`).

```go
package main

import (
	"fmt"

	"github.com/goloop/trit/v2"
)

func main() {
	enabled := trit.Unknown
	enabled.Default(trit.True)

	fmt.Println(enabled.And(trit.True))              // True
	fmt.Println(trit.Consensus(trit.True, enabled)) // True
}
```

**Детальніше:** [github.com/goloop/trit](https://github.com/goloop/trit) · [довідник](https://pkg.go.dev/github.com/goloop/trit/v2)

## websocket

`websocket` реалізує протокол WebSocket (RFC 6455) поверх стандартної
бібліотеки. Він надає server-side upgrade, client-side dial, розширення
permessage-deflate і узгодження subprotocol.

З'єднання - це `Conn`. Server upgrade за замовчуванням приймає only-same-origin
запити, що захищає від cross-site hijacking; коли потрібні cross-origin клієнти,
задайте дозволені origins явно.

```go
package main

import (
	"net/http"

	"github.com/goloop/websocket"
)

func echo(w http.ResponseWriter, r *http.Request) {
	ws, err := websocket.Upgrade(w, r)
	if err != nil {
		return
	}
	defer ws.Close()

	for {
		mt, data, err := ws.ReadMessage()
		if err != nil {
			break
		}
		if err := ws.WriteMessage(mt, data); err != nil {
			break
		}
	}
}
```

**Детальніше:** [github.com/goloop/websocket](https://github.com/goloop/websocket) · [довідник](https://pkg.go.dev/github.com/goloop/websocket)

## app

`app` - невелике ядро життєвого циклу й композиції: воно володіє послідовністю
start/stop сервісу, щоб твій `main` її не повторював. Ти реєструєш компоненти -
будь-що з `Name`, неблокуючим `Start` і `Stop` - і `app` стартує їх у порядку,
чекає сигнал, скасування батьківського контексту чи фатальну помилку компонента,
тоді зупиняє їх у зворотному порядку в межах обмеженого таймауту.

Це не framework: без глобального стану, без DI-контейнера, без роутингу. Готові
компоненти покривають типові випадки - `HTTPServer` обгортає `*http.Server`,
`Worker` обгортає блокуючий цикл, `Closer` обгортає функцію очищення - а знімок
стану (read-only) може перетворити observability-модуль на health-перевірку,
причому жоден пакет не імпортує інший.

```go
package main

import (
	"context"
	"net/http"
	"time"

	"github.com/goloop/app"
)

func main() {
	a := app.New("api", app.WithShutdownTimeout(10*time.Second))
	a.Use(app.HTTPServer("http", &http.Server{Addr: ":8080", Handler: mux}))
	a.Use(app.Closer("db", pool.Close))

	if err := a.Run(context.Background()); err != nil {
		panic(err)
	}
}
```

**Детальніше:** [github.com/goloop/app](https://github.com/goloop/app) · [довідник](https://pkg.go.dev/github.com/goloop/app)

## observe

`observe` надає health-, readiness- і build-info-ендпоінти для сервісу, лише
стандартна бібліотека. Він володіє моделлю операційного статусу, тож кожен
сервіс виставляє однакову форму `/healthz`, `/readyz` і build-info без
копіпасту handlers. Health - це liveness рівня процесу, завжди ok; readiness
виконує зареєстровані перевірки залежностей паралельно й падає, якщо якась
непридатна.

Кожна перевірка обмежена per-check таймаутом, що діє навіть коли перевірка
ігнорує свій контекст, перевірка, що панікує, відновлюється замість падіння
процесу, а деталі помилок редагуються за замовчуванням, тож health-ендпоінт
ніколи не витікає хостами чи креденшалами. Він ніколи не імпортує lifecycle-ядро:
містком служить перевірка над знімком стану.

```go
package main

import (
	"context"
	"net/http"

	"github.com/goloop/observe"
)

func main() {
	obs := observe.New(observe.WithService("api"))
	obs.Check("postgres", func(ctx context.Context) error { return pool.Ping(ctx) })

	http.Handle("/healthz", obs.HealthHandler()) // 200, поки процес живий
	http.Handle("/readyz", obs.ReadyHandler())   // 503, якщо залежність впала
	http.ListenAndServe(":8080", nil)
}
```

**Детальніше:** [github.com/goloop/observe](https://github.com/goloop/observe) · [довідник](https://pkg.go.dev/github.com/goloop/observe)

## jwt

`jwt` видає й перевіряє компактні JSON Web Tokens, навмисно обмежені до HS256,
лише стандартна бібліотека. Він слідує JWS compact serialization з RFC 7515 і
registered claims з RFC 7519, але підтримує рівно один алгоритм зі строгими
дефолтами: немає узгодження алгоритмів, немає `none` і немає асиметричних
ключів - менша поверхня є безпечнішою поверхнею.

Ключ має бути щонайменше 32 байти; перевірка вимагає `alg=HS256` і наявного
expiry, порівнює HMAC constant-time перевіркою до читання payload і відхиляє
`crit`-заголовки, нестрогий base64url і registered claim неправильного типу.
Ротація ключів вбудована: підписуй основним ключем, перевіряй проти будь-якого
налаштованого.

```go
package main

import (
	"fmt"
	"time"

	"github.com/goloop/jwt"
)

func main() {
	key := []byte("a-32-byte-or-longer-secret-key!!")

	token, _ := jwt.Sign(jwt.Claims{
		Subject:   "user-123",
		Issuer:    "api",
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}, key)

	claims, err := jwt.Verify(token, key, jwt.WithIssuer("api"))
	fmt.Println(claims.Subject, err) // user-123 <nil>
}
```

**Детальніше:** [github.com/goloop/jwt](https://github.com/goloop/jwt) · [довідник](https://pkg.go.dev/github.com/goloop/jwt)

## auth

`auth` надає примітиви автентифікації: хешування паролів, access-токени,
HTTP-middleware і ротацію refresh-токенів. Це безпечні будівельні блоки без
перетворення на identity-платформу - немає репозиторію користувачів, немає
RBAC-схеми й немає OAuth, тож персистентність і керування користувачами лишаються
у твоєму застосунку. Він залежить лише від стандартної бібліотеки та сусіднього
`jwt`.

Паролі хешуються самоописним PBKDF2-кодуванням (інтерфейс `PasswordHasher`
дозволяє замінити алгоритм); access-токени - це HS256 JWT з обов'язковим expiry
та constant-time перевіркою; middleware `Bearer` і `Cookie` автентифікують
запит, а `Require` й `RequireScope` вимагають наявності й scope. Refresh-токени
підтримують ротацію.

```go
package main

import (
	"net/http"

	"github.com/goloop/auth"
)

func main() {
	tm := auth.NewTokenManager(secret, auth.WithIssuer("api"))

	token, _ := tm.Issue(auth.Subject{ID: "user-1", Scopes: []string{"read"}})
	_ = token

	http.Handle("/me", tm.Bearer(auth.Require(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			sub, _ := auth.SubjectFrom(r.Context())
			w.Write([]byte(sub.ID))
		}))))
}
```

**Детальніше:** [github.com/goloop/auth](https://github.com/goloop/auth) · [довідник](https://pkg.go.dev/github.com/goloop/auth)

## session

`session` надає безпечні, підписані cookie-сесії для браузерних застосунків,
лише стандартна бібліотека. Уся сесія живе в HMAC-SHA256 підписаній cookie, тож
немає серверного сховища, яке треба тримати. Це компаньйон до token-based auth, а
не заміна: `session` володіє cookie й браузерним станом, а auth-пакет володіє
суб'єктами, паролями й токенами.

Cookie є HttpOnly, SameSite=Lax і підписана; вмикай `Secure` у продакшені.
Payload версіонований, тож формат може еволюціонувати без ламання старих cookie,
а ротація ключів вбудована. Читай через `LoadOrNew` (або middleware плюс
`session.From`), тоді `Save`, щоб зберегти.

```go
package main

import (
	"net/http"

	"github.com/goloop/session"
)

func main() {
	m := session.New(secret,
		session.WithName("sid"),
		session.WithSecure(true),
		session.WithTTL(24*time.Hour),
	)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s := m.LoadOrNew(r)
		s.Set("theme", "dark")
		_ = m.Save(w, s)
	})
}
```

**Детальніше:** [github.com/goloop/session](https://github.com/goloop/session) · [довідник](https://pkg.go.dev/github.com/goloop/session)

## mail

`mail` будує й надсилає вихідну пошту, лише стандартна бібліотека. Це невеликий
транспорт і будівник повідомлень, а не marketing-платформа чи template-двигун:
він конструює RFC 5322/MIME повідомлення й доставляє його через SMTP, з drop-in
транспортами для розробки й тестів. Text- та/або HTML-тіла, вкладення,
валідовані адреси й RFC 2047-кодовані display names - усе оброблено.

`Sender` доставляє повідомлення: `NewSMTP` надсилає через SMTP (STARTTLS за
замовчуванням), `NewLogger` пише збудоване повідомлення у writer для розробки, а
`NewCapture` записує повідомлення для тестів - тож той самий шлях коду працює у
продакшені й під тестом.

```go
package main

import (
	"context"

	"github.com/goloop/mail"
)

func main() {
	sender := mail.NewSMTP(mail.SMTPConfig{
		Host: "smtp.example.com",
		From: mail.Address{Email: "no-reply@example.com"},
	})

	_ = sender.Send(context.Background(), mail.Message{
		To:      []mail.Address{{Email: "user@example.com"}},
		Subject: "Welcome",
		Text:    "Hello and welcome.",
		HTML:    "<p>Hello and welcome.</p>",
	})
}
```

**Детальніше:** [github.com/goloop/mail](https://github.com/goloop/mail) · [довідник](https://pkg.go.dev/github.com/goloop/mail)

## cli

`cli` - командний інструмент, а не бібліотека, що генерує й перевіряє goloop-first
Go-проєкти. Це невеликий диспетчер поверх `os.Args` без сторонніх залежностей, і
він ніколи не комітить, не пушить і не деплоїть. Встанови через
`go install github.com/goloop/cli@latest` (команда - `goloop`).

Він генерує мінімальний сервіс (`goloop new`), повідомляє, які очікувані
інструменти встановлені (`goloop doctor`), підсумовує модуль у поточній теці
(`goloop status`) і сканує дерево на випадкові артефакти редактора/асистента
(`goloop check`).

```text
goloop new <name> [--module path] [--dir path] [--force]
goloop check [dir]     сканувати дерево проєкту
goloop doctor          перевірити наявність очікуваних інструментів
goloop status          підсумувати модуль у поточній теці
goloop version         вивести версію CLI
```

**Детальніше:** [github.com/goloop/cli](https://github.com/goloop/cli) · [довідник](https://pkg.go.dev/github.com/goloop/cli)

## Як обрати

Використовуйте `env` і `opt` на старті програми, `mux`, `middlewares`, `qp` і
`resp` у HTTP handlers, `websocket` для realtime-з'єднань, `ai` щоб говорити до
LLM-провайдерів за одним інтерфейсом, `pgc` щоб компілювати свій SQL у
типізований Go проти PostgreSQL, `is` для валідації, `log` для operational
output, `set` і `g` у business logic, `key` для публічних reversible IDs, `kind`
коли парсеру чи декодеру треба інтроспектувати типи, `scs`, `slug` і `t13n` для
роботи з рядками, а `trit` тоді, коли unknown state є повноцінним значенням.

Для форми цілого сервісу візьми прикладний ярус: `app`, щоб володіти життєвим
циклом start/stop і graceful shutdown, `observe` для health- і
readiness-ендпоінтів, `auth`, `jwt` і `session` для токенів, захисту маршрутів і
браузерних сесій, а `mail` для вихідної пошти. Команда `cli` генерує новий
goloop-first проєкт, тож ти стартуєш із робочого скелета. Як ці яруси
поєднуються - див. [Побудова сервісу](#побудова-сервісу).

Кожен модуль навмисно невеликий. Не потрібно приймати всю групу одразу:
встановлюйте тільки той module, який закриває конкретну задачу перед вами.
