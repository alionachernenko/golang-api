package entities

import "time"

type Article struct {
	Id        int       `json:"id"`
	AuthorId  int       `json:"authorId"`
	CompanyId int       `json:"companyId"`
	Title     string    `json:"title"`
	Text      string    `json:"text"`
	CoverUrl  string    `json:"coverUrl"`
	Rating    int       `json:"rating"`
	CreatedAt time.Time `json:"createdAt"`
}
