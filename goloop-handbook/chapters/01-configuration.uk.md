[« Передмова](00-preface.uk.md) · [Зміст](../main.uk.md) · [JSON HTTP API »](02-http-json-api.uk.md)

---

# 01. Конфігурація, якою можна керувати

**Задача.** Читати налаштування сервісу з файлу `.env` у локальній розробці,
дати справжнім змінним оточення перемагати в проді, а прапорцям командного рядка
- перемагати обидва; з розумними дефолтами, коли нічого не задано. Тримати
секрети подалі від командного рядка.

**Модулі.** [`env`](https://github.com/goloop/env) читає `.env`-файли й оточення
в структуру; [`opt`](https://github.com/goloop/opt) розбирає прапорці командного
рядка в *ту саму* структуру.

**Рецепт.** [`recipes/001-configuration`](../recipes/001-configuration/)

## Ідея

Конфігурація - це перше, що потрібно кожному сервісу, і перше, що роблять
неправильно: `.env`, який мусить існувати, інакше програма падає; прапорці, що
не збігаються зі змінними оточення; секрети, що осідають в історії shell.

GoLoop трактує всю конфігурацію як одну звичайну структуру. Кожне поле каже,
звідки може взятися його значення, у трьох тегах:

```go
type Config struct {
	Addr     string        `env:"APP_ADDR" def:":8080" opt:"addr" help:"listen address"`
	Env      string        `env:"APP_ENV" def:"dev" opt:"env" help:"dev or prod"`
	Timeout  time.Duration `env:"APP_TIMEOUT" def:"15s" opt:"timeout" help:"request timeout"`
	Debug    bool          `env:"APP_DEBUG" def:"false" opt:"debug" help:"verbose logging"`
	Secret   string        `env:"APP_SECRET" opt:"-"`
	Replicas int           `env:"APP_REPLICAS" def:"1" opt:"replicas" help:"worker count"`
}
```

- `env` називає змінну оточення.
- `def` - вбудований дефолт.
- `opt` називає прапорець; `opt:"-"` означає «ніколи не прапорець», і саме так
  `Secret` лишається поза `--help` та історією shell.

Типи полів - звичайні Go-типи: `time.Duration`, `bool`, `int` розбираються за
тебе.

## Нашарування джерел

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

`env.LoadSafe` тут тихий герой: `env.Load` падає, якщо названого файлу немає, і
це змушує кожного викликача обгортати його `os.Stat`. `LoadSafe` пропускає
відсутній файл і все одно повідомляє справжні помилки парсингу - саме та
поведінка «прочитай `.env`, якщо він є», якої ти хочеш.

## Звіт виконання

Зібрано, протестовано й запущено чотири рази, щоб показати порядок пріоритету
(`go test ./...`, далі `go run .` з різними джерелами):

```text
$ go test ./...
ok  	goloop.one/handbook/001-configuration	0.002s

$ go run .                       # лише дефолти
addr=:8080 env=dev timeout=15s debug=false replicas=1 secret_set=false

$ printf 'APP_ENV=staging\nAPP_ADDR=:9090\nAPP_SECRET=from-dotenv\n' > .env
$ go run .                       # застосувався файл .env
addr=:9090 env=staging timeout=15s debug=false replicas=1 secret_set=true

$ APP_ENV=prod go run .          # справжня змінна перекриває значення з .env
addr=:9090 env=prod timeout=15s debug=false replicas=1 secret_set=true

$ APP_ENV=prod go run . --env qa --replicas 6 --debug   # прапорець перекриває все
addr=:9090 env=qa timeout=15s debug=true replicas=6 secret_set=true
```

Прочитай чотири запуски згори донизу - і видно драбину: дефолти, потім `.env`,
потім справжнє оточення б'є `.env`, потім прапорець б'є все. Від `.env` і далі
`secret_set=true`, а `APP_SECRET` ніде не з'являється як прапорець.

## Що ти дізнався

- Опиши конфігурацію **один раз**, як структуру з тегами `env`/`def`/`opt`.
- Застосовуй джерела від найнижчого пріоритету: `env.LoadSafe` ->
  `env.Unmarshal` -> `opt.UnmarshalArgs`.
- Для секретів став `opt:"-"`, щоб їх читали лише з оточення.
- `LoadSafe` дає тому самому бінару працювати в dev і prod без зміни коду.

Далі: віддамо щось по HTTP.

---

[« Передмова](00-preface.uk.md) · [Зміст](../main.uk.md) · [JSON HTTP API »](02-http-json-api.uk.md)
