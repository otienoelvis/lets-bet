package tracing

import (
	"context"
	"database/sql"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// DBTracer provides tracing utilities for database operations
type DBTracer struct {
	tracer trace.Tracer
}

// NewDBTracer creates a new database tracer
func NewDBTracer() *DBTracer {
	return &DBTracer{
		tracer: otel.Tracer("database"),
	}
}

// TraceQuery traces a database query execution
func (dt *DBTracer) TraceQuery(ctx context.Context, query string, args []any) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		attribute.String("db.system", "postgresql"),
		attribute.String("db.statement", query),
		attribute.Int("db.args_count", len(args)),
	}

	return dt.tracer.Start(ctx, "database.query", trace.WithAttributes(attrs...))
}

// TraceExec traces a database exec operation
func (dt *DBTracer) TraceExec(ctx context.Context, query string, args []any) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		attribute.String("db.system", "postgresql"),
		attribute.String("db.statement", query),
		attribute.Int("db.args_count", len(args)),
		attribute.String("db.operation", "exec"),
	}

	return dt.tracer.Start(ctx, "database.exec", trace.WithAttributes(attrs...))
}

// TraceTransaction traces a database transaction
func (dt *DBTracer) TraceTransaction(ctx context.Context, operation string) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", operation),
	}

	return dt.tracer.Start(ctx, "database.transaction", trace.WithAttributes(attrs...))
}

// TracedDB wraps sql.DB with tracing
type TracedDB struct {
	*sql.DB
	tracer *DBTracer
}

// NewTracedDB creates a new traced database wrapper
func NewTracedDB(db *sql.DB) *TracedDB {
	return &TracedDB{
		DB:     db,
		tracer: NewDBTracer(),
	}
}

// ExecContext traces ExecContext calls
func (tdb *TracedDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	ctx, span := tdb.tracer.TraceExec(ctx, query, args)
	defer span.End()

	result, err := tdb.DB.ExecContext(ctx, query, args...)
	if err != nil {
		span.SetAttributes(attribute.Bool("error", true))
		span.SetAttributes(attribute.String("error.message", err.Error()))
	}

	return result, err
}

// QueryContext traces QueryContext calls
func (tdb *TracedDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	ctx, span := tdb.tracer.TraceQuery(ctx, query, args)
	defer span.End()

	rows, err := tdb.DB.QueryContext(ctx, query, args...)
	if err != nil {
		span.SetAttributes(attribute.Bool("error", true))
		span.SetAttributes(attribute.String("error.message", err.Error()))
	}

	return rows, err
}

// QueryRowContext traces QueryRowContext calls
func (tdb *TracedDB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	ctx, span := tdb.tracer.TraceQuery(ctx, query, args)
	defer span.End()

	return tdb.DB.QueryRowContext(ctx, query, args...)
}

// BeginTx traces transaction begin
func (tdb *TracedDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	ctx, span := tdb.tracer.TraceTransaction(ctx, "begin")
	defer span.End()

	tx, err := tdb.DB.BeginTx(ctx, opts)
	if err != nil {
		span.SetAttributes(attribute.Bool("error", true))
		span.SetAttributes(attribute.String("error.message", err.Error()))
	}

	return tx, err
}

// TracedTx wraps sql.Tx with tracing
type TracedTx struct {
	*sql.Tx
	tracer *DBTracer
}

// NewTracedTx creates a new traced transaction wrapper
func NewTracedTx(tx *sql.Tx) *TracedTx {
	return &TracedTx{
		Tx:     tx,
		tracer: NewDBTracer(),
	}
}

// ExecContext traces transaction ExecContext calls
func (ttx *TracedTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	ctx, span := ttx.tracer.TraceExec(ctx, query, args)
	defer span.End()

	result, err := ttx.Tx.ExecContext(ctx, query, args...)
	if err != nil {
		span.SetAttributes(attribute.Bool("error", true))
		span.SetAttributes(attribute.String("error.message", err.Error()))
	}

	return result, err
}

// QueryContext traces transaction QueryContext calls
func (ttx *TracedTx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	ctx, span := ttx.tracer.TraceQuery(ctx, query, args)
	defer span.End()

	rows, err := ttx.Tx.QueryContext(ctx, query, args...)
	if err != nil {
		span.SetAttributes(attribute.Bool("error", true))
		span.SetAttributes(attribute.String("error.message", err.Error()))
	}

	return rows, err
}

// QueryRowContext traces transaction QueryRowContext calls
func (ttx *TracedTx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	ctx, span := ttx.tracer.TraceQuery(ctx, query, args)
	defer span.End()

	return ttx.Tx.QueryRowContext(ctx, query, args...)
}

// Commit traces transaction commit
func (ttx *TracedTx) Commit() error {
	_, span := ttx.tracer.TraceTransaction(context.Background(), "commit")
	defer span.End()

	err := ttx.Tx.Commit()
	if err != nil {
		span.SetAttributes(attribute.Bool("error", true))
		span.SetAttributes(attribute.String("error.message", err.Error()))
	}

	return err
}

// Rollback traces transaction rollback
func (ttx *TracedTx) Rollback() error {
	_, span := ttx.tracer.TraceTransaction(context.Background(), "rollback")
	defer span.End()

	err := ttx.Tx.Rollback()
	if err != nil {
		span.SetAttributes(attribute.Bool("error", true))
		span.SetAttributes(attribute.String("error.message", err.Error()))
	}

	return err
}
