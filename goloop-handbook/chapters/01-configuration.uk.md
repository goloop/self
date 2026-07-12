[« Передмова](00-preface.uk.md) · [Зміст](../main.uk.md) · [JSON HTTP API »](02-http-json-api.uk.md)

---

# 01. Конфігурація, якою можна керувати

**Задача.** Читати налаштування сервісу з файлу `.env` у локальній розробці,
дати справжнім змінним оточення перемагати в проді, а прапорцям - перемагати
обидва; з розумними дефолтами, коли нічого не задано. Тримати секрети подалі від
командного рядка.

**Модулі.** [`env`](https://github.com/goloop/env) читає `.env`-файли й оточення
в структуру; [`opt`](https://github.com/goloop/opt) розбирає прапорці в *ту саму*
структуру.

**Рецепт.** [`recipes/001-configuration`](../recipes/001-configuration/)

## Структура конфігурації

Опишіть усю конфігурацію один раз, як звичайну структуру. Кожне поле каже,
звідки може взятися його значення, у трьох тегах:

```go
type Config struct {
	Addr     string        `env:"APP_ADDR" def:":8080" opt:"addr" help:"listen address"`
	Env      string        `env:"APP_ENV" def:"dev" opt:"env" help:"dev or prod"`
	Timeout  time.Duration `env:"APP_TIMEOUT" def:"15s" opt:"timeout" help:"request timeout"`
	Debug    bool          `env:"APP_DEBUG" def:"false" opt:"debug" help:"verbose logging"`
	Secret   string        `env:"APP_SECRET" opt:"-"`
	Replicas int           `env:"APP_REPLICAS" def:"1" opt:"replicas" help:"worker count"`
	Origins  []string      `env:"APP_ORIGINS" def:"http://localhost:3000" sep:"," opt:"origins"`
}
```

`env` називає змінну оточення, `def` - дефолт, `opt` - прапорець. `opt:"-"`
означає «ніколи не прапорець», і саме так `Secret` лишається поза `--help` та
історією shell. Тег `sep` розбиває одну змінну на зріз, тож
`APP_ORIGINS="https://a.example,https://b.example"` стає `[]string`. Типи полів -
звичайні Go-типи, розбираються за вас.

## Приклад A - нашаруйте джерела

Пріоритет - це просто порядок застосування джерел, від найнижчого:

```go
func load(args []string) (Config, error) {
	// LoadSafe читає .env, коли він є, і нічого не робить, коли його немає,
	// тож той самий бінар працює локально (з файлом) і в проді (без нього).
	if err := env.LoadSafe(); err != nil {
		return Config{}, fmt.Errorf("loading .env: %w", err)
	}
	var cfg Config
	if err := env.Unmarshal(&cfg); err != nil { // дефолти + оточення
		return Config{}, fmt.Errorf("environment: %w", err)
	}
	if err := opt.UnmarshalArgs(&cfg, args); err != nil { // прапорці перемагають
		return Config{}, fmt.Errorf("flags: %w", err)
	}
	return cfg, nil
}
```

`env.LoadSafe` тут тихий герой: `env.Load` падає, коли названого файлу немає, і
це змушує кожного викликача обгортати його. `LoadSafe` пропускає відсутній файл
і все одно повідомляє справжні помилки парсингу - саме та поведінка «прочитай
`.env`, якщо він є», якої ви хочете.

## Приклад B - розберіть сніпет у мапу

Іноді вам потрібна не структура, а лише пари ключ/значення. `env.Parse` читає
`.env`-текст з будь-якого `io.Reader` у `map[string]string`:

```go
m, _ := env.Parse(strings.NewReader("HOST=db.internal\nPORT=5432\n# a comment\nTAGS=a,b,c\n"))
// m["HOST"] == "db.internal", коментарі проігноровано
```

## Приклад C - запишіть структуру назад

`env.MarshalWriter` перетворює структуру назад на `.env`-рядки, зручно для
генерації шаблону. Зверніть увагу: він пише *кожне* поле, зокрема `Secret`, тож
редагуйте секрети перед тим, як ділитися згенерованим файлом.

```go
var b strings.Builder
_ = env.MarshalWriter(&b, cfg) // APP_ADDR=:8080\nAPP_ENV=...\n
```

## Приклад D - зробіть поле обов'язковим

Відсутнє налаштування має впасти голосно на старті, а не виринути таємничим
нульовим значенням пізніше. Додайте `required` до тегу `env`: без дефолта й без
заданого значення `env.Unmarshal` повертає `env.ErrRequired`, називаючи ключ:

```go
type mustHave struct {
	DatabaseURL string `env:"DATABASE_URL,required"`
}

var need mustHave
err := env.Unmarshal(&need)              // відсутнє -> env.ErrRequired
errors.Is(err, env.ErrRequired)          // true, а повідомлення називає DATABASE_URL

os.Setenv("DATABASE_URL", "postgres://localhost/app")
_ = env.Unmarshal(&need)                 // тепер nil; need.DatabaseURL задано
```

Використовуйте `required` для значень, без яких сервіс справді не стартує, і
`def` для всього, що має розумний запасний варіант.

## Звіт виконання

Протестовано, потім запущено раз із наявним `.env` і заданим прапорцем:

```text
$ go test ./...
ok  	goloop.one/handbook/001-configuration	0.002s

$ printf 'APP_ENV=staging\nAPP_SECRET=from-dotenv\n' > .env
$ go run . --replicas 4
A. layered config (defaults < .env/env < flags):
   addr=:8080 env=staging timeout=15s debug=false replicas=4 secret_set=true
B. parse a .env snippet into a map (no struct):
   HOST=db.internal PORT=5432 TAGS=a,b,c
C. marshal the struct back to .env lines (a template):
   APP_ADDR=:8080
   APP_ENV=staging
   APP_TIMEOUT=15s
   APP_DEBUG=false
   APP_SECRET=from-dotenv
   APP_REPLICAS=4
   APP_ORIGINS=http://localhost:3000
D. a required field (env:"...,required"):
   missing -> error: true
   present -> ok=true value=postgres://localhost/app
```

У A `env=staging` прийшло з `.env`, `replicas=4` - з прапорця, `origins` було
єдиним дефолтним значенням, а секрет прочитано, але ніде не надруковано. У C та
сама структура повертається в `.env`-рядки (і так, `APP_SECRET` там є - це
нагадування редагувати). У D обов'язковий ключ дає помилку без значення й
успіх, щойно його задано.

## Що ви дізналися

- Опишіть конфігурацію **один раз**, як структуру з тегами `env`/`def`/`opt`.
- Застосовуйте джерела від найнижчого пріоритету: `env.LoadSafe` ->
  `env.Unmarshal` -> `opt.UnmarshalArgs`; для секретів `opt:"-"`.
- Тег `sep` розбиває одну змінну на зріз; `env:"NAME,required"` робить значення
  обов'язковим і повертає `env.ErrRequired`, коли його немає.
- `env.Parse` читає сніпет у мапу, коли структура не потрібна.
- `env.MarshalWriter` пише структуру назад у `.env`-рядки (редагуйте секрети).

Далі: віддамо щось по HTTP.

---

[« Передмова](00-preface.uk.md) · [Зміст](../main.uk.md) · [JSON HTTP API »](02-http-json-api.uk.md)
