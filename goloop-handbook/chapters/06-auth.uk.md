[« Мовні моделі](05-ai.uk.md) · [Зміст](../main.uk.md) · [Слаги й регістри »](07-slug.uk.md)

---

# 06. Сесії, токени й паролі

**Задача.** Безпечно впускати користувачів. Зберігати пароль так, щоб витік бази
не роздав plaintext; видавати підписаний токен, який API-клієнт присилає з
кожним запитом; і тримати браузерну сесію в підписаній cookie. Нічого з цього не
винаходьте самі.

**Модулі.** [`auth`](https://github.com/goloop/auth) хешує паролі й видає токени
(справжній JWT через [`jwt`](https://github.com/goloop/jwt)),
[`argon2id`](https://github.com/goloop/argon2id) - memory-hard хешер паролів за
інтерфейсом `auth`, [`session`](https://github.com/goloop/session) веде підписану
cookie-сесію, а [`key`](https://github.com/goloop/key) карбує короткі публічні id.

**Рецепт.** [`recipes/006-auth`](../recipes/006-auth/)

## Приклад A - хеш пароля

Ніколи не зберігайте пароль. `auth.NewPBKDF2` повертає хешер; `Hash` створює
солений PBKDF2-дайджест, а `Verify` перевіряє кандидата за сталий час:

```go
hasher := auth.NewPBKDF2()
encoded, _ := hasher.Hash([]byte("correct horse battery staple"))
// encoded == "pbkdf2-sha256$600000$...": алгоритм, вартість і сіль ідуть разом
ok := hasher.Verify(encoded, []byte("correct horse battery staple")) == nil // true
no := hasher.Verify(encoded, []byte("guess")) == nil                        // false
```

## Приклад B - видати access-токен

`auth.TokenManager` підписує `Subject` у JWT. `Verify` повертає суб'єкта, а
підроблений токен не проходить. `key` дає короткий публічний id для URL:

```go
tm := auth.NewTokenManager([]byte(signingSecret), auth.WithIssuer("handbook"))
token, _ := tm.Issue(auth.Subject{ID: "42", Email: "ada@example.com", Roles: []string{"admin"}})
sub, err := tm.Verify(token) // sub.ID == "42", sub.Roles == ["admin"]
_, bad := tm.Verify(token + "x") // bad != nil

pid, _ := key.NewFixed("23456789abcdefghjkmnpqrstuvwxyz", 10)
code, _ := pid.RandomCrypto() // напр. "qj23hctskd", безпечно показати
```

## Приклад C - cookie-сесія

Для браузера тримайте сесію в підписаній cookie. `Save` її пише; пізніший запит
читає її назад через `Load`. Тут `httptest` заміняє браузер:

```go
mgr := session.New([]byte(sessionKey))
s := mgr.LoadOrNew(req)
s.Subject = "42"
s.Set("theme", "dark")
_ = mgr.Save(w, s) // підписує й ставить cookie

// на наступному запиті з тією cookie:
loaded, _ := mgr.Load(req2) // loaded.Subject == "42", loaded.Get("theme") == "dark"
```

## Приклад D - зробіть токен таким, що спливає

Access-токен має бути короткоживучим. `auth.WithTTL` задає час життя;
`auth.WithClock` фіксує «зараз», тож тест може побачити прострочення без
очікування. Тут два менеджери ділять секрет, але читають різні годинники: один
перевіряє в межах вікна, інший - після нього:

```go
issuedAt := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
short := auth.NewTokenManager(secret, auth.WithTTL(30*time.Second),
	auth.WithClock(func() time.Time { return issuedAt }))
tok, _ := short.Issue(auth.Subject{ID: "42"})

early := auth.NewTokenManager(secret,
	auth.WithClock(func() time.Time { return issuedAt.Add(10 * time.Second) }))
late := auth.NewTokenManager(secret,
	auth.WithClock(func() time.Time { return issuedAt.Add(time.Minute) }))

_, errEarly := early.Verify(tok) // nil: ще в межах 30с вікна
_, errLate := late.Verify(tok)   // не nil: спливло
```

У проді ви передаєте справжній годинник (за замовчуванням) і просто видаєте
токени з коротким TTL; фіксований годинник - лише щоб показати межу точно.

## Приклад E - refresh-токен

Короткому access-токену потрібен довший спосіб отримати новий.
`auth.NewRefreshToken` повертає запис для зберігання й непрозорий рядок
`"id.secret"` для клієнта. Ви зберігаєте лише хеш, ніколи не секрет, тож витік
вашого сховища не дасть карбувати токени. `ParseRefreshToken` розбиває
повернений рядок, а `record.Verify(secret)` перевіряє його за сталий час:

```go
record, opaque, _ := auth.NewRefreshToken("42", 30*24*time.Hour)
// віддайте `opaque` клієнту; збережіть `record` (record.Hash, ніколи не секрет).

id, secret, _ := auth.ParseRefreshToken(opaque) // знайдіть запис за id
ok := record.Verify(secret) == nil              // true; невірний секрет - false
```

## Приклад F - memory-hard хешування через argon2id

`PasswordHasher` - це інтерфейс, а `auth.NewPBKDF2` - лише одна його реалізація.
[`argon2id`](https://github.com/goloop/argon2id) - інша: memory-hard хешер, що
коштує фіксований обсяг RAM на кожну спробу, і саме це робить викрадений хеш
дорогим для злому. `argon2id.New()` має ту саму форму `Hash`/`Verify`, тож
підставляється без взаємного імпорту пакетів:

```go
var hasher auth.PasswordHasher = argon2id.New() // memory-hard, рекомендований
encoded, _ := hasher.Hash([]byte(password))     // $argon2id$v=19$m=65536,t=1,p=4$...
ok := hasher.Verify(encoded, []byte(attempt)) == nil
```

Беріть PBKDF2, коли треба лише stdlib і FIPS-сумісність; тягніться до `argon2id`
як до дефолту для нової системи, де memory-hardness - сильніша гарантія.

## Звіт виконання

```text
$ go test ./...
ok  	goloop.one/handbook/006-auth	0.314s

$ go run .
A. password hash and verify (auth PBKDF2):
   stored hash: pbkdf2-sha256$600000$tx8dy2JXhGg... (87 bytes, no plaintext)
   verify correct password: true
   verify wrong password:   false
B. access token issue and verify (auth + jwt), key public id:
   token (JWT): eyJhbGciOiJIUzI1NiIsInR5cCI6IkpX...
   verified subject: id=42 email=ada@example.com roles=[admin]
   tampered token rejected: true
   key public id: x4t8tumhjw
C. signed session cookie (session):
   set cookie session (v1.eyJpZCI6ImVlYzM0MDgxY...)
   loaded session: subject=42 theme=dark
D. token expiry (auth WithTTL + WithClock):
   valid 10s in:  true
   expired at 60s: true
E. refresh token (auth NewRefreshToken / ParseRefreshToken):
   client token: 8ab00e6c4ab95267... (id.secret)
   stored: id=8ab00e6c... hash=86fb21aba9ac... (no secret)
   verify presented secret: true
   verify wrong secret:     false
F. argon2id password hashing (auth.PasswordHasher, memory-hard):
   stored hash: $argon2id$v=19$m=65536,t... (memory-hard)
   verify correct password: true
   verify wrong password:   false
```

Хеш не тримає plaintext, і невірний пароль не проходить; JWT повертає свого
суб'єкта, а один змінений символ відхиляється; сесія переживає круговий рейс
через cookie; короткоживучий токен дійсний на 10с і відхилений на 60с;
refresh-токен перевіряє свій секрет, тоді як зберігається лише його хеш; а
`argon2id.New()` хешує через той самий інтерфейс `PasswordHasher`.

## Що ви дізналися

- `auth.NewPBKDF2` хешує й перевіряє паролі; `PasswordHasher` - інтерфейс, тож
  `argon2id.New()` (memory-hard) підставляється так само й рекомендований як
  дефолт для нової системи; дайджест несе власні алгоритм,
  вартість і сіль, тож `Verify` більше нічого не треба.
- `auth.TokenManager` видає й перевіряє підписані токени (JWT); `Subject` несе
  id, email, ролі й скоупи. Захищайте маршрути його middleware `Protect` (див.
  главу про весь стек).
- `auth.WithTTL` обмежує час життя токена, а `WithClock` робить прострочення
  тестованим; довший `auth.NewRefreshToken` карбує нові, тоді як ви зберігаєте
  лише його хеш.
- `session` тримає підписану cookie-сесію через `Save`/`Load`; `key` карбує
  короткі однозначні публічні id.
- Ніщо з цього не зберігає секрет, який можна злити: хеші, не паролі; підписи,
  не довіру.

Далі: читабельні URL з будь-якого тексту.

---

[« Мовні моделі](05-ai.uk.md) · [Зміст](../main.uk.md) · [Слаги й регістри »](07-slug.uk.md)
