package main

import (
	"bufio"
	"log"
	"net/http"
	"os"
	"strings"

	"anime-wor/internal/auth"
	"anime-wor/internal/db"
	"anime-wor/internal/handlers"
)

func main() {
	loadEnv(".env")

	if err := db.Init(); err != nil {
		log.Fatalf("DB init failed: %v", err)
	}
	auth.Init()

	// Папка для загруженных файлов
	handlers.SetUploadDir("public/uploads")

	mux := http.NewServeMux()

	// ---------- API ----------
	// Users
	mux.HandleFunc("POST /api/users/register", handlers.Register)
	mux.HandleFunc("POST /api/users/login", handlers.Login)
	mux.HandleFunc("POST /api/users/logout", handlers.Logout)
	mux.HandleFunc("GET /api/users/me", handlers.Me)
	mux.HandleFunc("GET /api/users/favorites", auth.RequireAuth(handlers.GetFavorites))
	mux.HandleFunc("POST /api/users/favorites/{animeId}", auth.RequireAuth(handlers.ToggleFavorite))
	mux.HandleFunc("POST /api/users/history", auth.RequireAuth(handlers.AddHistory))
	mux.HandleFunc("GET /api/users/history", auth.RequireAuth(handlers.GetHistory))

	// Anime
	mux.HandleFunc("GET /api/anime", handlers.ListAnime)
	mux.HandleFunc("GET /api/anime/{id}", handlers.GetAnime)
	mux.HandleFunc("POST /api/anime", auth.RequireAdmin(handlers.CreateAnime))
	mux.HandleFunc("PUT /api/anime/{id}", auth.RequireAdmin(handlers.UpdateAnime))
	mux.HandleFunc("DELETE /api/anime/{id}", auth.RequireAdmin(handlers.DeleteAnime))

	// Banners
	mux.HandleFunc("GET /api/banners", handlers.ListBanners)
	mux.HandleFunc("POST /api/banners", auth.RequireAdmin(handlers.CreateBanner))
	mux.HandleFunc("DELETE /api/banners/{id}", auth.RequireAdmin(handlers.DeleteBanner))

	// Upload
	mux.HandleFunc("POST /api/upload", auth.RequireAdmin(handlers.Upload))
	mux.HandleFunc("DELETE /api/upload/{filename}", auth.RequireAdmin(handlers.DeleteFile))
	mux.HandleFunc("GET /api/upload/list", auth.RequireAdmin(handlers.ListFiles))
	mux.HandleFunc("GET /api/anime/{id}/rating", handlers.GetRating)
	mux.HandleFunc("POST /api/anime/{id}/rating", auth.RequireAuth(handlers.SetRating))
	mux.HandleFunc("GET /api/schedule", handlers.ListSchedule)
	mux.HandleFunc("POST /api/schedule", auth.RequireAdmin(handlers.CreateSchedule))
	mux.HandleFunc("DELETE /api/schedule/{id}", auth.RequireAdmin(handlers.DeleteSchedule))
	mux.HandleFunc("GET /api/admin/stats", auth.RequireAdmin(handlers.Stats))

	// Comments
	mux.HandleFunc("GET /api/anime/{id}/comments", handlers.ListComments)
	mux.HandleFunc("POST /api/anime/{id}/comments", auth.RequireAuth(handlers.AddComment))
	mux.HandleFunc("DELETE /api/comments/{id}", auth.RequireAuth(handlers.DeleteComment))
	mux.HandleFunc("GET /api/admin/comments", auth.RequireAdmin(handlers.AdminListComments))
	mux.HandleFunc("POST /api/admin/comments/{id}/approve", auth.RequireAdmin(handlers.AdminApproveComment))

	// ---------- Статика ----------
	fs := http.FileServer(http.Dir("public"))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// SPA fallback: не-api пути отдают index.html
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			path := "public" + r.URL.Path
			if _, err := os.Stat(path); os.IsNotExist(err) || r.URL.Path == "/" {
				http.ServeFile(w, r, "public/index.html")
				return
			}
		}
		fs.ServeHTTP(w, r)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("✓ Anime-Wor запущен: http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

// Простой парсер .env
func loadEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, `"'`)
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}
