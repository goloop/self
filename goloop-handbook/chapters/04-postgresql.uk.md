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
SELECT * FROM notes WHERE title ILIKE '%' || $1 || '%' ORDER BY id DESC;
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
func (q *Queries) SearchNotes(ctx context.Context, arg1 string) ([]Note, error)
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
total, _ := q.CountNotes(ctx)     // *int64
```

## Приклад C - пошук, з масивом

`SearchNotes` бере параметр, а колонка `text[]` повертається як Go `[]string`
без жодного scan-коду з вашого боку:

```go
found, _ := q.SearchNotes(ctx, "list") // []Note, кожен зі своїм .Tags []string
```

## Звіт виконання

Міграції застосовано, протестовано проти справжнього PostgreSQL, потім запущено:

```text
$ pgc migrate
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
```

Вставка повернула типізований `Note` зі згенерованим id; читання повернули
`Note` і `[]Note`; пошук збігся з обома заголовками, що містять «list»; а `tags`
рухалися між Postgres і Go як `[]string` наскрізь.

## Що ви дізналися

- У `pgc` дві роботи з командного рядка: `pgc migrate` (застосувати схему) і
  `pgc generate` (скомпілювати запити в типізований Go).
- Анотуйте запит `-- name: X :one` чи `:many`; `pgc` пише типізований метод і
  структуру. `INSERT ... RETURNING *` стає типізованим записом.
- Згенерований пакет використовує лише `database/sql`; Postgres `text[]` стає Go
  `[]string`.
- Оскільки колонки типізовані в Go, перейменування однієї в міграції ламає
  збірку, а не прод.

Частина II продовжується запитом до мовної моделі про ці дані.

---

[« Валідація й чистка](03-validate-and-clean.uk.md) · [Зміст](../main.uk.md) · [Мовні моделі »](05-ai.uk.md)
