[« JSON HTTP API](02-http-json-api.uk.md) · [Зміст](../main.uk.md) · [PostgreSQL »](04-postgresql.uk.md)

---

# 03. Валідація й чистка вводу

**Задача.** Форма реєстрації приходить повна людського безладу: зайві пробіли,
змішаний регістр, невидимий символ, вставлений із чату, номер телефону з тире й
дужками. Відхилити те, що справді невалідне, і тихо канонізувати те, що просто
неохайне - перш ніж будь-що з цього дійде до бази.

**Модулі.** [`is`](https://github.com/goloop/is) відповідає на *«чи це
валідне?»* предикатами лише для читання; [`norm`](https://github.com/goloop/norm)
відповідає на *«зроби це канонічною формою»* чисткою й згортанням.

**Рецепт.** [`recipes/003-validate-and-clean`](../recipes/003-validate-and-clean/)

## Дві половини

Обробка вводу - це дві роботи, і їх змішування породжує баги. Одні поля мусять
бути *точно* валідними; інші мають бути *поблажливими*. `is` - половина лише для
читання: він ніколи не змінює значення, лише повідомляє. `norm` - половина
запису: він повертає почищене значення і, де доречно, чи воно валідне.

## Приклад A - форма реєстрації

```go
name := norm.Clean(f.Name) // прибрати невидимі/контрол-байти, стиснути пробіли, тримити
if name == "" {
	return Account{}, ErrEmptyName
}
email, ok := norm.EmailFold(f.Email) // валідація + весь адрес у нижній регістр
if !ok {
	return Account{}, ErrInvalidEmail
}
phone := ""
if f.Phone != "" {
	if e164, ok := norm.E164(f.Phone); ok {
		phone = e164 // +14155550132
	}
}
```

`norm.Clean` прибирає невидимі й контрольні символи (зокрема вставлений
невидимий пробіл), стискає пробіли й тримить. `norm.EmailFold` валідує й
переводить у нижній регістр увесь адрес, тож `Ada@Example.COM` і
`ada@example.com` - той самий акаунт. Необов'язкове поле чистять, коли воно є, і
відкидають, коли ні.

## Приклад B - галерея `is`

Коли вам потрібне лише «так/ні» - на параметрі запиту чи значенні конфігу -
беріть `is` напряму. Він ніколи не змінює ввід:

```go
is.Email("a@b.co")   // true
is.URL("https://goloop.one") // true
is.UUID("9b1deb4d-...")      // true
is.Numeric("12345")  // true
is.Numeric("12a3")   // false - літери не числові
```

## Приклад C - toolkit `norm`

Окрім типізованих нормалізаторів, `norm` має toolkit символів на основі класів
(`Letters`, `Digits`, `Punct`, `Invisible`, ...), щоб надати форму будь-якому
значенню:

```go
norm.DigitsOnly("+1 (415) 555-0132") // "14155550132"
norm.AlnumOnly("user.name_42!")      // "username42"
norm.Remove("a,b;c.d", norm.Punct)   // "abcd"
norm.Keep("Go1.24!", norm.Letters)   // "Go"
```

## Звіт виконання

```text
$ go test ./...
ok  	goloop.one/handbook/003-validate-and-clean	0.004s

$ go run .
A. clean and validate a signup form:
   form 1: name="Ada Lovelace" email="ada@example.com" phone="+14155550132"
   form 2: name="Grace Hopper" email="grace@example.com" phone=""
   form 3: rejected: a name is required
   form 4: rejected: a valid email is required
B. is predicates (read-only, never change the value):
   is.Email(a@b.co)               = true
   is.URL(https://goloop.one)     = true
   is.UUID(9b1deb...)             = true
   is.Numeric(12345)              = true
   is.Numeric(12a3)               = false
C. norm toolkit (shape a value to a canonical form):
   DigitsOnly("+1 (415) 555-0132")  = "14155550132"
   AlnumOnly("user.name_42!")   = "username42"
   Remove("a,b;c.d", Punct) = "abcd"
   Keep("Go1.24!", Letters) = "Go"
```

Форма 1 прийшла з невидимим пробілом в імені, email-ом у змішаному регістрі з
пробілами й форматованим телефоном; вона вийшла чистою, згорнутою і в E.164.
Ім'я форми 3 - самотній таб, що чиститься в порожнечу, тож її відхилено. Ніщо
недовірене не прослизнуло, і ніщо просто неохайне не викинуто.

## Що ви дізналися

- Розділяйте роботу: **валідуйте** те, що має бути точним (`is`), **чистьте** те,
  що має бути поблажливим (`norm`).
- `norm.Clean` - безпечна one-call чистка; `norm.EmailFold` дає case-insensitive
  ключ-ідентичність; `norm.E164` тощо повертають `(значення, ok)`.
- Предикати `is` (`Email`, `URL`, `UUID`, `Numeric`, ...) - це read-only
  «так/ні» для параметра чи значення конфігу.
- Toolkit `norm` (`DigitsOnly`, `AlnumOnly`, `Keep`, `Remove`) надає форму
  значенню через класи символів.

Ви дійшли кінця Частини I. Частина II переходить до збереження цих чистих даних у
PostgreSQL і запитів до мовної моделі про них.

---

[« JSON HTTP API](02-http-json-api.uk.md) · [Зміст](../main.uk.md) · [PostgreSQL »](04-postgresql.uk.md)
