package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// =========================================================
//  KODIK — авто-токен + API
// =========================================================

// Публичный скрипт плеера Kodik, откуда берётся токен
const kodikPlayersScriptURL = "https://kodik-add.com/add-players.min.js?v=2"

// Кэш токена (чтобы не запрашивать каждый раз)
var (
	cachedKodikToken     string
	cachedKodikTokenTime time.Time
)

// ---------- ПОЛУЧЕНИЕ ТОКЕНА ----------

// GetPublicKodikToken извлекает публичный токен из скрипта плеера Kodik
// (аналог getPublicToken() из kodikwrapper / aniparsec-ru)
func GetPublicKodikToken() (string, error) {
	// Если токен получен меньше 10 минут назад — используем кэш
	if cachedKodikToken != "" && time.Since(cachedKodikTokenTime) < 10*time.Minute {
		return cachedKodikToken, nil
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", kodikPlayersScriptURL, nil)
	if err != nil {
		return "", fmt.Errorf("не удалось создать запрос: %w", err)
	}
	// User-Agent — как у обычного браузера
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://animego.me/")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("не удалось скачать скрипт Kodik: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("скрипт Kodik вернул статус %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("не удалось прочитать ответ: %w", err)
	}

	script := string(body)

	// Ищем токен в скрипте
	// Обычно выглядит как: token: 'abcdef1234567890...' или token="..."
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`token\s*[:=]\s*["']([a-f0-9]{30,})["']`),
		regexp.MustCompile(`["']token["']\s*:\s*["']([a-f0-9]{30,})["']`),
		regexp.MustCompile(`token["']?\s*[:=]\s*["']([a-f0-9]{20,})["']`),
		regexp.MustCompile(`["']([a-f0-9]{40,})["']`), // запасной вариант
	}

	for _, re := range patterns {
		matches := re.FindStringSubmatch(script)
		if len(matches) >= 2 && len(matches[1]) >= 30 {
			token := matches[1]
			cachedKodikToken = token
			cachedKodikTokenTime = time.Now()
			fmt.Printf("[Kodik] Токен получен: %s...\n", token[:10])
			return token, nil
		}
	}

	return "", fmt.Errorf("не удалось найти токен в скрипте Kodik")
}

// ---------- API ЗАПРОСЫ ----------

// KodikResult — один результат поиска в Kodik
type KodikResult struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Link        string `json:"link"`
	Title       string `json:"title"`
	TitleOrig   string `json:"title_orig"`
	Translation struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
		Type  string `json:"type"`
	} `json:"translation"`
	Year        int    `json:"year"`
	Quality     string `json:"quality"`
	ShikimoriID string `json:"shikimori_id"`
}

// KodikResponse — ответ API Kodik
type KodikResponse struct {
	Time    string        `json:"time"`
	Total   int           `json:"total"`
	Results []KodikResult `json:"results"`
	Error   string        `json:"error,omitempty"`
}

// SearchKodikByShikimori ищет видео в Kodik по ID аниме на Shikimori
func SearchKodikByShikimori(shikimoriID string) ([]KodikResult, error) {
	token, err := GetPublicKodikToken()
	if err != nil {
		return nil, fmt.Errorf("токен: %w", err)
	}

	apiURL := fmt.Sprintf(
		"https://kodik-api.com/search?token=%s&shikimori_id=%s&limit=50",
		token, shikimoriID,
	)

	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("не удалось обратиться к API Kodik: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Проверяем, не вернул ли Kodik ошибку
	if strings.Contains(string(body), `"error"`) {
		var errResp KodikResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("Kodik вернул ошибку: %s", errResp.Error)
		}
	}

	var result KodikResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("не удалось разобрать ответ: %w (ответ: %s)", err, string(body)[:min(200, len(body))])
	}

	return result.Results, nil
}

// ---------- HTTP-ОБРАБОТЧИК ----------

// FetchKodikVideo — HTTP-обработчик для получения видео из Kodik
// Пример: GET /api/kodik/fetch?shikimori_id=40748
func FetchKodikVideo(w http.ResponseWriter, r *http.Request) {
	shikimoriID := r.URL.Query().Get("shikimori_id")
	if shikimoriID == "" {
		writeErr(w, 400, "Параметр 'shikimori_id' обязателен")
		return
	}

	results, err := SearchKodikByShikimori(shikimoriID)
	if err != nil {
		writeErr(w, 500, "Ошибка Kodik: "+err.Error())
		return
	}

	if len(results) == 0 {
		writeJSON(w, 200, map[string]any{
			"shikimori_id": shikimoriID,
			"count":        0,
			"results":      []KodikResult{},
			"message":      "Видео не найдено в Kodik",
		})
		return
	}

	// Группируем по озвучкам
	type TranslationGroup struct {
		TranslationID    int           `json:"translation_id"`
		TranslationTitle string        `json:"translation_title"`
		TranslationType  string        `json:"translation_type"`
		Episodes         []KodikResult `json:"episodes"`
	}

	groupsMap := make(map[int]*TranslationGroup)
	order := []int{}

	for _, r := range results {
		tid := r.Translation.ID
		if _, ok := groupsMap[tid]; !ok {
			groupsMap[tid] = &TranslationGroup{
				TranslationID:    tid,
				TranslationTitle: r.Translation.Title,
				TranslationType:  r.Translation.Type,
				Episodes:         []KodikResult{},
			}
			order = append(order, tid)
		}
		groupsMap[tid].Episodes = append(groupsMap[tid].Episodes, r)
	}

	groups := make([]*TranslationGroup, 0, len(order))
	for _, tid := range order {
		groups = append(groups, groupsMap[tid])
	}

	writeJSON(w, 200, map[string]any{
		"shikimori_id": shikimoriID,
		"total":        len(results),
		"voices":       groups,
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
