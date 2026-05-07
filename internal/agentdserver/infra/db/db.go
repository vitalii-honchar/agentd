package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

const sqliteDriver = "sqlite"

var (
	ErrNameRequired     = errors.New("db name is required")
	ErrNameHasSeparator = errors.New("db name must not contain path separators")
	ErrNotOpen          = errors.New("db is not open")
)

//go:embed migrations
var embeddedMigrations embed.FS

var PragmasSettings = map[string]string{
	"journal_mode": "WAL",
	"synchronous":  "FULL",
	"busy_timeout": "5000",
	"foreign_keys": "ON",
	"temp_store":   "MEMORY",
	"cache_size":   "-2000",
}

var PragmasRuntime = map[string]string{
	"journal_mode":       "WAL",
	"synchronous":        "NORMAL",
	"busy_timeout":       "5000",
	"foreign_keys":       "ON",
	"temp_store":         "MEMORY",
	"cache_size":         "-4000",
	"wal_autocheckpoint": "1000",
}

type Config struct {
	Path         string
	MaxOpenConns int
	Pragmas      map[string]string
}

type Option func(*DB)

func WithMigrations(fsys fs.FS) Option {
	return func(d *DB) {
		if fsys != nil {
			d.migrations = fsys
		}
	}
}

type DB struct {
	*sql.DB

	name       string
	cfg        Config
	migrations fs.FS
}

func New(name string, cfg Config, opts ...Option) (*DB, error) {
	if strings.TrimSpace(name) == "" {
		return nil, ErrNameRequired
	}
	if strings.ContainsAny(name, `/\`) {
		return nil, fmt.Errorf("%w: %q", ErrNameHasSeparator, name)
	}
	if strings.TrimSpace(cfg.Path) == "" {
		return nil, fmt.Errorf("db path is required")
	}
	if cfg.MaxOpenConns < 1 {
		return nil, fmt.Errorf("max open conns must be at least 1")
	}

	handle, err := open(cfg.Path, cfg.Pragmas)
	if err != nil {
		return nil, err
	}
	handle.SetMaxOpenConns(cfg.MaxOpenConns)
	handle.SetMaxIdleConns(cfg.MaxOpenConns)

	database := &DB{
		DB:         handle,
		name:       name,
		cfg:        cfg,
		migrations: embeddedMigrations,
	}
	for _, opt := range opts {
		opt(database)
	}

	return database, nil
}

func (d *DB) Name() string {
	return d.name
}

func (d *DB) Start(ctx context.Context) error {
	if d.DB == nil {
		return ErrNotOpen
	}

	return ApplyMigrations(ctx, d.DB, d.migrations, "migrations/"+d.name)
}

func (d *DB) Stop(_ context.Context) error {
	if d == nil || d.DB == nil {
		return nil
	}
	if err := d.Close(); err != nil {
		return fmt.Errorf("close db %q: %w", d.name, err)
	}

	return nil
}

func (d *DB) Ping(ctx context.Context) error {
	if d.DB == nil {
		return ErrNotOpen
	}
	if err := d.PingContext(ctx); err != nil {
		return fmt.Errorf("ping db %q: %w", d.name, err)
	}

	return nil
}

func ApplyMigrations(ctx context.Context, handle *sql.DB, fsys fs.FS, dir string) error {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return fmt.Errorf("read migrations %s: %w", dir, err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		body, err := fs.ReadFile(fsys, dir+"/"+name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if err := applyScript(ctx, handle, string(body)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}

	return nil
}

func open(path string, pragmas map[string]string) (*sql.DB, error) {
	if err := ensureParentDir(path); err != nil {
		return nil, err
	}

	values := url.Values{}
	for name, value := range pragmas {
		values.Add("_pragma", fmt.Sprintf("%s(%s)", name, value))
	}

	dsn := "file:" + path
	if encoded := values.Encode(); encoded != "" {
		dsn += "?" + encoded
	}

	handle, err := sql.Open(sqliteDriver, dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	if err := handle.PingContext(context.Background()); err != nil {
		_ = handle.Close()

		return nil, fmt.Errorf("ping sqlite %s: %w", path, err)
	}

	return handle, nil
}

func ensureParentDir(path string) error {
	if strings.HasPrefix(path, ":memory:") || strings.Contains(path, "mode=memory") {
		return nil
	}

	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create db directory %s: %w", dir, err)
	}

	return nil
}

func applyScript(ctx context.Context, handle *sql.DB, script string) error {
	tx, err := handle.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration tx: %w", err)
	}
	if _, err := tx.ExecContext(ctx, script); err != nil {
		_ = tx.Rollback()
		if isDuplicateColumnMigration(err) {
			return nil
		}

		return fmt.Errorf("exec migration: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration tx: %w", err)
	}

	return nil
}

func isDuplicateColumnMigration(err error) bool {
	return strings.Contains(err.Error(), "duplicate column name:")
}
