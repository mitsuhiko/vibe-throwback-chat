package db

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type DB struct {
	readDB   *sql.DB
	writeDB  *sql.DB
	readDBX  *sqlx.DB
	writeDBX *sqlx.DB
}

// ReadDB returns the read-only database connection
func (db *DB) ReadDB() *sql.DB {
	return db.readDB
}

// WriteDB returns the write database connection
func (db *DB) WriteDB() *sql.DB {
	return db.writeDB
}

// ReadDBX returns the read-only sqlx database connection
func (db *DB) ReadDBX() *sqlx.DB {
	return db.readDBX
}

// WriteDBX returns the write sqlx database connection
func (db *DB) WriteDBX() *sqlx.DB {
	return db.writeDBX
}

func New(dbPath string) (*DB, error) {
	// WAL mode DSN with optimizations
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)"+
		"&_pragma=busy_timeout(5000)"+
		"&_pragma=foreign_keys(ON)"+
		"&_pragma=synchronous(NORMAL)", dbPath)

	// Read connection pool
	readDB, err := sql.Open("sqlite", dsn+"&_txlock=deferred")
	if err != nil {
		return nil, fmt.Errorf("failed to open read database: %w", err)
	}

	// Configure read pool for concurrent reads
	readDB.SetMaxOpenConns(runtime.NumCPU() * 2)
	readDB.SetMaxIdleConns(runtime.NumCPU())

	// Write connection pool
	writeDB, err := sql.Open("sqlite", dsn+"&_txlock=immediate")
	if err != nil {
		readDB.Close()
		return nil, fmt.Errorf("failed to open write database: %w", err)
	}

	// Configure write pool for single writer
	writeDB.SetMaxOpenConns(1)
	writeDB.SetMaxIdleConns(1)

	db := &DB{
		readDB:   readDB,
		writeDB:  writeDB,
		readDBX:  sqlx.NewDb(readDB, "sqlite3"),
		writeDBX: sqlx.NewDb(writeDB, "sqlite3"),
	}

	// Run migrations using write connection
	if err := db.RunMigrations(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

func (db *DB) Close() error {
	var err1, err2 error
	if db.readDB != nil {
		err1 = db.readDB.Close()
	}
	if db.writeDB != nil {
		err2 = db.writeDB.Close()
	}
	if err1 != nil {
		return err1
	}
	return err2
}

func (db *DB) RunMigrations() error {
	// Ensure migrations table exists
	if _, err := db.writeDB.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			filename TEXT NOT NULL UNIQUE,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get applied migrations
	applied := make(map[string]bool)
	rows, err := db.readDB.Query("SELECT filename FROM migrations")
	if err != nil {
		return fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var filename string
		if err := rows.Scan(&filename); err != nil {
			return err
		}
		applied[filename] = true
	}

	// Read embedded migration files
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migration files: %w", err)
	}

	// Sort files to ensure order
	var filenames []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".sql") {
			filenames = append(filenames, entry.Name())
		}
	}
	sort.Strings(filenames)

	for _, filename := range filenames {
		if applied[filename] {
			continue
		}

		log.Printf("Running migration: %s", filename)

		content, err := migrationFiles.ReadFile(filepath.Join("migrations", filename))
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		// Split by semicolon and execute each statement
		statements := strings.Split(string(content), ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}

			if _, err := db.writeDB.Exec(stmt); err != nil {
				return fmt.Errorf("failed to execute migration %s: %w", filename, err)
			}
		}

		// Mark as applied
		if _, err := db.writeDB.Exec("INSERT INTO migrations (filename) VALUES (?)", filename); err != nil {
			return fmt.Errorf("failed to mark migration as applied: %w", err)
		}
	}

	return nil
}
