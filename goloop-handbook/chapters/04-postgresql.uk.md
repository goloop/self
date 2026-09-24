[« Валідація й чистка](03-validate-and-clean.uk.md) · [Зміст](../main.uk.md) · [Мовні моделі »](05-ai.uk.md)

---

# 04. Типобезпечний PostgreSQL з міграціями

**Задача.** Розвивати схему бази з часом і робити до неї запити з Go без ручного
scan-коду й без ризику одрукуватися в назві колонки. Коли ви перейменовуєте
колонку, код, що її використовував, має перестати компілюватися, а не падати в
рантаймі.

**Модуль.** [`pgc`](https://github.com/goloop/pgc) - це компілятор SQL у Go для
PostgreSQL із власними міграціями. У нього дві роботи, які запускають з
командного рядка:

- `pgc migrate` застосовує `.sql`-файли з `migrations/` по порядку;
- `pgc generate` перетворює запити з `queries/` на типізовані Go-методи.

**Рецепт.** [`recipes/004-postgresql`](../recipes/004-postgresql/)

## Ідея

Ви пишете руками два види SQL: міграції, що формують схему, і запити, що читають
і пишуть її. `pgc` читає обидва. З міграції він знає таблицю; із запиту генерує
Go-метод із типізованими параметрами й типізованим результатом. Згенерований
пакет залежить лише від `database/sql`, тож програма нижче не імпортує жодного
пакета GoLoop: вона просто використовує код, який написав `pgc`.

Схема - це один файл міграції, `migrations/001_notes.sql`:

```sql
CREATE TABLE notes (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title      text NOT NULL,
    body       text NOT NULL DEFAULT '',
    tags       text[] NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now()
);
```

Запити - це анотований SQL, `queries/notes.sql`. Тег `:one` / `:many` каже
`pgc`, чи метод повертає один рядок, чи зріз:

```sql
-- name: CreateNote :one
INSERT INTO notes (title, body, tags) VALUES ($1, $2, $3) RETURNING *;

-- name: SearchNotes :many
SELECT * FROM notes WHERE title ILIKE '%' || @query || '%' ORDER BY id DESC;
```

`pgc generate` перетворює це на типізовану структуру `Note` й типізовані методи:

```go
type Note struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Tags      []string  `json:"tags"` // Postgres text[] стає []string
	CreatedAt time.Time `json:"created_at"`
}

func (q *Queries) CreateNote(ctx context.Context, title, body string, tags []string) (Note, error)
func (q *Queries) SearchNotes(ctx context.Context, query string) ([]Note, error)
```

## Приклад A - запис

`CreateNote` виконує `INSERT ... RETURNING *` і повертає повністю типізований
`Note`, зокрема згенеровані базою `id` і `created_at`:

```go
q := store.New(db) // обгортає *sql.DB, безпечно ділити
n, err := q.CreateNote(ctx, "Reading list", "Books to read this month.",
	[]string{"personal", "books"})
```

## Приклад B - читання

Читання одного рядка й кількох - це звичайні виклики методів, що повертають
типізовані рядки:

```go
got, _ := q.NoteByID(ctx, n.ID)   // Note
list, _ := q.ListNotes(ctx, 10)   // []Note
total, _ := q.CountNotes(ctx)     // int64
```

## Приклад C - пошук, з масивом

`SearchNotes` бере параметр, а колонка `text[]` повертається як Go `[]string`
без жодного scan-коду з вашого боку:

```go
found, _ := q.SearchNotes(ctx, "list") // []Note, кожен зі своїм .Tags []string
```

## Приклад D - транзакція

Деякі записи мають статися разом або не статися зовсім. `pgc` генерує метод
`WithTx`: `q.WithTx(tx)` повертає `Queries`, прив'язаний до `*sql.Tx`, тож ті
самі типізовані методи тепер працюють усередині транзакції. І `*sql.DB`, і
`*sql.Tx` задовольняють малий інтерфейс `DBTX`, який потрібен згенерованому
коду, тож окремого набору методів вчити не треба:

```go
tx, _ := db.BeginTx(ctx, nil)
qtx := q.WithTx(tx) // ті самі методи, тепер усередині tx

if _, err := qtx.CreateNote(ctx, "Groceries", "Milk, bread.", []string{"home"}); err != nil {
	_ = tx.Rollback() // одна помилка скасовує весь пакет
	return err
}
if _, err := qtx.CreateNote(ctx, "Standup", "Blockers and plan.", []string{"work"}); err != nil {
	_ = tx.Rollback()
	return err
}
_ = tx.Commit() // обидві нотатки лягають разом
```

Відкотіть замість коміту - і записи зникають: `CreateNote` усередині відкоченої
`tx` ніколи не стає видимим, тож `CountNotes` після цього незмінний.

## Приклад E - міграції, що стережуть себе самі

Міграція виконується проти найважливішої бази, тож `pgc migrate` перевіряє всю
теку й звіряє її з тим, що застосував раніше, ще до того, як щось запустити.
Три помилки, які він зупиняє, - з реальної сесії з цим рецептом. Друга
міграція, що комітить сама, відхиляється ще до старту, бо `pgc` і так виконує
кожен файл у транзакції разом із записом про те, що його виконано:

```text
$ pgc migrate
pgc: database book@127.0.0.1:5440/book (from PGC_DATABASE_URL)
pgc: migrate: 002_pinned.sql, statement 2: COMMIT - each file already runs in a transaction of its own; remove it, or mark the file "-- pgc: no-transaction" to run it statement by statement
```

Редагування вже застосованого файлу не змінює історію мовчки.
`pgc migrate status` називає файл і завершується з ненульовим кодом, тож крок CI
може на цьому зупинитись, а `pgc migrate` нічого не застосовує, доки файл не
повернуто:

```text
$ pgc migrate status
pgc: database book@127.0.0.1:5440/book (from PGC_DATABASE_URL)
changed  001_notes.sql                            2026-09-24 09:36
pending  002_pinned.sql
pgc: 1 migration(s) need attention: changed or missing files, or no-transaction files left unfinished
```

Поверніть `001_notes.sql` як був - і `002_pinned.sql` застосовується. Нова
колонка змінює те, що повертає `SELECT *`, і `pgc check` - який нічого не
пише - каже, що закомічений Go більше не відповідає:

```text
$ pgc migrate
pgc: database book@127.0.0.1:5440/book (from PGC_DATABASE_URL)
applied 002_pinned.sql

$ pgc check
pgc: the generated package differs from what the queries produce; run pgc generate:
  out of date: internal/store/models.go
  out of date: internal/store/notes.sql.go
```

Це і є весь цикл у CI: `pgc migrate status` стереже історію, `pgc check` -
згенерований код, і обидва голосно падають, замість того щоб пустити в прод
схему, що розійшлась. (Кожна команда, що торкається бази, спершу друкує, якої
саме, без пароля, - `DATABASE_URL`, призначений для іншого, помітно ще до того,
як щось станеться.)

## Звіт виконання

Міграції застосовано, протестовано проти справжнього PostgreSQL, потім запущено:

```text
$ pgc migrate
pgc: database book@127.0.0.1:5440/book (from PGC_DATABASE_URL)
applied 001_notes.sql

$ go test ./...            # проти БД; коли її немає - охайно пропускає
ok  	goloop.one/handbook/004-postgresql	0.010s

$ go run .
A. write (CreateNote -> typed Note):
   inserted id=1 title="Reading list" tags=[personal books]
B. read (NoteByID, ListNotes, CountNotes):
   NoteByID(1) -> "Reading list"
   - #2 "Release checklist" [work]
   - #1 "Reading list" [personal books]
   total = 2
C. search (SearchNotes, parameter + text[] -> []string):
   match #2 "Release checklist" tags=[work]
   match #1 "Reading list" tags=[personal books]
D. transaction (WithTx, commit then rollback):
   committed 2 notes: 2 -> 4
   after rollback: 4 -> 4 (unchanged)
```

Вставка повернула типізований `Note` зі згенерованим id; читання повернули
`Note` і `[]Note`; пошук збігся з обома заголовками, що містять «list»; а `tags`
рухалися між Postgres і Go як `[]string` наскрізь. Закомічена транзакція додала
дві нотатки одразу (2 -> 4); відкочена лишила лічильник недоторканим (4 -> 4),
доводячи атомарність записів.

## Нотатка про nullable JSON

Один тип поля варто виділити. Nullable-колонка `json` чи `jsonb` генерується як
`*json.RawMessage` (у дефолтному pointer-режимі) - вказівник, тож SQL `NULL`
сканується як nil-вказівник, а значення - як байти. Це важливо, бо різниця між
«немає значення» і «значення = JSON `null`» реальна в базі, і згенерований код
її зберігає: тягніться до `COALESCE` у запиті лише коли справді хочете злити ці
два. Колонка `NOT NULL` json лишається звичайним `json.RawMessage`.

## Що ви дізналися

- У `pgc` дві роботи з командного рядка: `pgc migrate` (застосувати схему) і
  `pgc generate` (скомпілювати запити в типізований Go).
- Анотуйте запит `-- name: X :one` чи `:many`; `pgc` пише типізований метод і
  структуру. `INSERT ... RETURNING *` стає типізованим записом.
- Згенерований пакет використовує лише `database/sql`; Postgres `text[]` стає Go
  `[]string`.
- `q.WithTx(tx)` запускає ті самі типізовані методи всередині `*sql.Tx`: коміт -
  і пакет лягає разом, відкіт - і його наче не було.
- Оскільки колонки типізовані в Go, перейменування однієї в міграції ламає
  збірку, а не прод.
- Nullable-колонка `json`/`jsonb` - це `*json.RawMessage`, тож SQL `NULL` і JSON
  `null` лишаються різними.
- `pgc migrate` відхиляє файл, що комітить сам, і застосований файл, який
  редагували; `pgc migrate status` і `pgc check` падають у CI, коли історія чи
  згенерований код розійшлися.

Частина II продовжується запитом до мовної моделі про ці дані.

---

[« Валідація й чистка](03-validate-and-clean.uk.md) · [Зміст](../main.uk.md) · [Мовні моделі »](05-ai.uk.md)
