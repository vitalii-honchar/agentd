package db

import (
	"context"
	"errors"
	"testing"
	"testing/fstest"
)

func TestNewRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		name string
		cfg  Config
		want error
	}{
		"missing name": {
			cfg:  Config{Path: memoryPath(t), MaxOpenConns: 1},
			want: ErrNameRequired,
		},
		"name separator": {
			name: "runtime/agent",
			cfg:  Config{Path: memoryPath(t), MaxOpenConns: 1},
			want: ErrNameHasSeparator,
		},
		"missing path": {
			name: "settings",
			cfg:  Config{MaxOpenConns: 1},
		},
		"bad conns": {
			name: "settings",
			cfg:  Config{Path: memoryPath(t)},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			database, err := New(tt.name, tt.cfg)
			if database != nil {
				_ = database.Stop(context.Background())
			}
			if err == nil {
				t.Fatal("New returned nil error")
			}
			if tt.want != nil && !errors.Is(err, tt.want) {
				t.Fatalf("New error: got %v want %v", err, tt.want)
			}
		})
	}
}

func TestStartAppliesMigrationsInOrder(t *testing.T) {
	t.Parallel()

	migrations := fstest.MapFS{
		"migrations/settings/002_insert.sql": {
			Data: []byte("INSERT INTO order_check(id, value) VALUES (1, 'ok');"),
		},
		"migrations/settings/001_init.sql": {
			Data: []byte("CREATE TABLE order_check(id INTEGER PRIMARY KEY, value TEXT NOT NULL);"),
		},
	}

	database, err := New(
		"settings",
		Config{Path: memoryPath(t), MaxOpenConns: 1, Pragmas: PragmasSettings},
		WithMigrations(migrations),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() {
		if err := database.Stop(context.Background()); err != nil {
			t.Fatalf("Stop: %v", err)
		}
	}()

	if err := database.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	var value string
	if err := database.QueryRowContext(
		context.Background(),
		"SELECT value FROM order_check WHERE id = 1",
	).Scan(&value); err != nil {
		t.Fatalf("query migrated row: %v", err)
	}
	if value != "ok" {
		t.Fatalf("value: got %q want ok", value)
	}
}

func TestPingRequiresOpenDatabase(t *testing.T) {
	t.Parallel()

	database := &DB{}
	if err := database.Ping(context.Background()); !errors.Is(err, ErrNotOpen) {
		t.Fatalf("Ping error: got %v want %v", err, ErrNotOpen)
	}
}

func memoryPath(t *testing.T) string {
	t.Helper()

	return ":memory:"
}
