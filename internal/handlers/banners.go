package handlers

import (
	"database/sql"
	"errors"
	"net/http"

	"anime-wor/internal/db"
	"anime-wor/internal/models"
)

// GET /api/banners
func ListBanners(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query(`SELECT id, title, subtitle, image, anime_id FROM banners ORDER BY created_at DESC`)
	if err != nil {
		writeErr(w, 500, "Ошибка БД")
		return
	}
	defer rows.Close()

	list := []models.Banner{}
	for rows.Next() {
		var b models.Banner
		var sub, img, aid sql.NullString
		_ = rows.Scan(&b.ID, &b.Title, &sub, &img, &aid)
		b.Subtitle = sub.String
		b.Image = img.String
		b.AnimeID = aid.String
		list = append(list, b)
	}
	writeJSON(w, 200, map[string]any{"banners": list})
}

// POST /api/banners  (admin)
func CreateBanner(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title    string `json:"title"`
		Subtitle string `json:"subtitle"`
		Image    string `json:"image"`
		AnimeID  string `json:"animeId"`
	}
	if err := decode(r, &body); err != nil || body.Title == "" {
		writeErr(w, 400, "Некорректный запрос")
		return
	}
	id := newID("b")
	_, err := db.DB.Exec(
		`INSERT INTO banners (id, title, subtitle, image, anime_id) VALUES (?, ?, ?, ?, ?)`,
		id, body.Title, body.Subtitle, body.Image, body.AnimeID,
	)
	if err != nil {
		writeErr(w, 500, "Ошибка сохранения")
		return
	}
	writeJSON(w, 200, map[string]any{"banner": models.Banner{
		ID: id, Title: body.Title, Subtitle: body.Subtitle, Image: body.Image, AnimeID: body.AnimeID,
	}})
}

// DELETE /api/banners/{id}  (admin)
func DeleteBanner(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Сначала получим URL картинки
	var image string
	_ = db.DB.QueryRow(`SELECT image FROM banners WHERE id = ?`, id).Scan(&image)

	res, err := db.DB.Exec(`DELETE FROM banners WHERE id = ?`, id)
	if err != nil {
		writeErr(w, 500, "Ошибка удаления")
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeErr(w, 404, "Не найдено")
		return
	}

	// Удаляем файл с диска
	DeleteFileByURL(image)

	writeJSON(w, 200, map[string]any{"ok": true})
}

var _ = errors.Is // неиспользуемый импорт на случай чистки
