package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"
	_ "github.com/mattn/go-sqlite3"
)

// Service represents a service that interacts with a database.
type Service interface {
	// Health returns a map of health status information.
	// The keys and values in the map are service-specific.
	Health() map[string]string

	// Close terminates the database connection.
	// It returns an error if the connection cannot be closed.
	Close() error

	GetClient() *sql.DB

	GetUser(string) (int64, error)
}

type service struct {
	db *sql.DB
}

var (
	// db url parameters for WAL mode, timeout for concurrent writes, and for foreing key checking
	dburl      = os.Getenv("BLUEPRINT_DB_URL") + "?_journal=WAL&_timeout=5000&_fk=true"
	dbInstance *service
)

func New() Service {
	// Reuse Connection
	if dbInstance != nil {
		return dbInstance
	}

	db, err := sql.Open("sqlite3", dburl)
	if err != nil {
		// This will not be a connection error, but a DSN parse error or
		// another initialization error.
		log.Fatal(err)
	}

	err = Init(db)
	if err != nil {
		log.Fatal(err)
	}

	dbInstance = &service{
		db: db,
	}
	return dbInstance
}

func Init(db *sql.DB) error {
	// Users table initialization query if it does not exist
	const createUsersTable string = `CREATE TABLE IF NOT EXISTS users (
		id INTEGER NOT NULL PRIMARY KEY,
		username TEXT NOT NULL UNIQUE,
		password BLOB NOT NULL
	);`

	// Execute initialization query
	if _, err := db.Exec(createUsersTable); err != nil {
		return fmt.Errorf("error creating User table: %v", err)
	}

	// Sessions table initializaiton query if it does not exist
	const createSessionsTable string = `CREATE TABLE IF NOT EXISTS sessions (
		id INTEGER NOT NULL PRIMARY KEY,
		sessionId TEXT NOT NULL,
		createdAt TEXT NOT NULL,
		lastActive TEXT NOT NULL,
		data BLOB NOT NULL
	);`

	// Execute initialization query
	if _, err := db.Exec(createSessionsTable); err != nil {
		return fmt.Errorf("error creating Sessions table: %v", err)
	}

	return nil
}

// Health checks the health of the database connection by pinging the database.
// It returns a map with keys indicating various health statistics.
func (s *service) Health() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	stats := make(map[string]string)

	// Ping the database
	err := s.db.PingContext(ctx)
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		log.Fatalf("db down: %v", err) // Log the error and terminate the program
		return stats
	}

	// Database is up, add more statistics
	stats["status"] = "up"
	stats["message"] = "It's healthy"

	// Get database stats (like open connections, in use, idle, etc.)
	dbStats := s.db.Stats()
	stats["open_connections"] = strconv.Itoa(dbStats.OpenConnections)
	stats["in_use"] = strconv.Itoa(dbStats.InUse)
	stats["idle"] = strconv.Itoa(dbStats.Idle)
	stats["wait_count"] = strconv.FormatInt(dbStats.WaitCount, 10)
	stats["wait_duration"] = dbStats.WaitDuration.String()
	stats["max_idle_closed"] = strconv.FormatInt(dbStats.MaxIdleClosed, 10)
	stats["max_lifetime_closed"] = strconv.FormatInt(dbStats.MaxLifetimeClosed, 10)

	// Evaluate stats to provide a health message
	if dbStats.OpenConnections > 40 { // Assuming 50 is the max for this example
		stats["message"] = "The database is experiencing heavy load."
	}

	if dbStats.WaitCount > 1000 {
		stats["message"] = "The database has a high number of wait events, indicating potential bottlenecks."
	}

	if dbStats.MaxIdleClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many idle connections are being closed, consider revising the connection pool settings."
	}

	if dbStats.MaxLifetimeClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many connections are being closed due to max lifetime, consider increasing max lifetime or revising the connection usage pattern."
	}

	return stats
}

// Close closes the database connection.
// It logs a message indicating the disconnection from the specific database.
// If the connection is successfully closed, it returns nil.
// If an error occurs while closing the connection, it returns the error.
func (s *service) Close() error {
	log.Printf("Disconnected from database: %s", dburl)
	return s.db.Close()
}

func (s *service) GetClient() *sql.DB {
	return s.db
}

func (s *service) GetUser(username string) (int64, error) {
	var id int64
	err := s.db.QueryRow(
		"SELECT id From users (username) VALUES (?)",
		username,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("user not found in db: %v", err)
	}

	return id, nil
}
