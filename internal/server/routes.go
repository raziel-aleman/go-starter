package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/raziel-aleman/go-starter/internal/auth"
	sm "github.com/raziel-aleman/go-starter/internal/session"
)

func (s *Server) RegisterRoutes() http.Handler {
	mux := http.NewServeMux()

	// Register public routes
	mux.HandleFunc("/home", s.HelloWorldHandler)

	mux.HandleFunc("/health", s.HealthHandler)

	mux.HandleFunc("/", s.HomeHandler)

	mux.HandleFunc("/logout", s.LogoutHandler)

	mux.HandleFunc("/debug", s.DebugSessionHandler)

	mux.HandleFunc("/login", s.LoginHandler)

	mux.HandleFunc("/register", s.RegisterHandler)

	// Register private routes with Auth Middleware
	mux.Handle("/protected", auth.AuthMiddleware(s.db, http.HandlerFunc(s.ProtectedHandler)))

	// Wrap the mux with CORS middleware, Sessions middleware
	return s.corsMiddleware(s.sm.SessionMiddleware(mux))
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*") // Replace "*" with specific origins if needed
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
		w.Header().Set("Access-Control-Allow-Credentials", "false") // Set to "true" if credentials are required

		// Handle preflight OPTIONS requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Proceed with the next handler
		next.ServeHTTP(w, r)
	})
}

func (s *Server) HelloWorldHandler(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{"message": "Hello World"}
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(jsonResp); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}

func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	resp, err := json.Marshal(s.db.Health())
	if err != nil {
		http.Error(w, "Failed to marshal health check response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(resp); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}

// homeHandler shows how to interact with the session.
func (s *Server) HomeHandler(w http.ResponseWriter, r *http.Request) {
	session := sm.GetSession(r)
	if session == nil {
		http.Error(w, "Session not found", http.StatusInternalServerError)
		return
	}

	// Example: Get username from session
	username := session.Get("username")
	if username == "" {
		username = "guest"
		session.Put("username", username) // Set a default if not present
	}

	fmt.Fprintf(w, "Welcome! Your session ID is: %s\n", session.ID)
	fmt.Fprintf(w, "User ID from session: %v\n", username)
	fmt.Fprintf(w, "Session CSRF token: %s\n", session.Get("csrf_token"))
	fmt.Fprintf(w, "Session created at: %s\n", session.CreatedAt.Format(time.RFC3339))
	fmt.Fprintf(w, "Session last active: %s\n", session.LastActive.Format(time.RFC3339))

	// Example: Increment a counter in the session
	visits := session.Get("visits")
	if visits == nil {
		visits = 0
	}
	session.Put("visits", visits.(int)+1)
	fmt.Fprintf(w, "You have visited this page %d times in this session.\n", session.Get("visits").(int))
}

// protectedHandler demonstrates a route that requires a logged-in user.
func (s *Server) ProtectedHandler(w http.ResponseWriter, r *http.Request) {
	session := sm.GetSession(r)
	userID := session.Get("username")
	fmt.Fprintf(w, "Welcome, %s! This is a protected area.\n", userID)
}

// logoutHandler destroys the current session.
func (s *Server) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Signal to the SessionResponseWriter that the session has been destroyed.
	// This ensures the session cookie is cleared correctly by the middleware.
	if srw, ok := w.(*sm.SessionResponseWriter); ok {
		err := auth.Logout(r, srw)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		srw.StatusCode = http.StatusSeeOther
		srw.ResponseWriter.Header().Set("Location", "http://localhost:"+strconv.Itoa(s.port)+"/")
	}

	log.Printf("Logged out successfully! Session destroyed.\n")
}

// DebugSessionHandler for inspecting raw session data (for debugging only).
func (s *Server) DebugSessionHandler(w http.ResponseWriter, r *http.Request) {
	session := sm.GetSession(r)
	if session == nil {
		http.Error(w, "No active session.", http.StatusNotFound)
		return
	}

	session.RLock()
	defer session.RUnlock()
	// Encode session data to JSON for easy viewing
	jsonBytes, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		http.Error(w, "Error marshalling session data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonBytes)
}

// loginHandler simulates a user login and updates the session.
func (s *Server) LoginHandler(w http.ResponseWriter, r *http.Request) {
	session := sm.GetSession(r)

	// In a real application, you'd get the user from a the client.
	// For example purposes, we'll just set a dummy user.
	user := auth.User{Username: "user123", Password: []byte("general123")}

	//err := auth.VerifyCredentials(s.db.GetClient(), user)
	err := auth.VerifyCredentials(s.db, user)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if srw, ok := w.(*sm.SessionResponseWriter); ok {
		if session.Get("username") != "guest" {
			log.Printf("%s already logged in", session.Get("username"))
			srw.StatusCode = http.StatusFound
			srw.ResponseWriter.Header().Set("Location", "http://localhost:"+strconv.Itoa(s.port)+"/")
			return
		}
		err := auth.Login(r, srw, user)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		srw.StatusCode = http.StatusSeeOther
		srw.ResponseWriter.Header().Set("Location", "http://localhost:"+strconv.Itoa(s.port)+"/")
	}

	log.Printf("User logged in successfully! Session updated for user: %s\n", user.Username)
}

func (s *Server) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.User{Username: "user123", Password: []byte("general123")}
	_, err := auth.Register(s.db, user)
	if err != nil {
		log.Println(err)
		w.Header().Set("Location", "http://localhost:"+strconv.Itoa(s.port)+"/")
		w.WriteHeader(http.StatusSeeOther)
		return
	}

	session := sm.GetSession(r)
	session.Put("username", user.Username)

	if srw, ok := w.(*sm.SessionResponseWriter); ok {
		srw.Session = session
		err := auth.Login(r, srw, user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		srw.StatusCode = http.StatusSeeOther
		srw.ResponseWriter.Header().Set("Location", "http://localhost:"+strconv.Itoa(s.port)+"/")
	}
}
