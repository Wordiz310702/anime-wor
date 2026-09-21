package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"
)

func kodikServiceURL() string {
	u := os.Getenv("KODIK_SERVICE_URL")
	if u == "" {
		u = "http://localhost:3333"
	}
	return u
}

// ParseKodikLink — принимает seria-ссылку Kodik, возвращает .m3u8
// POST /api/kodik/parse
// Body: { "link": "//kodikplayer.com/seria/..." }
func ParseKodikLink(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Link string `json:"link"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, 400, "Некорректный JSON")
		return
	}
	if body.Link == "" {
		writeErr(w, 400, "Параметр 'link' обязателен")
		return
	}

	payload, _ := json.Marshal(map[string]string{"link": body.Link})
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Post(
		kodikServiceURL()+"/api/parse",
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		writeErr(w, 500, "Kodik-сервис недоступен: "+err.Error())
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(respBody)
}
