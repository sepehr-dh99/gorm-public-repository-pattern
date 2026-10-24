package publicRepository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type record struct {
	Key    string `gorm:"column:record_key;primaryKey"`
	Name   string
	Active bool
}

// A scripted database/sql connection exercises GORM's real query callbacks
// without a database server, CGO, or a test-only database dependency.
type sqlStep struct {
	contains string
	absent   string
	args     []driver.Value
	columns  []string
	rows     [][]driver.Value
	affected int64
	err      error
}

type scriptedConn struct {
	t     *testing.T
	steps []sqlStep
}

func (c *scriptedConn) next(query string, args []driver.NamedValue) sqlStep {
	c.t.Helper()
	if len(c.steps) == 0 {
		c.t.Fatalf("unexpected SQL: %s", query)
	}
	s := c.steps[0]
	c.steps = c.steps[1:]
	if !strings.Contains(query, s.contains) || (s.absent != "" && strings.Contains(query, s.absent)) {
		c.t.Fatalf("SQL %q: want %q, forbid %q", query, s.contains, s.absent)
	}
	if s.args != nil {
		actual := make([]driver.Value, len(args))
		for i := range args {
			actual[i] = args[i].Value
		}
		if !reflect.DeepEqual(actual, s.args) {
			c.t.Fatalf("args: got %#v, want %#v", actual, s.args)
		}
	}
	return s
}

func (c *scriptedConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	s := c.next(query, args)
	if s.err != nil {
		return nil, s.err
	}
	return &scriptedRows{columns: s.columns, rows: s.rows}, nil
}

func (c *scriptedConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	s := c.next(query, args)
	return driver.RowsAffected(s.affected), s.err
}
func (c *scriptedConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected Prepare")
}
func (c *scriptedConn) Close() error              { return nil }
func (c *scriptedConn) Begin() (driver.Tx, error) { return nil, errors.New("unexpected Begin") }

type scriptedRows struct {
	columns []string
	rows    [][]driver.Value
}

func (r *scriptedRows) Columns() []string { return r.columns }
func (r *scriptedRows) Close() error      { return nil }
func (r *scriptedRows) Next(dest []driver.Value) error {
	if len(r.rows) == 0 {
		return io.EOF
	}
	copy(dest, r.rows[0])
	r.rows = r.rows[1:]
	return nil
}

type testConnector struct{ conn *scriptedConn }

func (c testConnector) Connect(context.Context) (driver.Conn, error) { return c.conn, nil }
func (c testConnector) Driver() driver.Driver                        { return testDriver{} }

type testDriver struct{}

func (testDriver) Open(string) (driver.Conn, error) { return nil, errors.New("use connector") }

type testDialect struct{ pool *sql.DB }

func (testDialect) Name() string { return "scripted" }
func (d testDialect) Initialize(db *gorm.DB) error {
	db.ConnPool = d.pool
	callbacks.RegisterDefaultCallbacks(db, &callbacks.Config{})
	return nil
}
func (testDialect) Migrator(*gorm.DB) gorm.Migrator { return nil }
func (testDialect) DataTypeOf(*schema.Field) string { return "text" }
func (testDialect) DefaultValueOf(*schema.Field) clause.Expression {
	return clause.Expr{SQL: "DEFAULT"}
}
func (testDialect) BindVarTo(w clause.Writer, _ *gorm.Statement, _ interface{}) { w.WriteByte('?') }
func (testDialect) QuoteTo(w clause.Writer, name string) {
	w.WriteByte('"')
	w.WriteString(strings.ReplaceAll(name, ".", "\".\""))
	w.WriteByte('"')
}
func (testDialect) Explain(query string, vars ...interface{}) string {
	return logger.ExplainSQL(query, nil, "'", vars...)
}

func testDB(t *testing.T, steps ...sqlStep) *gorm.DB {
	t.Helper()
	c := &scriptedConn{t: t, steps: steps}
	pool := sql.OpenDB(testConnector{conn: c})
	t.Cleanup(func() {
		pool.Close()
		if len(c.steps) != 0 {
			t.Errorf("%d SQL steps were not executed", len(c.steps))
		}
	})
	db, err := gorm.Open(testDialect{pool: pool}, &gorm.Config{SkipDefaultTransaction: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestUintPointerCompatibility(t *testing.T) {
	type numeric struct{ ID uint }
	id := uint(7)
	repo := NewMainRepository[numeric](testDB(t, sqlStep{contains: `"numerics"."id" = ?`, args: []driver.Value{int64(7), int64(1)}, columns: []string{"id"}, rows: [][]driver.Value{{int64(7)}}}))
	got, err := repo.FindById(&id)
	if err != nil || got.ID != id {
		t.Fatalf("got %#v, %v", got, err)
	}
}

func TestExistUsesBoundedSelect(t *testing.T) {
	for _, exists := range []bool{false, true} {
		rows := [][]driver.Value{}
		if exists {
			rows = append(rows, []driver.Value{int64(1)})
		}
		repo := NewMainRepository[record](testDB(t, sqlStep{contains: `SELECT 1 FROM "records" WHERE active = ? LIMIT ?`, absent: "count(", args: []driver.Value{true, int64(1)}, columns: []string{"1"}, rows: rows}))
		got, err := repo.Exist(func(q *gorm.DB) *gorm.DB { return q.Where("active = ?", true).Preload("Unused") })
		if err != nil || got != exists {
			t.Fatalf("got %v, %v", got, err)
		}
	}
}

func TestExistPreservesSoftDeleteScope(t *testing.T) {
	type archivedRecord struct {
		ID        uint
		DeletedAt gorm.DeletedAt
	}
	repo := NewMainRepository[archivedRecord](testDB(t, sqlStep{
		contains: `WHERE "archived_records"."deleted_at" IS NULL LIMIT ?`,
		columns:  []string{"1"}, rows: [][]driver.Value{{int64(1)}},
	}))
	exists, err := repo.Exist()
	if err != nil || !exists {
		t.Fatalf("got %v, %v", exists, err)
	}
}

// Both assignments must compile: neither signatures nor required methods may
// change for consumers implementing the original repository contract.
type originalRepository[T any] interface {
	FindById(*uint, ...QueryBuilder) (*T, error)
	FindAll(...QueryBuilder) (*[]T, error)
	Create(*T, ...QueryBuilder) (*T, error)
	CreateMany(*[]T, ...QueryBuilder) (*[]T, error)
	Update(*T, *uint, ...QueryBuilder) (*T, error)
	Delete(*T, ...QueryBuilder) error
	FindAllPaginated(*Pagination, ...QueryBuilder) (*[]T, int64, int64, error)
	Count(...QueryBuilder) (*int64, error)
	Exist(...QueryBuilder) (bool, error)
	QueryBuilder(...QueryBuilder) *gorm.DB
	WithTransaction(func(IMainRepository[T]) error) error
}

var _ IMainRepository[record] = (originalRepository[record])(nil)
var _ originalRepository[record] = (IMainRepository[record])(nil)

func TestExistPreservesDatabaseError(t *testing.T) {
	want := errors.New("database unavailable")
	repo := NewMainRepository[record](testDB(t, sqlStep{contains: "SELECT 1", err: want}))
	exists, err := repo.Exist()
	if exists || !errors.Is(err, want) {
		t.Fatalf("got %v, %v", exists, err)
	}
}
