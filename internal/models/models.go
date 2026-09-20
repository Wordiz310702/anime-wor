package models

type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	CreatedAt int64  `json:"created_at,omitempty"`
}

type Voice struct {
	Name     string   `json:"name"`
	Episodes []string `json:"episodes"`
}

type Anime struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Original    string   `json:"original"`
	Year        int      `json:"year"`
	Rating      float64  `json:"rating"`
	Genres      []string `json:"genres"`
	Description string   `json:"description"`
	Poster      string   `json:"poster"`
	Status      string   `json:"status"`
	Voices      []Voice  `json:"voices"`
}

type Banner struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Image    string `json:"image"`
	AnimeID  string `json:"anime_id"`
}
