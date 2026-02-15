package domain

type Anime struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func NewAnime(id, title string) Anime {
	return Anime{
		ID:    id,
		Title: title,
	}
}

func (a Anime) GetID() string {
	return a.ID
}

func (a Anime) GetTitle() string {
	return a.Title
}
