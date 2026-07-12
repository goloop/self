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

## Приклад D - валідуйте вхідний слаг

Слаг не лише виходить у URL, який ви будуєте; він також повертається, як
`{slug}` у шляху запиту. `slug.IsValid` відповідає «так/ні», не змінюючи
значення, тож обробник може відхилити спотворений сегмент ще до того, як той
торкнеться бази:

```go
slug.IsValid("hello-world")  // true
slug.IsValid("Hello World!") // false - пробіли й пунктуація
slug.IsValid("-bad-")        // false - дефіс на початку/кінці
```

## Приклад E - тримайте акроніми у верхньому регістрі

За замовчуванням `scs` не знає, що `API` - акронім, тож `userApiToken` стає
`UserApiToken`. Навчіть його ваших акронімів через `WithAcronyms`, і вони
лишаться у верхньому регістрі в усіх стилях:

```go
acr := scs.New(scs.WithAcronyms("API", "ID"))
acr.ToPascal("userApiToken") // "UserAPIToken" (звичайний scs: "UserApiToken")
acr.ToPascal("user_id")      // "UserID"       (звичайний scs: "UserId")
```

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
D. validate an incoming slug (slug.IsValid):
   "hello-world"    -> IsValid=true
   "Hello World!"   -> IsValid=false
   "-bad-"          -> IsValid=false
E. acronyms in case conversion (scs WithAcronyms):
   "userApiToken" -> plain pascal="UserApiToken", acronym pascal="UserAPIToken"
   "user_id"      -> plain pascal="UserId", acronym pascal="UserID"
```

Два заголовки «Getting Started» стали `-2` і `-3`, бо перший був уже зайнятий;
кириличні заголовки транслітерувалися й потім послагувалися; а один `Caser` дав
усі стилі іменування з того самого вводу. У D `IsValid` прийняв коректний слаг і
відхилив два спотворені; у E `WithAcronyms` втримав `API` та `ID` у верхньому
регістрі там, де звичайний `scs` перевів їх у title-регістр.

## Що ви дізналися

- `slug.New(slug.WithLowercase())` робить слаги в нижньому регістрі; `MakeUnique`
  тримає їх унікальними проти вашої перевірки «чи зайнято».
- `slug.IsValid` валідує слаг, що приходить у шляху запиту, тож обробник може
  відхилити спотворений сегмент рано.
- `t13n` транслітерує не-латинський текст в ASCII, тож кириличний чи інший
  заголовок отримує справжній слаг, коли ви проганяєте його через `t13n` перед
  `slug`.
- `scs.Caser` конвертує один ідентифікатор між snake, kebab, camel, Pascal,
  screaming-snake і title; `scs.WithAcronyms` тримає відомі акроніми у верхньому
  регістрі.

Починається Частина III: запуск цих шматків як сервісу.

---

[« Сесії, токени й паролі](06-auth.uk.md) · [Зміст](../main.uk.md) · [Життєвий цикл »](08-lifecycle.uk.md)
