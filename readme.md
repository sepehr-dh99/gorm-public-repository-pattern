# Generic GORM Repository

A generic Go repository library for GORM. Applications provide their database
connection, model structs, driver, and migrations. Requires Go 1.22.1 or later.

## Installation

```sh
go get github.com/sepehr-dh99/gorm-public-repository-pattern
go get gorm.io/driver/postgres
```

## Usage

Save this example as `main.go` in your application. Set `DATABASE_DSN` to your
PostgreSQL connection string, then run `go run .`.

```go
package main

import (
    "fmt"
    "log"
    "os"
    "time"

    repository "github.com/sepehr-dh99/gorm-public-repository-pattern"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

type User struct {
    ID        uint `gorm:"primaryKey"`
    Name      string
    Email     string
    Status    int64
    CreatedAt time.Time
}

func main() {
    db, err := gorm.Open(postgres.Open(os.Getenv("DATABASE_DSN")), &gorm.Config{})
    if err != nil {
        log.Fatal(err)
    }
    if err := db.AutoMigrate(&User{}); err != nil {
        log.Fatal(err)
    }

    users := repository.NewMainRepository[User](db)
    created, err := users.Create(&User{Name: "Alice", Email: "alice@example.com", Status: 1})
    if err != nil {
        log.Fatal(err)
    }
    user, err := users.FindById(&created.ID)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("User:", user)

    page, count, pages, err := users.FindAllPaginated(
        &repository.Pagination{Page: 1, Limit: 20},
        func(q *gorm.DB) *gorm.DB {
            return q.Where("status = ?", 1).Order("created_at DESC, id DESC")
        },
    )
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Users: %v; total: %d; pages: %d\n", *page, count, pages)
}
```

## Compatibility and behavior

The original `IMainRepository[T]` interface and method signatures are preserved.
`FindById` and `Update` take `*uint` IDs. No additional methods are required from
existing implementations or mocks. CRUD methods preserve their previous error
and GORM behavior:

- `Update` uses GORM `Save`, including its insert/upsert behavior, and filters
  updates by the `id` column. It is not an update-only operation.
- `Delete` preserves GORM soft deletes and protection against unfiltered deletes.
- Zero affected rows do not introduce a new repository error. For explicit row
  counts, use `QueryBuilder(...)` and GORM's `RowsAffected`.
- `FindById` preserves GORM's `ErrRecordNotFound` for missing records.

`Exist` checks for a match with `SELECT 1` limited to one row, rather than
counting every matching record. Filters and soft-delete scopes still apply.
Association preloads are skipped; `Count` is available when a count is needed.

## Pagination

Callers should supply non-nil pagination with positive `Page` and `Limit` values.
The original pagination behavior is preserved: query builders apply to both
`Count` and `Find`, including caller-supplied limits and offsets. Returned values
remain data, result count, total pages, and error, in that order.

Count and data are separate queries, so concurrent writes can change results
between them. Use a transaction with suitable database isolation when a shared
snapshot is required. Use deterministic ordering with a unique tie-breaker.

## Transactions

`WithTransaction` commits on success and rolls back on callback errors or
panics. Use the repository passed to the callback for transactional operations:

```go
err := users.WithTransaction(func(txRepo repository.IMainRepository[User]) error {
    _, err := txRepo.Create(&User{Name: "Bob"})
    return err
})
```

## Validation

```sh
go test ./...
go vet ./...
```

Tests exercise GORM query callbacks through a scripted `database/sql` driver.
They need no running database. Production-dialect behavior still requires
integration testing against a disposable database.
