package handlers

import (
	"net/http"
	"strings"

	"anime-wor/internal/auth"
	"anime-wor/internal/db"
)

// GET /api/anime/{id}/comments
func ListComments(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rows, err := db.DB.Query(`
		SELECT c.id, c.user_id, u.username, c.text, c.created_at
		FROM comments c
		JOIN users u ON u.id = c.user_id
		WHERE c.anime_id = ? AND c.approved = 1
		ORDER BY c.created_at DESC
		LIMIT 200
	`, id)
	if err != nil {
		writeErr(w, 500, "Ошибка БД")
		return
	}
	defer rows.Close()

	type comment struct {
		ID        int64  `json:"id"`
		UserID    int64  `json:"user_id"`
		Username  string `json:"username"`
		Text      string `json:"text"`
		CreatedAt int64  `json:"created_at"`
	}
	list := []comment{}
	for rows.Next() {
		var c comment
		_ = rows.Scan(&c.ID, &c.UserID, &c.Username, &c.Text, &c.CreatedAt)
		list = append(list, c)
	}
	writeJSON(w, 200, map[string]any{"comments": list})
}

// POST /api/anime/{id}/comments  { text }
func AddComment(w http.ResponseWriter, r *http.Request) {
	u := auth.UserFromCtx(r)
	id := r.PathValue("id")

	var body struct {
		Text string `json:"text"`
	}
	if err := decode(r, &body); err != nil {
		writeErr(w, 400, "Некорректный запрос")
		return
	}
	body.Text = strings.TrimSpace(body.Text)
	if len(body.Text) < 2 || len(body.Text) > 2000 {
		writeErr(w, 400, "Комментарий должен быть от 2 до 2000 символов")
		return
	}

	res, err := db.DB.Exec(
		`INSERT INTO comments (user_id, anime_id, text) VALUES (?, ?, ?)`,
		u.ID, id, body.Text,
	)
	if err != nil {
		writeErr(w, 500, "Ошибка сохранения")
		return
	}
	cid, _ := res.LastInsertId()
	writeJSON(w, 200, map[string]any{
		"comment": map[string]any{
			"id":         cid,
			"user_id":    u.ID,
			"username":   u.Username,
			"text":       body.Text,
			"created_at": 0,
		},
	})
}

// DELETE /api/comments/{id}  — свой комментарий или админ
func DeleteComment(w http.ResponseWriter, r *http.Request) {
	u := auth.UserFromCtx(r)
	cid := r.PathValue("id")

	var ownerID int64
	err := db.DB.QueryRow(`SELECT user_id FROM comments WHERE id = ?`, cid).Scan(&ownerID)
	if err != nil {
		writeErr(w, 404, "Комментарий не найден")
		return
	}
	if u.Role != "admin" && u.ID != ownerID {
		writeErr(w, 403, "Доступ запрещён")
		return
	}
	_, _ = db.DB.Exec(`DELETE FROM comments WHERE id = ?`, cid)
	writeJSON(w, 200, map[string]any{"ok": true})
}

// GET /api/admin/comments  — все комментарии для модерации
func AdminListComments(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query(`
		SELECT c.id, c.user_id, u.username, c.anime_id, c.text, c.approved, c.created_at
		FROM comments c
		JOIN users u ON u.id = c.user_id
		ORDER BY c.created_at DESC
		LIMIT 500
	`)
	if err != nil {
		writeErr(w, 500, "Ошибка БД")
		return
	}
	defer rows.Close()

	type item struct {
		ID        int64  `json:"id"`
		UserID    int64  `json:"user_id"`
		Username  string `json:"username"`
		AnimeID   string `json:"anime_id"`
		Text      string `json:"text"`
		Approved  int    `json:"approved"`
		CreatedAt int64  `json:"created_at"`
	}
	list := []item{}
	for rows.Next() {
		var c item
		_ = rows.Scan(&c.ID, &c.UserID, &c.Username, &c.AnimeID, &c.Text, &c.Approved, &c.CreatedAt)
		list = append(list, c)
	}
	writeJSON(w, 200, map[string]any{"comments": list})
}

// POST /api/admin/comments/{id}/approve  { approved: 0|1 }
func AdminApproveComment(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("id")
	var body struct {
		Approved int `json:"approved"`
	}
	if err := decode(r, &body); err != nil {
		writeErr(w, 400, "Некорректный запрос")
		return
	}
	_, _ = db.DB.Exec(`UPDATE comments SET approved = ? WHERE id = ?`, body.Approved, cid)
	writeJSON(w, 200, map[string]any{"ok": true})
}
