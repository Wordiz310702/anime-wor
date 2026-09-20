package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
)

var uploadDir string

func SetUploadDir(dir string) {
	uploadDir = dir
	_ = os.MkdirAll(dir, 0o755)
}

// POST /api/upload?kind=poster|banner|video
func Upload(w http.ResponseWriter, r *http.Request) {
	kind := r.URL.Query().Get("kind")
	if kind == "" {
		kind = "poster"
	}

	// Разные лимиты для разных типов
	maxSize := int64(20 << 20) // 20 МБ для картинок
	if kind == "video" {
		maxSize = 500 << 20 // 500 МБ для видео
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)
	if err := r.ParseMultipartForm(maxSize); err != nil {
		writeErr(w, 400, "Файл слишком большой")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeErr(w, 400, "Файл не получен")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))

	if kind == "video" {
		// Видео: сохраняем как есть
		allowed := map[string]bool{".mp4": true, ".webm": true, ".mkv": true, ".mov": true}
		if !allowed[ext] {
			writeErr(w, 400, "Видео: mp4, webm, mkv, mov")
			return
		}
		name := genName(ext)
		dst, err := os.Create(filepath.Join(uploadDir, name))
		if err != nil {
			writeErr(w, 500, "Ошибка сохранения")
			return
		}
		defer dst.Close()
		if _, err := io.Copy(dst, file); err != nil {
			writeErr(w, 500, "Ошибка записи")
			return
		}
		writeJSON(w, 200, map[string]string{"url": "/uploads/" + name})
		return
	}

	// Картинки: jpg/png/webp/gif
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true}
	if !allowed[ext] {
		writeErr(w, 400, "Только изображения: jpg, png, webp, gif")
		return
	}

	// Декодируем в память
	img, format, err := image.Decode(file)
	if err != nil {
		writeErr(w, 400, "Некорректное изображение")
		return
	}

	// GIF не пересжимаем (потеряется анимация)
	if format == "gif" {
		name := genName(".gif")
		// Перечитаем файл с начала
		file.Seek(0, 0)
		dst, _ := os.Create(filepath.Join(uploadDir, name))
		defer dst.Close()
		io.Copy(dst, file)
		writeJSON(w, 200, map[string]string{"url": "/uploads/" + name})
		return
	}

	// Автосжатие: разные размеры для разных типов
	var maxWidth int
	switch kind {
	case "banner":
		maxWidth = 1920
	case "poster":
		maxWidth = 800
	default:
		maxWidth = 1200
	}

	bounds := img.Bounds()
	if bounds.Dx() > maxWidth {
		ratio := float64(maxWidth) / float64(bounds.Dx())
		newHeight := int(float64(bounds.Dy()) * ratio)
		img = imaging.Resize(img, maxWidth, newHeight, imaging.Lanczos)
	}

	name := genName(".jpg")
	dstPath := filepath.Join(uploadDir, name)

	// Сохраняем как JPEG с качеством 85
	if err := imaging.Save(img, dstPath, imaging.JPEGQuality(85)); err != nil {
		writeErr(w, 500, "Ошибка сохранения файла")
		return
	}

	writeJSON(w, 200, map[string]string{"url": "/uploads/" + name})
}

// DELETE /api/upload/{filename}  (admin)
func DeleteFile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("filename")
	// Защита от path traversal
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		writeErr(w, 400, "Некорректное имя файла")
		return
	}
	path := filepath.Join(uploadDir, name)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			writeErr(w, 404, "Файл не найден")
			return
		}
		writeErr(w, 500, "Ошибка удаления")
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

// GET /api/upload/list  (admin) — список файлов в uploads/
func ListFiles(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		writeErr(w, 500, "Ошибка чтения папки")
		return
	}
	type f struct {
		Name    string `json:"name"`
		URL     string `json:"url"`
		Size    int64  `json:"size"`
		ModTime int64  `json:"mod_time"`
	}
	list := []f{}
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		info, _ := e.Info()
		list = append(list, f{
			Name:    e.Name(),
			URL:     "/uploads/" + e.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime().Unix(),
		})
	}
	writeJSON(w, 200, map[string]any{"files": list})
}

// Вспомогательная: удаление файла по URL (для удаления баннеров/аниме)
func DeleteFileByURL(url string) {
	if url == "" || !strings.HasPrefix(url, "/uploads/") {
		return
	}
	name := strings.TrimPrefix(url, "/uploads/")
	if strings.Contains(name, "..") || strings.Contains(name, "/") {
		return
	}
	_ = os.Remove(filepath.Join(uploadDir, name))
}

func genName(ext string) string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), hex.EncodeToString(b), ext)
}
