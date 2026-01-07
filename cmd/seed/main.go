package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const EnvDatabaseDSN = "DATABASE_DSN"

func main() {
	var (
		dsn      = flag.String("dsn", "", "Database connection string")
		all      = flag.Bool("all", false, "Run all seeders")
		profiles = flag.Bool("profiles", false, "Seed profiles")
		file     = flag.String("file", "", "External seed file (overrides embedded)")
		list     = flag.Bool("list", false, "List available seeders")
	)
	flag.Parse()

	if *list {
		fmt.Println("Available seeders:")
		for _, s := range listSeeders() {
			fmt.Printf("  - %s: %s\n", s.Name(), s.Description())
		}
		return
	}

	if *dsn == "" {
		*dsn = os.Getenv(EnvDatabaseDSN)
	}
	if *dsn == "" {
		log.Fatalf("database connection string required: use -dsn flag or %s env var", EnvDatabaseDSN)
	}

	db, err := sql.Open("pgx", *dsn)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	ctx := context.Background()

	switch {
	case *all:
		if err := runAllSeeders(ctx, db); err != nil {
			log.Fatalf("seeding failed: %v", err)
		}
		fmt.Println("all seeders completed successfully")

	case *profiles:
		if *file != "" {
			if seeder, ok := getSeeder("profiles"); ok {
				seeder.(*ProfileSeeder).SetFile(*file)
			}
		}
		if err := runSeeder(ctx, db, "profiles"); err != nil {
			log.Fatalf("seeding failed: %v", err)
		}
		fmt.Println("profiles seeded successfully")

	default:
		fmt.Println("usage: seed -dsn <connection-string> [-all|-profiles] [-file <path>] [-list]")
		flag.PrintDefaults()
	}
}
