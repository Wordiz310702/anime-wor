package handlers

import (
	"net/http"

	"anime-wor/internal/db"
)

// GET /api/admin/stats  (admin)
func Stats(w http.ResponseWriter, r *http.Request) {
	var users, anime, banners, comments, favorites, ratings, schedules int
	_ = db.DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&users)
	_ = db.DB.QueryRow(`SELECT COUNT(*) FROM anime`).Scan(&anime)
	_ = db.DB.QueryRow(`SELECT COUNT(*) FROM banners`).Scan(&banners)
	_ = db.DB.QueryRow(`SELECT COUNT(*) FROM comments`).Scan(&comments)
	_ = db.DB.QueryRow(`SELECT COUNT(*) FROM favorites`).Scan(&favorites)
	_ = db.DB.QueryRow(`SELECT COUNT(*) FROM ratings`).Scan(&ratings)
	_ = db.DB.QueryRow(`SELECT COUNT(*) FROM schedule`).Scan(&schedules)

	type topItem struct {
		AnimeID string  `json:"anime_id"`
		Title   string  `json:"title"`
		Poster  string  `json:"poster"`
		Views   int     `json:"views"`
		Rating  float64 `json:"rating"`
	}

	// Топ-5 по просмотрам
	rows, _ := db.DB.Query(`
		SELECT wh.anime_id, a.title, a.poster, COUNT(*) as views
		FROM watch_history wh
		LEFT JOIN anime a ON a.id = wh.anime_id
		GROUP BY wh.anime_id
		ORDER BY views DESC
		LIMIT 5
	`)
	topViews := []topItem{}
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var it topItem
			_ = rows.Scan(&it.AnimeID, &it.Title, &it.Poster, &it.Views)
			topViews = append(topViews, it)
		}
	}

	// Топ-5 по рейтингу
	rows2, _ := db.DB.Query(`
		SELECT anime_id, AVG(score) as avg_score, COUNT(*) as cnt
		FROM ratings
		GROUP BY anime_id
		ORDER BY avg_score DESC
		LIMIT 5
	`)
	topRated := []map[string]any{}
	if rows2 != nil {
		defer rows2.Close()
		for rows2.Next() {
			var aid string
			var avg float64
			var cnt int
			_ = rows2.Scan(&aid, &avg, &cnt)
			var title, poster string
			_ = db.DB.QueryRow(`SELECT title, poster FROM anime WHERE id = ?`, aid).Scan(&title, &poster)
			topRated = append(topRated, map[string]any{
				"anime_id": aid, "title": title, "poster": poster,
				"avg": avg, "count": cnt,
			})
		}
	}

	writeJSON(w, 200, map[string]any{
		"counts": map[string]int{
			"users": users, "anime": anime, "banners": banners,
			"comments": comments, "favorites": favorites,
			"ratings": ratings, "schedules": schedules,
		},
		"top_views": topViews,
		"top_rated": topRated,
	})
}
