package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"anime-wor/internal/auth"
	"anime-wor/internal/db"
	"anime-wor/internal/models"
)

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func decode(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// POST /api/users/register
func Register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decode(r, &body); err != nil {
		writeErr(w, 400, "Некорректный запрос")
		return
	}
	body.Username = strings.TrimSpace(body.Username)
	body.Email = strings.TrimSpace(body.Email)

	if len(body.Username) < 3 || len(body.Username) > 24 {
		writeErr(w, 400, "Логин должен быть от 3 до 24 символов")
		return
	}
	if len(body.Password) < 6 {
		writeErr(w, 400, "Пароль минимум 6 символов")
		return
	}
	if !strings.Contains(body.Email, "@") || !strings.Contains(body.Email, ".") {
		writeErr(w, 400, "Некорректный email")
		return
	}

	var existing int64
	err := db.DB.QueryRow(
		`SELECT id FROM users WHERE username = ? OR email = ?`,
		body.Username, body.Email,
	).Scan(&existing)
	if err == nil {
		writeErr(w, 409, "Логин или email уже заняты")
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(body.Password), 10)
	res, err := db.DB.Exec(
		`INSERT INTO users (username, email, password, role) VALUES (?, ?, ?, 'user')`,
		body.Username, body.Email, string(hash),
	)
	if err != nil {
		writeErr(w, 500, "Ошибка создания пользователя")
		return
	}
	id, _ := res.LastInsertId()
	u := models.User{ID: id, Username: body.Username, Email: body.Email, Role: "user"}

	token, _ := auth.Sign(u)
	auth.SetCookie(w, token)
	writeJSON(w, 200, map[string]any{"user": u})
}

// POST /api/users/login
func Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decode(r, &body); err != nil {
		writeErr(w, 400, "Некорректный запрос")
		return
	}

	var u models.User
	var hash string
	err := db.DB.QueryRow(
		`SELECT id, username, email, password, role FROM users WHERE username = ? OR email = ?`,
		body.Username, body.Username,
	).Scan(&u.ID, &u.Username, &u.Email, &hash, &u.Role)

	if errors.Is(err, sql.ErrNoRows) {
		writeErr(w, 401, "Неверный логин или пароль")
		return
	}
	if err != nil {
		writeErr(w, 500, "Ошибка сервера")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(body.Password)) != nil {
		writeErr(w, 401, "Неверный логин или пароль")
		return
	}

	token, _ := auth.Sign(u)
	auth.SetCookie(w, token)
	writeJSON(w, 200, map[string]any{"user": u})
}

// POST /api/users/logout
func Logout(w http.ResponseWriter, r *http.Request) {
	auth.ClearCookie(w)
	writeJSON(w, 200, map[string]any{"ok": true})
}

// GET /api/users/me
func Me(w http.ResponseWriter, r *http.Request) {
	u := auth.GetUser(r)
	if u == nil {
		writeErr(w, 401, "Не авторизован")
		return
	}
	writeJSON(w, 200, map[string]any{"user": u})
}

// GET /api/users/favorites
func GetFavorites(w http.ResponseWriter, r *http.Request) {
	u := auth.UserFromCtx(r)
	rows, err := db.DB.Query(`SELECT anime_id FROM favorites WHERE user_id = ?`, u.ID)
	if err != nil {
		writeErr(w, 500, "Ошибка БД")
		return
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		_ = rows.Scan(&id)
		ids = append(ids, id)
	}
	writeJSON(w, 200, map[string]any{"favorites": ids})
}

// POST /api/users/favorites/{animeId}
func ToggleFavorite(w http.ResponseWriter, r *http.Request) {
	u := auth.UserFromCtx(r)
	animeID := r.PathValue("animeId")

	var exists int
	err := db.DB.QueryRow(
		`SELECT 1 FROM favorites WHERE user_id = ? AND anime_id = ?`,
		u.ID, animeID,
	).Scan(&exists)

	if err == nil {
		_, _ = db.DB.Exec(
			`DELETE FROM favorites WHERE user_id = ? AND anime_id = ?`,
			u.ID, animeID,
		)
		writeJSON(w, 200, map[string]any{"favorite": false})
		return
	}
	_, _ = db.DB.Exec(
		`INSERT INTO favorites (user_id, anime_id) VALUES (?, ?)`, u.ID, animeID,
	)
	writeJSON(w, 200, map[string]any{"favorite": true})
}

// POST /api/users/history
func AddHistory(w http.ResponseWriter, r *http.Request) {
	u := auth.UserFromCtx(r)
	var body struct {
		AnimeID string `json:"animeId"`
		Episode int    `json:"episode"`
		Voice   string `json:"voice"`
	}
	if err := decode(r, &body); err != nil || body.AnimeID == "" {
		writeErr(w, 400, "Нет данных")
		return
	}
	_, _ = db.DB.Exec(
		`INSERT INTO watch_history (user_id, anime_id, episode, voice) VALUES (?, ?, ?, ?)`,
		u.ID, body.AnimeID, body.Episode, body.Voice,
	)
	writeJSON(w, 200, map[string]any{"ok": true})
}

// GET /api/users/history
func GetHistory(w http.ResponseWriter, r *http.Request) {
	u := auth.UserFromCtx(r)
	rows, err := db.DB.Query(`
		SELECT wh.anime_id, wh.episode, wh.voice, wh.watched_at, a.title, a.poster
		FROM watch_history wh
		LEFT JOIN anime a ON a.id = wh.anime_id
		WHERE wh.user_id = ?
		ORDER BY wh.watched_at DESC LIMIT 50
	`, u.ID)
	if err != nil {
		writeErr(w, 500, "Ошибка БД")
		return
	}
	defer rows.Close()

	type item struct {
		AnimeID   string `json:"anime_id"`
		Episode   int    `json:"episode"`
		Voice     string `json:"voice"`
		WatchedAt int64  `json:"watched_at"`
		Title     string `json:"title"`
		Poster    string `json:"poster"`
	}
	list := []item{}
	for rows.Next() {
		var it item
		_ = rows.Scan(&it.AnimeID, &it.Episode, &it.Voice, &it.WatchedAt, &it.Title, &it.Poster)
		list = append(list, it)
	}
	writeJSON(w, 200, map[string]any{"history": list})
}
