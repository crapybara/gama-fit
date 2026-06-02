package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"gama-fit/database"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		w.Write([]byte("<p class='text-red-500'>Username and password are required</p>"))
		return
	}

	if len(username) > 50 || len(password) > 72 {
		w.Write([]byte("<p class='text-red-500'>Username or password is too long</p>"))
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	var userID int
	err = database.DB.QueryRow("INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING id", username, string(hashedPassword)).Scan(&userID)
	if err != nil {
		w.Write([]byte("<p class='text-red-500'>Username already exists</p>"))
		return
	}

	// Initialize user stats and macros
	_, _ = database.DB.Exec("INSERT INTO user_stats (user_id) VALUES ($1)", userID)
	_, _ = database.DB.Exec("INSERT INTO user_macros_final (user_id, calories, protein, carbs, fats) VALUES ($1, 2500, 200, 300, 70)", userID)

	w.Header().Set("HX-Redirect", "/login.html")
	w.Write([]byte("<p class='text-green-500'>Registration successful! Redirecting...</p>"))
}

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	rememberMe := r.FormValue("remember") == "on"

	if len(username) > 50 || len(password) > 72 {
		w.Write([]byte("<p class='text-red-500'>Invalid username or password</p>"))
		return
	}

	var id int
	var hashedPassword string
	err := database.DB.QueryRow("SELECT id, password_hash FROM users WHERE username = $1", username).Scan(&id, &hashedPassword)
	if err != nil {
		w.Write([]byte("<p class='text-red-500'>Invalid username or password</p>"))
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		w.Write([]byte("<p class='text-red-500'>Invalid username or password</p>"))
		return
	}

	sessionID := uuid.New().String()
	expiresAt := time.Now().Add(24 * time.Hour)
	if rememberMe {
		expiresAt = time.Now().Add(30 * 24 * time.Hour)
	}

	_, err = database.DB.Exec("INSERT INTO sessions (id, user_id, expires_at) VALUES ($1, $2, $3)", sessionID, id, expiresAt)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Expires:  expiresAt,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true if using HTTPS
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)

	w.Header().Set("HX-Redirect", "/index.html")
	w.Write([]byte("<p class='text-green-500'>Login successful! Redirecting...</p>"))
}

func GetUserID(r *http.Request) (int, error) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return 0, err
	}

	var userID int
	var expiresAt time.Time
	err = database.DB.QueryRow("SELECT user_id, expires_at FROM sessions WHERE id = $1", cookie.Value).Scan(&userID, &expiresAt)
	if err != nil {
		return 0, err
	}

	if time.Now().After(expiresAt) {
		_, _ = database.DB.Exec("DELETE FROM sessions WHERE id = $1", cookie.Value)
		return 0, sql.ErrNoRows
	}

	return userID, nil
}

func HandleCurrentUser(w http.ResponseWriter, r *http.Request) {
	userID, _ := GetUserID(r)

	var username string
	if err := database.DB.QueryRow("SELECT username FROM users WHERE id = $1", userID).Scan(&username); err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"username": username})
}

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := GetUserID(r)
		if err != nil {
			if r.Header.Get("HX-Request") == "true" {
				w.Header().Set("HX-Redirect", "/login.html")
				return
			}
			http.Redirect(w, r, "/login.html", http.StatusSeeOther)
			return
		}
		// Pass userID in a way that handlers can use it
		// For simplicity, we can just call the next handler and let it call GetUserID again,
		// but that's inefficient. Context is better.
		next(w, r)
	}
}
