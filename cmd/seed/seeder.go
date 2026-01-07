// Package main provides the seed command for populating the database with
// initial or test data. It supports multiple seeders that can be run
// individually or together within a single transaction.
package main

import (
	"context"
	"database/sql"
	"fmt"
)

// Seeder defines the interface for database seeders.
// Each seeder is responsible for populating a specific domain's data.
type Seeder interface {
	// Name returns the unique identifier for this seeder.
	Name() string

	// Description returns a human-readable description of what this seeder does.
	Description() string

	// Seed executes the seeding logic within the provided transaction.
	// The transaction allows all-or-nothing semantics across multiple seeders.
	Seed(ctx context.Context, tx *sql.Tx) error
}

var seeders = map[string]Seeder{}

// registerSeeder adds a seeder to the global registry.
// Seeders self-register via init() functions.
func registerSeeder(s Seeder) {
	seeders[s.Name()] = s
}

// getSeeder retrieves a seeder by name from the registry.
func getSeeder(name string) (Seeder, bool) {
	s, ok := seeders[name]
	return s, ok
}

// listSeeders returns all registered seeders.
func listSeeders() []Seeder {
	result := make([]Seeder, 0, len(seeders))
	for _, s := range seeders {
		result = append(result, s)
	}
	return result
}

// runSeeder executes a single seeder by name within a transaction.
// Returns an error if the seeder is not found or if seeding fails.
func runSeeder(ctx context.Context, db *sql.DB, name string) error {
	seeder, ok := getSeeder(name)
	if !ok {
		return fmt.Errorf("seeder not found: %s", name)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	if err := seeder.Seed(ctx, tx); err != nil {
		tx.Rollback()
		return fmt.Errorf("seed %s: %w", name, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// runAllSeeders executes all registered seeders within a single transaction.
// If any seeder fails, the entire transaction is rolled back.
func runAllSeeders(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	for name, seeder := range seeders {
		if err := seeder.Seed(ctx, tx); err != nil {
			tx.Rollback()
			return fmt.Errorf("seed %s: %w", name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
