package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"anime-wor/internal/db"
	"anime-wor/internal/models"
)

// Явный список колонок в порядке, соответствующем rowToAnime.
// ВАЖНО: SELECT * НЕ используем — из-за ALTER TABLE порядок колонок в базе
// отличается от ожидаемого.
const animeColumns = `id, title, original, year, rating, genres, description, poster, status, voices`

// rowToAnime конвертирует строку БД в модель Anime.
// Ожидаемый порядок колонок: id, title, original, year, rating, genres, description, poster, status, voices
func rowToAnime(row interface{ Scan(...any) error }) (*models.Anime, error) {
	var a models.Anime
	var genres, voices, status, poster, description, original sql.NullString
	var year sql.NullInt64
	var rating sql.NullFloat64

	err := row.Scan(
		&a.ID, &a.Title, &original, &year, &rating,
		&genres, &description, &poster, &status, &voices,
	)
	if err != nil {
		return nil, err
	}

	a.Original = original.String
	a.Year = int(year.Int64)
	a.Rating = rating.Float64
	a.Description = description.String
	a.Poster = poster.String
	a.Status = status.String
	if a.Status == "" {
		a.Status = "Завершён"
	}

	if genres.String != "" {
		for _, p := range strings.Split(genres.String, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				a.Genres = append(a.Genres, p)
			}
		}
	}
	if a.Genres == nil {
		a.Genres = []string{}
	}

	if voices.String != "" {
		_ = json.Unmarshal([]byte(voices.String), &a.Voices)
	}
	if a.Voices == nil {
		a.Voices = []models.Voice{}
	}

	return &a, nil
}

// GET /api/anime
func ListAnime(w http.ResponseWriter, r *http.Request) {
	query := `SELECT ` + animeColumns + ` FROM anime ORDER BY created_at DESC`
	rows, err := db.DB.Query(query)
	if err != nil {
		log.Printf("[ListAnime] Query error: %v", err)
		writeErr(w, 500, "Ошибка БД: "+err.Error())
		return
	}
	defer rows.Close()

	list := []*models.Anime{}
	for rows.Next() {
		a, err := rowToAnime(rows)
		if err != nil {
			log.Printf("[ListAnime] Scan error: %v", err)
			continue
		}
		list = append(list, a)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[ListAnime] Rows error: %v", err)
	}

	log.Printf("[ListAnime] Отдаём записей: %d", len(list))
	writeJSON(w, 200, map[string]any{"anime": list})
}

// GET /api/anime/{id}
func GetAnime(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	query := `SELECT ` + animeColumns + ` FROM anime WHERE id = ?`
	row := db.DB.QueryRow(query, id)
	a, err := rowToAnime(row)
	if errors.Is(err, sql.ErrNoRows) {
		writeErr(w, 404, "Не найдено")
		return
	}
	if err != nil {
		log.Printf("[GetAnime] Scan error: %v", err)
		writeErr(w, 500, "Ошибка: "+err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"anime": a})
}

func newID(prefix string) string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}

// POST /api/anime  (admin)
func CreateAnime(w http.ResponseWriter, r *http.Request) {
	var a models.Anime
	if err := decode(r, &a); err != nil {
		writeErr(w, 400, "Некорректный запрос")
		return
	}
	if a.ID == "" {
		a.ID = newID("a")
	}
	if a.Status == "" {
		a.Status = "Завершён"
	}
	if a.Genres == nil {
		a.Genres = []string{}
	}
	if a.Voices == nil {
		a.Voices = []models.Voice{}
	}

	voices, _ := json.Marshal(a.Voices)
	_, err := db.DB.Exec(`
		INSERT INTO anime (id, title, original, year, rating, genres, description, poster, status, voices)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.Title, a.Original, a.Year, a.Rating,
		strings.Join(a.Genres, ", "), a.Description, a.Poster, a.Status, string(voices),
	)
	if err != nil {
		log.Printf("[CreateAnime] INSERT error: %v", err)
		writeErr(w, 500, "Ошибка сохранения: "+err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"anime": a})
}

// PUT /api/anime/{id}  (admin)
func UpdateAnime(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var a models.Anime
	if err := decode(r, &a); err != nil {
		writeErr(w, 400, "Некорректный запрос")
		return
	}
	if a.Status == "" {
		a.Status = "Завершён"
	}
	if a.Genres == nil {
		a.Genres = []string{}
	}
	if a.Voices == nil {
		a.Voices = []models.Voice{}
	}

	voices, _ := json.Marshal(a.Voices)
	res, err := db.DB.Exec(`
		UPDATE anime SET title=?, original=?, year=?, rating=?, genres=?, description=?, poster=?, status=?, voices=?
		WHERE id = ?`,
		a.Title, a.Original, a.Year, a.Rating,
		strings.Join(a.Genres, ", "), a.Description, a.Poster, a.Status, string(voices),
		id,
	)
	if err != nil {
		log.Printf("[UpdateAnime] UPDATE error: %v", err)
		writeErr(w, 500, "Ошибка обновления: "+err.Error())
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeErr(w, 404, "Не найдено")
		return
	}
	a.ID = id
	writeJSON(w, 200, map[string]any{"anime": a})
}

// DELETE /api/anime/{id}  (admin)
func DeleteAnime(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var poster string
	_ = db.DB.QueryRow(`SELECT poster FROM anime WHERE id = ?`, id).Scan(&poster)

	_, err := db.DB.Exec(`DELETE FROM anime WHERE id = ?`, id)
	if err != nil {
		log.Printf("[DeleteAnime] DELETE error: %v", err)
		writeErr(w, 500, "Ошибка удаления")
		return
	}

	DeleteFileByURL(poster)

	writeJSON(w, 200, map[string]any{"ok": true})
}
