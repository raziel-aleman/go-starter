package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"github.com/raziel-aleman/go-starter/internal/database"
	"github.com/raziel-aleman/go-starter/internal/session"
	"github.com/raziel-aleman/go-starter/internal/store"
)

type Server struct {
	port int
	db   database.Service
	sm   *session.SessionManager
}

func NewServer() *http.Server {
	port, _ := strconv.Atoi(os.Getenv("PORT"))

	// Initialize the session store (using in-memory for this example)
	store := store.NewInMemorySessionStore()

	// Configure session manager parameters
	sessionManager := session.NewSessionManager(
		store,
		"GOSESSID",     // Name of the session cookie
		30*time.Minute, // Idle expiration: session expires after 30 minutes of inactivity
		24*time.Hour,   // Absolute expiration: session expires after 24 hours regardless of activity
	)

	NewServer := &Server{
		port: port,
		db:   database.New(),
		sm:   sessionManager,
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}
