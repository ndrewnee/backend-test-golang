package db

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ndrewnee/backend-test-golang/migrations"
)

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		content, err := fs.ReadFile(migrations.FS, entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}

		for _, statement := range splitSQLStatements(string(content)) {
			if _, err := pool.Exec(ctx, statement); err != nil {
				return fmt.Errorf("run migration %s: %w", entry.Name(), err)
			}
		}
	}

	return nil
}

func splitSQLStatements(content string) []string {
	parts := strings.Split(content, ";")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		statement := strings.TrimSpace(part)
		if statement == "" {
			continue
		}
		statements = append(statements, statement+";")
	}
	return statements
}
