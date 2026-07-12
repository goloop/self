[« Сесії, токени й паролі](06-auth.uk.md) · [Зміст](../main.uk.md) · [Життєвий цикл »](08-lifecycle.uk.md)

---

# 07. Слаги, транслітерація й регістри

**Задача.** Перетворити людський заголовок на чистий URL-слаг, навіть коли
заголовок кирилицею; тримати слаг унікальним; і конвертувати ідентифікатор між
стилями іменування.

**Модулі.** [`slug`](https://github.com/goloop/slug) робить URL-слаги,
[`t13n`](https://github.com/goloop/t13n) транслітерує не-латинський текст в
ASCII, а [`scs`](https://github.com/goloop/scs) конвертує між snake, kebab,
camel, Pascal тощо.

**Рецепт.** [`recipes/007-slug`](../recipes/007-slug/)

## Приклад A - унікальний слаг

`slug.New(slug.WithLowercase())` дає мейкер, що повертає слаги в нижньому
регістрі; `MakeUnique` додає `-2`, `-3`, ..., поки колбек повідомляє про збіг:

```go
s := slug.New(slug.WithLowercase())
taken := map[string]bool{"getting-started": true}
out := s.MakeUnique("Getting Started", func(x string) bool { return taken[x] })
// out == "getting-started-2"
```

## Приклад B - транслітерувати, потім слаг

Кириличний заголовок сам по собі не має ASCII-слага. `t13n` спершу транслітерує
його в латиницю; `slug` потім робить URL-сегмент:

```go
tr := t13n.New()
latin := tr.Make("Привіт, світ") // "Privit, svit"
url := s.Make(latin)             // "privit-svit"
```

## Приклад C - стилі іменування

Один `scs.Caser` конвертує ім'я між усіма поширеними стилями:

```go
c := scs.New()
c.ToSnake("userAPIToken")          // "user_api_token"
c.ToKebab("userAPIToken")          // "user-api-token"
c.ToPascal("userAPIToken")         // "UserApiToken"
c.ToScreamingSnake("userAPIToken") // "USER_API_TOKEN"
```

(За замовчуванням `scs` не вважає `API` акронімом; увімкніть це через
`scs.New(scs.WithAcronyms("API"))`, коли хочете `UserAPIToken`.)

## Звіт виконання

```text
$ go test ./...
ok  	goloop.one/handbook/007-slug	0.005s

$ go run .
A. unique URL slug (slug):
   "Getting Started"  -> "getting-started-2"
   "Getting Started"  -> "getting-started-3"
   "Hello, World!"    -> "hello-world"
B. transliterate then slug (t13n + slug):
   "Привіт, світ"       -> "Privit, svit"         -> "privit-svit"
   "Огляд архітектури"  -> "Ogliad arkhitekturi"  -> "ogliad-arkhitekturi"
C. naming styles (scs):
   from "userAPIToken":
     snake="user_api_token" kebab="user-api-token"
     pascal="UserApiToken" camel="userApiToken"
     screaming="USER_API_TOKEN" title="User Api Token"
```

Два заголовки «Getting Started» стали `-2` і `-3`, бо перший був уже зайнятий;
кириличні заголовки транслітерувалися й потім послагувалися; а один `Caser` дав
усі стилі іменування з того самого вводу.

## Що ви дізналися

- `slug.New(slug.WithLowercase())` робить слаги в нижньому регістрі; `MakeUnique`
  тримає їх унікальними проти вашої перевірки «чи зайнято».
- `t13n` транслітерує не-латинський текст в ASCII, тож кириличний чи інший
  заголовок отримує справжній слаг, коли ви проганяєте його через `t13n` перед
  `slug`.
- `scs.Caser` конвертує один ідентифікатор між snake, kebab, camel, Pascal,
  screaming-snake і title; акроніми налаштовувані.

Починається Частина III: запуск цих шматків як сервісу.

---

[« Сесії, токени й паролі](06-auth.uk.md) · [Зміст](../main.uk.md) · [Життєвий цикл »](08-lifecycle.uk.md)
