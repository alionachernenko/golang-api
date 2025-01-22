package entities

type User struct {
	Id        int    `json:"id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Fullname  string `json:"fullName"`
	CompanyId *int   `json:"companyId,omitempty"`
	Position  string `json:"position"`
	AvatarUrl string `json:"avatarURL"`
}
