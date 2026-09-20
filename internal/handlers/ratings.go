package handlers

import (
	"database/sql"
	"errors"
	"net/http"

	"anime-wor/internal/auth"
	"anime-wor/internal/db"
)

// GET /api/anime/{id}/rating  — средний рейтинг и свой голос
func GetRating(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var avg sql.NullFloat64
	var count int
	_ = db.DB.QueryRow(`SELECT AVG(score), COUNT(*) FROM ratings WHERE anime_id = ?`, id).
		Scan(&avg, &count)

	myScore := 0
	if u := auth.GetUser(r); u != nil {
		_ = db.DB.QueryRow(`SELECT score FROM ratings WHERE user_id = ? AND anime_id = ?`, u.ID, id).
			Scan(&myScore)
	}

	writeJSON(w, 200, map[string]any{
		"avg":   avg.Float64,
		"count": count,
		"my":    myScore,
	})
}

// POST /api/anime/{id}/rating  { score: 1-10 }
func SetRating(w http.ResponseWriter, r *http.Request) {
	u := auth.UserFromCtx(r)
	id := r.PathValue("id")

	var body struct {
		Score int `json:"score"`
	}
	if err := decode(r, &body); err != nil || body.Score < 1 || body.Score > 10 {
		writeErr(w, 400, "Оценка должна быть от 1 до 10")
		return
	}

	_, err := db.DB.Exec(`
		INSERT INTO ratings (user_id, anime_id, score) VALUES (?, ?, ?)
		ON CONFLICT(user_id, anime_id) DO UPDATE SET score = excluded.score, rated_at = strftime('%s','now')
	`, u.ID, id, body.Score)
	if err != nil {
		writeErr(w, 500, "Ошибка сохранения")
		return
	}

	var avg sql.NullFloat64
	var count int
	_ = db.DB.QueryRow(`SELECT AVG(score), COUNT(*) FROM ratings WHERE anime_id = ?`, id).Scan(&avg, &count)
	writeJSON(w, 200, map[string]any{"avg": avg.Float64, "count": count, "my": body.Score})
}

var _ = errors.Is
