package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// ShikimoriAnime — структура аниме из ответа Shikimori
// ShikimoriAnime — структура аниме из ответа Shikimori
// Score и Episodes приходят строками или null — используем interface{}
type ShikimoriAnime struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Russian string `json:"russian"`
	Image   struct {
		Original string `json:"original"`
		Preview  string `json:"preview"`
	} `json:"image"`
	Score    any    `json:"score"`    // может быть число, строка или null
	Episodes any    `json:"episodes"` // может быть число или null
	Status   string `json:"status"`
	Kind     string `json:"kind"`
	AiredOn  string `json:"aired_on"`
}

// SearchShikimori ищет аниме по названию
func SearchShikimori(query string, limit int) ([]ShikimoriAnime, error) {
	if limit <= 0 || limit > 50 {
		limit = 5
	}

	// Формируем URL
	apiURL := fmt.Sprintf(
		"https://shikimori.io/api/animes?search=%s&limit=%d",
		url.QueryEscape(query), limit,
	)

	// Создаём запрос с обязательным User-Agent
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать запрос: %w", err)
	}
	req.Header.Set("User-Agent", "Anime-Wor/1.0")

	// Выполняем запрос
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("не удалось обратиться к Shikimori: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Shikimori вернул статус %d", resp.StatusCode)
	}

	// Разбираем JSON
	var results []ShikimoriAnime
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("не удалось разобрать ответ: %w", err)
	}

	return results, nil
}

// SearchShikimoriHandler — HTTP-обработчик для поиска аниме на Shikimori
// Пример: GET /api/shikimori/search?q=Наруто&limit=5
// SearchShikimoriHandler — HTTP-обработчик для поиска аниме на Shikimori
// Пример: GET /api/shikimori/search?q=Наруто&limit=5
func SearchShikimoriHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeErr(w, 400, "Параметр 'q' обязателен")
		return
	}

	limit := 5
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	results, err := SearchShikimori(query, limit)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}

	// Приводим score и episodes к нормальному виду
	type CleanAnime struct {
		ID       int     `json:"id"`
		Name     string  `json:"name"`
		Russian  string  `json:"russian"`
		Poster   string  `json:"poster"`
		Score    float64 `json:"score"`
		Episodes int     `json:"episodes"`
		Status   string  `json:"status"`
		Kind     string  `json:"kind"`
		AiredOn  string  `json:"aired_on"`
	}

	cleaned := make([]CleanAnime, 0, len(results))
	for _, a := range results {
		cleaned = append(cleaned, CleanAnime{
			ID:       a.ID,
			Name:     a.Name,
			Russian:  a.Russian,
			Poster:   "https://shikimori.io" + a.Image.Original,
			Score:    toFloat(a.Score),
			Episodes: toInt(a.Episodes),
			Status:   a.Status,
			Kind:     a.Kind,
			AiredOn:  a.AiredOn,
		})
	}

	writeJSON(w, 200, map[string]any{
		"query":   query,
		"count":   len(cleaned),
		"results": cleaned,
	})
}

// toFloat приводит любое значение (число, строка, null) к float64
func toFloat(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case string:
		var f float64
		fmt.Sscanf(val, "%f", &f)
		return f
	default:
		return 0
	}
}

// toInt приводит любое значение к int
func toInt(v any) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case string:
		var n int
		fmt.Sscanf(val, "%d", &n)
		return n
	default:
		return 0
	}
}

// ShikimoriDetails — полная информация об аниме с Shikimori
type ShikimoriDetails struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Russian     string `json:"russian"`
	Description string `json:"description"`
	Score       any    `json:"score"`
	Episodes    any    `json:"episodes"`
	Status      string `json:"status"`
	Kind        string `json:"kind"`
	AiredOn     string `json:"aired_on"`
	Image       struct {
		Original string `json:"original"`
	} `json:"image"`
	Genres []struct {
		Russian string `json:"russian"`
		Name    string `json:"name"`
	} `json:"genres"`
	Studios []struct {
		Name string `json:"name"`
	} `json:"studios"`
}

// GetShikimoriDetails получает полную информацию об аниме по его ID на Shikimori
func GetShikimoriDetails(shikimoriID int) (*ShikimoriDetails, error) {
	apiURL := fmt.Sprintf("https://shikimori.io/api/animes/%d", shikimoriID)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать запрос: %w", err)
	}
	req.Header.Set("User-Agent", "Anime-Wor/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("не удалось обратиться к Shikimori: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Shikimori вернул статус %d", resp.StatusCode)
	}

	var details ShikimoriDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, fmt.Errorf("не удалось разобрать ответ: %w", err)
	}

	return &details, nil
}

// GetShikimoriDetailsHandler — HTTP-обработчик
// Пример: GET /api/shikimori/details?id=40748
func GetShikimoriDetailsHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		writeErr(w, 400, "Параметр 'id' обязателен")
		return
	}

	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		writeErr(w, 400, "Некорректный ID")
		return
	}

	details, err := GetShikimoriDetails(id)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}

	// Приводим к чистому виду
	genres := make([]string, 0, len(details.Genres))
	for _, g := range details.Genres {
		if g.Russian != "" {
			genres = append(genres, g.Russian)
		} else {
			genres = append(genres, g.Name)
		}
	}

	studios := make([]string, 0, len(details.Studios))
	for _, s := range details.Studios {
		studios = append(studios, s.Name)
	}

	year := 0
	if len(details.AiredOn) >= 4 {
		fmt.Sscanf(details.AiredOn[:4], "%d", &year)
	}

	writeJSON(w, 200, map[string]any{
		"id":          details.ID,
		"name":        details.Name,
		"russian":     details.Russian,
		"description": stripHTML(details.Description),
		"poster":      "https://shikimori.io" + details.Image.Original,
		"score":       toFloat(details.Score),
		"episodes":    toInt(details.Episodes),
		"status":      details.Status,
		"kind":        details.Kind,
		"year":        year,
		"genres":      genres,
		"studios":     studios,
	})
}

// stripHTML удаляет HTML-теги из описания
func stripHTML(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	// Заменяем HTML-сущности
	out := result.String()
	out = strings.ReplaceAll(out, "&quot;", `"`)
	out = strings.ReplaceAll(out, "&amp;", "&")
	out = strings.ReplaceAll(out, "&lt;", "<")
	out = strings.ReplaceAll(out, "&gt;", ">")
	out = strings.ReplaceAll(out, "&#39;", "'")
	return strings.TrimSpace(out)
}
