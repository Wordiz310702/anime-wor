package auth

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"anime-wor/internal/db"
	"anime-wor/internal/models"
)

var jwtSecret []byte

func Init() {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		s = "dev_secret_change_me"
	}
	jwtSecret = []byte(s)
}

type ctxKey string

const userCtxKey ctxKey = "user"

func Sign(u models.User) (string, error) {
	claims := jwt.MapClaims{
		"id":       u.ID,
		"username": u.Username,
		"role":     u.Role,
		"exp":      time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret)
}

func Parse(token string) (int64, string, string, error) {
	t, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected method")
		}
		return jwtSecret, nil
	})
	if err != nil || !t.Valid {
		return 0, "", "", errors.New("invalid token")
	}
	m, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return 0, "", "", errors.New("bad claims")
	}
	id := int64(m["id"].(float64))
	username, _ := m["username"].(string)
	role, _ := m["role"].(string)
	return id, username, role, nil
}

// GetUser вытаскивает пользователя из cookie или Authorization
func GetUser(r *http.Request) *models.User {
	token := ""
	if c, err := r.Cookie("token"); err == nil {
		token = c.Value
	} else if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		token = strings.TrimPrefix(h, "Bearer ")
	}
	if token == "" {
		return nil
	}
	id, _, _, err := Parse(token)
	if err != nil {
		return nil
	}
	var u models.User
	err = db.DB.QueryRow(
		`SELECT id, username, email, role FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Role)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return nil
	}
	return &u
}

func SetCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		MaxAge:   7 * 24 * 3600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		// Secure: true, // включить на HTTPS
	})
}

func ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: "token", Value: "", Path: "/", MaxAge: -1,
	})
}

// Middleware
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := GetUser(r)
		if u == nil {
			writeErr(w, http.StatusUnauthorized, "Требуется авторизация")
			return
		}
		ctx := context.WithValue(r.Context(), userCtxKey, u)
		next(w, r.WithContext(ctx))
	}
}

func RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := GetUser(r)
		if u == nil {
			writeErr(w, http.StatusUnauthorized, "Требуется авторизация")
			return
		}
		if u.Role != "admin" {
			writeErr(w, http.StatusForbidden, "Доступ запрещён")
			return
		}
		ctx := context.WithValue(r.Context(), userCtxKey, u)
		next(w, r.WithContext(ctx))
	}
}

func UserFromCtx(r *http.Request) *models.User {
	if u, ok := r.Context().Value(userCtxKey).(*models.User); ok {
		return u
	}
	return nil
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write([]byte(`{"error":"` + msg + `"}`))
}
