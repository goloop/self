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

## Приклад D - зберіть усі помилки одразу

Падіння на першому поганому полі змушує користувача виправити одне, надіслати
знову й впасти на наступному. Валідатор форми має повідомити *всі* проблеми за
один прохід. Поверніть мапу поля в повідомлення; порожня мапа означає, що все
дійсне:

```go
func validate(f SignupForm) map[string]string {
	problems := map[string]string{}
	if norm.Clean(f.Name) == "" {
		problems["name"] = "a name is required"
	}
	if !is.Email(f.Email) {
		problems["email"] = "must be a valid email address"
	}
	if f.Phone != "" && !is.Phone(f.Phone) {
		problems["phone"] = "must be a valid phone number, or empty"
	}
	return problems
}
```

Зверніть увагу знову на поділ праці: `norm.Clean` перед перевіркою на порожнечу
(щоб таб не був іменем), а read-only предикати `is` для точних перевірок.

## Приклад E - спершу почистіть, потім перевірте

Деякі значення приходять форматованими для людей, але валідувати їх треба в
канонічній формі. Спершу нормалізуйте через `norm`, тоді валідуйте через `is`.
Номер картки очищується від пробілів і перевіряється алгоритмом Луна; IBAN
переводиться у верхній регістр і перевіряється контрольною сумою:

```go
card, _ := norm.BankCard("4539 1488 0343 6467") // "4539148803436467"
is.BankCard(card)                                // true  (валідний Луна)
is.BankCard("4539148803436460")                  // false (одна невірна цифра)

iban, _ := norm.IBAN("de89 3704 0044 0532 0130 00") // "DE89370400440532013000"
is.IBAN(iban)                                        // true (валідна контрольна сума)
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
D. collect all field errors at once:
   name   -> a name is required
   email  -> must be a valid email address
   phone  -> must be a valid phone number, or empty
E. clean then check (canonical form, then validate):
   card "4539148803436467" -> is.BankCard=true (Luhn)
   card with a wrong digit -> is.BankCard=false
   iban "DE89370400440532013000" -> is.IBAN=true (checksum)
```

Форма 1 прийшла з невидимим пробілом в імені, email-ом у змішаному регістрі з
пробілами й форматованим телефоном; вона вийшла чистою, згорнутою і в E.164.
Ім'я форми 3 - самотній таб, що чиститься в порожнечу, тож її відхилено. У D
одна форма з трьома поганими полями повідомила всі три одразу. У E картка й IBAN
прийшли з пробілами й змішаним регістром, були нормалізовані, а тоді пройшли свої
перевірки Луна й контрольної суми, тоді як одна невірна цифра впала. Ніщо
недовірене не прослизнуло, і ніщо просто неохайне не викинуто.

## Що ви дізналися

- Розділяйте роботу: **валідуйте** те, що має бути точним (`is`), **чистьте** те,
  що має бути поблажливим (`norm`).
- `norm.Clean` - безпечна one-call чистка; `norm.EmailFold` дає case-insensitive
  ключ-ідентичність; `norm.E164` тощо повертають `(значення, ok)`.
- Предикати `is` (`Email`, `URL`, `UUID`, `Numeric`, `BankCard`, `IBAN`, ...) -
  це read-only «так/ні» для параметра чи значення конфігу.
- Toolkit `norm` (`DigitsOnly`, `AlnumOnly`, `Keep`, `Remove`) надає форму
  значенню через класи символів.
- Валідуйте кожне поле й збирайте помилки за один прохід, щоб форма повідомила
  всі свої проблеми одразу; для форматованих значень - спершу `norm`, тоді `is`.

Ви дійшли кінця Частини I. Частина II переходить до збереження цих чистих даних у
PostgreSQL і запитів до мовної моделі про них.

---

[« JSON HTTP API](02-http-json-api.uk.md) · [Зміст](../main.uk.md) · [PostgreSQL »](04-postgresql.uk.md)
