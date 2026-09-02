[« AI у продакшн-формі](14-ai-production.uk.md) · [Зміст](../main.uk.md)

---

# 15. Помилки, на які фронтенд робить switch

**Задача.** Коли запит падає, JSON API винен клієнту дві речі: повідомлення, яке
прочитає людина, і код, на який діятиме програма. Число, яке шле більшість API, -
поганий код: `1042` нічого не каже в лозі, тесті чи `switch`, а перетворити його
на значення потребує реєстру, який ніхто не тримає в актуальності. Слаг каже це
прямо - `slug_taken`, `invalid_json`, `rate_limit` - і фронтенд робить на нього
switch і шукає перекладене повідомлення за ним, будь-якою мовою, не довіряючи
прозі, яку бекенд випадково надіслав.

**Модулі.** [`resp`](https://github.com/goloop/resp) несе слаг поруч із числовим
кодом і повідомленням; [`qp`](https://github.com/goloop/qp) читає й обмежує
query-параметри; [`mux`](https://github.com/goloop/mux) маршрутизує.

**Рецепт.** [`recipes/015-error-slugs`](../recipes/015-error-slugs/)

## Форма

Кожна помилка, яку шле служба, має ті самі три поля, з одного хелпера, тож
жоден обробник не винаходить свою:

```go
func Fail(w http.ResponseWriter, status int, slug, message string) {
	_ = resp.Error(w, status, message, resp.WithErrorSlug(slug))
}

// Fail(w, 409, "slug_taken", "That slug is already in use") пише:
//   HTTP 409  {"code":409,"error":"slug_taken","message":"That slug is already in use"}
```

`code` - це HTTP-статус, віддзеркалений у тіло; `error` - машинний слаг;
`message` - для людини. Фронтенд читає `error`, ігнорує `message`, крім як для
показу як запасного, і шукає власний переклад за слагом.

## Приклад A - decode-набір, що називає відмову

Читання JSON-тіла має два різні режими відмови, і розрізняти їх важливо: зламане
тіло шле розробника до його серіалізатора, а це хибне місце, коли тіло просто
обрізало на ліміті розміру. `Decode` відповідає правильним слагом на кожен:

```go
func Decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBytes) // ліміт
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			Fail(w, 413, "body_too_large", "Request body exceeds the maximum size")
			return false
		}
		Fail(w, 400, "invalid_json", "Request body is not valid JSON")
		return false
	}
	return true
}
```

Ліміт розміру живе на `Decode` - одній функції, яку кличе кожен JSON-обробник -
тож покриває маршрути, яких ще немає, без вручну підтримуваного списку «тільки
JSON» маршрутів, чий режим відмови (забути додати новий маршрут) тихий. Це
патерн, який реальна служба використала замість `middlewares.MaxBytes` на рівні
роутера, бо її upload-маршрути потребували стелі в тисячі разів більшої і
ставили свою.

## Приклад B - погане query-значення - це помилка, не дефолт

`qp.Int` повертає дефолт на погане значення без нарікань, що правильно для
опційного важеля. Де погане значення має бути помилкою - неправильно набрана
пагінація - `ParseInt` повертає повний `Result`, і його поле `Error` несе
причину, тож помилка отримує слаг замість тихого скидання на сторінку 1:

```go
q := qp.New(r.URL)
pageR := q.ParseInt("page", qp.Default(1), qp.Min(1))
perPageR := q.ParseInt("per_page", qp.Default(20), qp.Min(1), qp.Max(100))
if pageR.Error != nil || perPageR.Error != nil {
	Fail(w, 400, "invalid_query", "page and per_page must be positive integers")
	return
}
```

Межі (`Min`, `Max`) роблять подвійну роботу: валідують і спиняють викликача від
запиту на мільйон рядків.

## Приклад C - каталог

Набір слагів, які відповідає служба, - це контракт, який ділить фронтенд.
Тримайте його в одній таблиці - згенерованій, чи спільній із фронтендом, чи
принаймні записаній:

| Слаг | Статус | i18n-ключ |
|---|---|---|
| `invalid_json` | 400 | `errors.invalid_json` |
| `invalid_query` | 400 | `errors.invalid_query` |
| `slug_required` | 422 | `errors.slug_required` |
| `slug_taken` | 409 | `errors.slug_taken` |
| `body_too_large` | 413 | `errors.body_too_large` |

Слаги - власний словник застосунку, тож `resp` їх не валідує - тримайте їх
фіксованим набором форми `[a-z0-9_]+`. Значення видається як є, тож це не місце
для будь-чого, що надав запит.

## Звіт про виконання

```
$ go run .
A/B. one shape, right slug per failure:
   bad JSON:        400 {"code":400,"error":"invalid_json","message":"Request body is not valid JSON"}
   missing slug:    422 {"code":422,"error":"slug_required","message":"A slug is required"}
   slug taken:      409 {"code":409,"error":"slug_taken","message":"That slug is already in use"}
   created:         200 {"slug":"fresh"}
C. query params, a slug on a bad value:
   good:            200 {"page":2,"per_page":50}
   bad page:        400 {"code":400,"error":"invalid_query","message":"page and per_page must be positive integers"}
D. the catalog the frontend switches on:
   invalid_json   -> 400  errors.invalid_json
   ...
```

## Що ви дізналися

- Слаг (`resp.WithErrorSlug`) - те, на що клієнт робить switch і за чим шукається
  i18n-ключ; число потребує реєстру, що застаріває.
- Один хелпер `Fail` дає кожній помилці ту саму форму `{code, error, message}`.
- Decode-набір із лімітом розміру відповідає `invalid_json` і `body_too_large`
  окремо й садить ліміт туди, де його вже кличе кожен JSON-обробник.
- `qp.ParseInt(...).Error` перетворює погане query-значення на 400 зі слагом, а
  не тихий дефолт.
- Тримайте слаги в одному каталозі - це контракт, який ділить фронтенд.

---

[« AI у продакшн-формі](14-ai-production.uk.md) · [Зміст](../main.uk.md) · [Каталог, який ведуть руками »](16-yaml-catalog.uk.md)
