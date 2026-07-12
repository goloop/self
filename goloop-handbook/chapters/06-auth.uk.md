[« Мовні моделі](05-ai.uk.md) · [Зміст](../main.uk.md) · [Слаги й регістри »](07-slug.uk.md)

---

# 06. Сесії, токени й паролі

**Задача.** Безпечно впускати користувачів. Зберігати пароль так, щоб витік бази
не роздав plaintext; видавати підписаний токен, який API-клієнт присилає з
кожним запитом; і тримати браузерну сесію в підписаній cookie. Нічого з цього не
винаходьте самі.

**Модулі.** [`auth`](https://github.com/goloop/auth) хешує паролі й видає токени
(справжній JWT через [`jwt`](https://github.com/goloop/jwt)),
[`session`](https://github.com/goloop/session) веде підписану cookie-сесію, а
[`key`](https://github.com/goloop/key) карбує короткі публічні id.

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

## Звіт виконання

```text
$ go test ./...
ok  	goloop.one/handbook/006-auth	0.297s

$ go run .
A. password hash and verify (auth PBKDF2):
   stored hash: pbkdf2-sha256$600000$4sLgsWRnhEQ... (87 bytes, no plaintext)
   verify correct password: true
   verify wrong password:   false
B. access token issue and verify (auth + jwt), key public id:
   token (JWT): eyJhbGciOiJIUzI1NiIsInR5cCI6IkpX...
   verified subject: id=42 email=ada@example.com roles=[admin]
   tampered token rejected: true
   key public id: qj23hctskd
C. signed session cookie (session):
   set cookie session (v1.eyJpZCI6ImJjYTBkODQyY...)
   loaded session: subject=42 theme=dark
```

Хеш не тримає plaintext, і невірний пароль не проходить; JWT повертає свого
суб'єкта, а один змінений символ відхиляється; сесія переживає круговий рейс
через cookie.

## Що ви дізналися

- `auth.NewPBKDF2` хешує й перевіряє паролі; дайджест несе власні алгоритм,
  вартість і сіль, тож `Verify` більше нічого не треба.
- `auth.TokenManager` видає й перевіряє підписані токени (JWT); `Subject` несе
  id, email, ролі й скоупи. Захищайте маршрути його middleware `Protect` (див.
  главу про весь стек).
- `session` тримає підписану cookie-сесію через `Save`/`Load`; `key` карбує
  короткі однозначні публічні id.
- Ніщо з цього не зберігає секрет, який можна злити: хеші, не паролі; підписи,
  не довіру.

Далі: читабельні URL з будь-якого тексту.

---

[« Мовні моделі](05-ai.uk.md) · [Зміст](../main.uk.md) · [Слаги й регістри »](07-slug.uk.md)
