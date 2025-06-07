package auth

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/raziel-aleman/go-starter/internal/database"
	"github.com/raziel-aleman/go-starter/internal/session"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Username string `json:"username"`
	Password []byte `json:"-"`
}

func Register(
	dbService database.Service,
	user User,
) (int64, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(user.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return 0, fmt.Errorf("error hashing user password while registering: %v", err)
	}

	result, err := dbService.RegisterUser(user.Username, hashedPassword)
	if err != nil {
		return 0, fmt.Errorf("error registering user: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("error retreiving inserted row id: %v", err)
	}

	return id, nil
}

func VerifyCredentials(
	dbService database.Service,
	user User,
) error {
	var passwordInDB []byte

	passwordInDB, err := dbService.VerifyCredentials(user.Username)
	if err != nil {
		return fmt.Errorf("invalid username: %w", err)
	}

	err = bcrypt.CompareHashAndPassword(
		passwordInDB,
		user.Password,
	)
	if err != nil {
		return fmt.Errorf("invalid password: %w", err)
	}

	return nil
}

func Login(
	r *http.Request,
	srw *session.SessionResponseWriter,
	user User,
) error {
	session := session.GetSession(r)
	if session == nil {
		return fmt.Errorf("session not found")
	}

	newSession, err := srw.Manager.Migrate(session)
	if err != nil {
		return fmt.Errorf("failed to migrate session: %w", err)
	}

	newSession.Put("username", user.Username)

	srw.Session = newSession

	return nil
}

func Logout(
	r *http.Request,
	srw *session.SessionResponseWriter,
) error {
	session := session.GetSession(r)
	if session == nil {
		// No session to destroy, or already destroyed
		return fmt.Errorf("no active session to log out from")
	}

	// Destroy the session in the store
	if err := srw.Manager.Store.Destroy(session.ID); err != nil {
		return fmt.Errorf("error destroying session %s: %v", session.ID, err)
	}

	srw.SessionDestroyed = true
	srw.Session = nil

	return nil
}

func AuthMiddleware(dbservice database.Service, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := session.GetSession(r)

		username := session.Get("username").(string)
		if username == "guest" {
			http.Error(w, "Unauthenticated", http.StatusForbidden)
			return
		}

		err := dbservice.UserExists(username)
		if err == sql.ErrNoRows {
			http.Error(w, "Unauthenticated", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
