package entities

type Company struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	Key         string `json:"key"`
	Description string `json:"description"`
	Website     string `json:"website"`
	LogoUrl     string `json:"logoURL"`
}
