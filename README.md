# godb

A relational database engine written from scratch in Go — storage layer, hash
indexing, a hand-rolled SQL parser, and a query executor, with no external
dependencies.

I built this to understand what actually happens between a `SELECT` statement
and the rows it returns. Rather than wrapping an existing engine or a parser
library, every layer here — tokenizer, parser, schema enforcement, indexing,
query execution — is implemented directly, so the whole path from SQL text to
result set is visible and inspectable in about 2,000 lines of Go.

## Scope

This is a teaching engine, not a production one, and the gaps are
intentional rather than oversights:

- **In-memory only.** Data lives in Go maps and is gone when the process
  exits. There is no WAL, no page cache, no disk format.
- **One index type.** `HashIndex` gives O(1) equality lookups and nothing
  else — no range scans, no `ORDER BY` support from an index.
- **A deliberately small SQL dialect.** `CREATE TABLE`, `CREATE INDEX`,
  `INSERT`, `SELECT`, `UPDATE`, `DELETE`. `WHERE` clauses support `AND` but
  not `OR` or parentheses, so the evaluator in `engine/where.go` stays a
  simple linear pass over a condition list rather than a general boolean
  expression tree.
- **No transactions.** Each statement is its own atomic operation; there's
  no `BEGIN`/`COMMIT`, no isolation levels, no rollback.

Each of these is a natural next milestone if you want to extend the project
— see [Extending it](#extending-it) below.

## Running it

```
go run ./cmd/repl
```

```
godb> CREATE TABLE users (id INT PRIMARY KEY, name TEXT, age INT);
table "users" created
godb> INSERT INTO users (id, name, age) VALUES (1, 'ada', 30), (2, 'grace', 40);
2 row(s) inserted
godb> SELECT * FROM users WHERE age > 25;
 id | name  | age
 ---+-------+----
 1  | ada   | 30
 2  | grace | 40
(2 row(s))
godb> UPDATE users SET age = 31 WHERE id = 1;
1 row(s) updated
godb> DELETE FROM users WHERE name = 'grace';
1 row(s) deleted
godb> \dt
  users
godb> \q
```

Run the tests with:

```
go test ./...
```

## Architecture

Four layers, in the order a query passes through them:

```
  SQL text
     |
     v
  parser/    tokenizer + recursive-descent parser -> AST
     |
     v
  engine/    walks the AST, decides how to execute it, talks to storage
     |
     v
  storage/   tables of typed rows, enforces the schema
     |
     v
  index/     hash indexes storage consults for fast equality lookups
```

**`storage/`** owns the actual data. `types.go` defines the three supported
column types (`INT`, `TEXT`, `BOOL`) and `Schema.CheckValue`, which every
insert and update runs through — a table can't end up with a string in an
`INT` column. `table.go` holds rows in a map keyed by an internal,
never-reused ID, plus a slice tracking insertion order so an unordered
`SELECT *` still comes back in a predictable order. A table also owns
whichever hash indexes have been built on it and keeps them in sync on every
write, under the same lock that guards the row data.

**`index/`** is a `HashIndex`: a map from column value to the set of row IDs
holding it. It only knows about `interface{}` values and `int` IDs, which
keeps it storage-agnostic and lets `storage` depend on it without a circular
import. A `PRIMARY KEY` column gets one automatically when its table is
created; `CREATE INDEX ON table (column)` builds one explicitly and
backfills it from existing rows. It's a hash index, not a B-tree, so it
speeds up `col = value` but can't help with `<`, `>`, or `ORDER BY` — those
still walk every row, which is the point: it's a small, concrete example of
why real databases carry more than one index structure.

**`parser/`** is a straightforward hand-written lexer and recursive-descent
parser. `parseStatement` dispatches on the leading keyword, and each
`parseX` method consumes exactly the tokens its grammar rule owns, erroring
the moment something unexpected shows up rather than trying to recover.

**`engine/`** ties it together behind a single `Execute(sql string)` entry
point. `where.go` is the one place anything resembling query planning
happens: if the first `WHERE` condition is an equality test on an indexed
column, it uses the index to narrow the candidate set before falling back to
a linear scan for whatever conditions remain; otherwise it's a full scan.
It's a deliberately simple stand-in for what a cost-based planner does with
many index types and join strategies.

## Extending it

Roughly in order of what they teach, if you want to take this further:

1. **Disk persistence** — an append-only WAL that replays on startup, the
   first thing every real storage engine does before anything fancier.
2. **A B-tree index** alongside the hash index, so range queries and
   `ORDER BY` can use one too.
3. **Page-based storage** — fixed-size pages with a buffer pool, instead of
   one big map, which is what lets a database scale past available RAM.
4. **A larger SQL surface** — `OR`, parenthesized `WHERE` clauses, `JOIN`,
   `ORDER BY`, `LIMIT`, `GROUP BY`.
5. **Transactions** — `BEGIN`/`COMMIT`/`ROLLBACK` with enough locking or
   MVCC for real isolation, instead of the current per-statement atomicity.

## Project layout

```
godb/
  storage/   tables, rows, schema and type enforcement
  index/     hash index
  parser/    lexer, AST, recursive-descent parser
  engine/    query executor (AST -> storage calls)
  cmd/repl/  interactive command-line shell
```
