package db

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init() error {
	dir := os.Getenv("DATA_DIR")
	if dir == "" {
		dir = "data"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "anime.db")

	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		return err
	}
	if err := db.Ping(); err != nil {
		return err
	}
	DB = db
	return migrate()
}

func migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'user',
		created_at INTEGER NOT NULL DEFAULT (strftime('%s','now'))
	);
	CREATE TABLE IF NOT EXISTS anime (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		original TEXT,
		year INTEGER,
		rating REAL DEFAULT 0,
		genres TEXT,
		description TEXT,
		poster TEXT,
		voices TEXT DEFAULT '[]',
		created_at INTEGER NOT NULL DEFAULT (strftime('%s','now'))
	);
	CREATE TABLE IF NOT EXISTS banners (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		subtitle TEXT,
		image TEXT,
		anime_id TEXT,
		created_at INTEGER NOT NULL DEFAULT (strftime('%s','now'))
	);
	CREATE TABLE IF NOT EXISTS favorites (
		user_id INTEGER NOT NULL,
		anime_id TEXT NOT NULL,
		added_at INTEGER NOT NULL DEFAULT (strftime('%s','now')),
		PRIMARY KEY (user_id, anime_id),
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	CREATE TABLE IF NOT EXISTS watch_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		anime_id TEXT NOT NULL,
		episode INTEGER NOT NULL,
		voice TEXT,
		watched_at INTEGER NOT NULL DEFAULT (strftime('%s','now')),
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	CREATE TABLE IF NOT EXISTS ratings (
    	user_id INTEGER NOT NULL,
    	anime_id TEXT NOT NULL,
    	score INTEGER NOT NULL CHECK (score BETWEEN 1 AND 10),
    	rated_at INTEGER NOT NULL DEFAULT (strftime('%s','now')),
    	PRIMARY KEY (user_id, anime_id),
    	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	CREATE TABLE IF NOT EXISTS comments (
    	id INTEGER PRIMARY KEY AUTOINCREMENT,
    	user_id INTEGER NOT NULL,
    	anime_id TEXT NOT NULL,
    	text TEXT NOT NULL,
    	approved INTEGER NOT NULL DEFAULT 1,
    	created_at INTEGER NOT NULL DEFAULT (strftime('%s','now')),
    	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	CREATE TABLE IF NOT EXISTS schedule (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		anime_id TEXT NOT NULL,
		weekday INTEGER NOT NULL CHECK (weekday BETWEEN 1 AND 7),
		episode INTEGER,
		air_time TEXT,
		FOREIGN KEY (anime_id) REFERENCES anime(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_comments_anime ON comments(anime_id, created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_ratings_anime ON ratings(anime_id);
	CREATE INDEX IF NOT EXISTS idx_schedule_weekday ON schedule(weekday);`

	if _, err := DB.Exec(schema); err != nil {
		return err
	}

	seedAdmin()
	seedAnime()
	seedBanners()
	return nil
}

func seedAdmin() {
	user := env("ADMIN_USER", "admin")
	pass := env("ADMIN_PASS", "admin123")

	var id int64
	err := DB.QueryRow(`SELECT id FROM users WHERE username = ?`, user).Scan(&id)
	if err == nil {
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(pass), 10)
	_, err = DB.Exec(
		`INSERT INTO users (username, email, password, role) VALUES (?, ?, ?, 'admin')`,
		user, user+"@anime-wor.local", string(hash),
	)
	if err == nil {
		log.Printf("[db] Создан админ: %s / %s", user, pass)
	}
}

func seedAnime() {
	var count int
	_ = DB.QueryRow(`SELECT COUNT(*) FROM anime`).Scan(&count)
	if count > 0 {
		return
	}

	type seedRow struct {
		ID, Title, Original, Genres, Desc, Poster, Status string
		Year                                              int
		Rating                                            float64
		Voices                                            []map[string]any
	}
	rows := []seedRow{
		{
			ID: "a1", Title: "Наруто", Original: "Naruto",
			Year: 2002, Rating: 8.4,
			Genres: "Экшен, Приключения, Сёнэн",
			Desc:   "Наруто Узумаки — юный ниндзя с девятихвостым лисом внутри. Он мечтает стать Хокаге.",
			Poster: "https://images.unsplash.com/photo-1578632767115-351597cf2477?w=500&q=80",
			Status: "Завершён",
			Voices: []map[string]any{
				{"name": "AniLibria", "episodes": []string{"https://www.youtube.com/embed/dQw4w9WgXcQ"}},
				{"name": "JAM Club", "episodes": []string{"https://www.youtube.com/embed/dQw4w9WgXcQ"}},
			},
		},
		{
			ID: "a5", Title: "Магическая битва", Original: "Jujutsu Kaisen",
			Year: 2020, Rating: 9.1,
			Genres: "Экшен, Мистика",
			Desc:   "Юдзи Итадори глотает проклятый палец и становится сосудом Сукуны.",
			Poster: "https://images.unsplash.com/photo-1613376023733-0a73315d9b06?w=500&q=80",
			Status: "Онгоинг",
			Voices: []map[string]any{
				{"name": "AniDUB", "episodes": []string{"https://www.youtube.com/embed/dQw4w9WgXcQ"}},
			},
		},
	}

	for _, r := range rows {
		vjson, _ := json.Marshal(r.Voices)
		_, _ = DB.Exec(
			`INSERT INTO anime (id, title, original, year, rating, genres, description, poster, status, voices)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			r.ID, r.Title, r.Original, r.Year, r.Rating, r.Genres, r.Desc, r.Poster, r.Status, string(vjson),
		)
	}
	log.Printf("[db] Загружены демо-аниме")
}

func seedBanners() {
	var count int
	_ = DB.QueryRow(`SELECT COUNT(*) FROM banners`).Scan(&count)
	if count > 0 {
		return
	}
	rows := []struct{ ID, Title, Sub, Img, AID string }{
		{"b1", "Наруто: Ураганные хроники", "Полное собрание приключений",
			"https://images.unsplash.com/photo-1607604276583-eef5d076aa5f?w=1600&q=80", "a2"},
		{"b2", "Магическая битва", "Мир проклятий и магов",
			"https://images.unsplash.com/photo-1613376023733-0a73315d9b06?w=1600&q=80", "a5"},
		{"b3", "Атака титанов", "Эпический финал",
			"https://images.unsplash.com/photo-1541562232579-512a21360020?w=1600&q=80", "a4"},
	}
	for _, r := range rows {
		_, _ = DB.Exec(
			`INSERT INTO banners (id, title, subtitle, image, anime_id) VALUES (?, ?, ?, ?, ?)`,
			r.ID, r.Title, r.Sub, r.Img, r.AID,
		)
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
