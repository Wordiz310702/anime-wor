package handlers

import (
	"database/sql"
	"net/http"

	"anime-wor/internal/db"
)

// GET /api/schedule  — все записи
func ListSchedule(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query(`
		SELECT s.id, s.anime_id, s.weekday, s.episode, s.air_time, a.title, a.poster
		FROM schedule s
		LEFT JOIN anime a ON a.id = s.anime_id
		ORDER BY s.weekday, s.air_time
	`)
	if err != nil {
		writeErr(w, 500, "Ошибка БД")
		return
	}
	defer rows.Close()

	type item struct {
		ID      int64  `json:"id"`
		AnimeID string `json:"anime_id"`
		Weekday int    `json:"weekday"`
		Episode int    `json:"episode"`
		AirTime string `json:"air_time"`
		Title   string `json:"title"`
		Poster  string `json:"poster"`
	}
	list := []item{}
	for rows.Next() {
		var it item
		var ep sql.NullInt64
		var t, ti, po sql.NullString
		_ = rows.Scan(&it.ID, &it.AnimeID, &it.Weekday, &ep, &t, &ti, &po)
		it.Episode = int(ep.Int64)
		it.AirTime = t.String
		it.Title = ti.String
		it.Poster = po.String
		list = append(list, it)
	}
	writeJSON(w, 200, map[string]any{"schedule": list})
}

// POST /api/schedule  (admin) { animeId, weekday, episode, airTime }
func CreateSchedule(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AnimeID string `json:"animeId"`
		Weekday int    `json:"weekday"`
		Episode int    `json:"episode"`
		AirTime string `json:"airTime"`
	}
	if err := decode(r, &body); err != nil || body.AnimeID == "" || body.Weekday < 1 || body.Weekday > 7 {
		writeErr(w, 400, "Некорректный запрос")
		return
	}
	_, err := db.DB.Exec(
		`INSERT INTO schedule (anime_id, weekday, episode, air_time) VALUES (?, ?, ?, ?)`,
		body.AnimeID, body.Weekday, body.Episode, body.AirTime,
	)
	if err != nil {
		writeErr(w, 500, "Ошибка сохранения")
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

// DELETE /api/schedule/{id}  (admin)
func DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_, _ = db.DB.Exec(`DELETE FROM schedule WHERE id = ?`, id)
	writeJSON(w, 200, map[string]any{"ok": true})
}
