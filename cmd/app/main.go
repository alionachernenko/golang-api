package main

import (
	"auth-service/internal/auth"
	"auth-service/internal/database"
	"auth-service/internal/transport"
	"fmt"
	"os"

	"net/http"
)

func main() {
	mux := http.NewServeMux()

	connStr := os.Getenv("POSTGRES_CONN_STR")

	storage, err := database.NewPostgresStorage(connStr)

	if err != nil {
		fmt.Printf("Failed to open database: %v", err)
	}

	resourse := transport.NewResourse(storage)

	mux.HandleFunc("/signin", resourse.Login)

	mux.HandleFunc("GET /users", auth.CheckAuth(resourse.GetUsers))
	mux.HandleFunc("GET /users/{id}", auth.CheckAuth(resourse.GetUserById))
	mux.HandleFunc("/users", resourse.CreateUser)
	mux.HandleFunc("/users/{id}", auth.CheckAuth(resourse.UpdateUser))
	mux.HandleFunc("DELETE /users/{id}", auth.CheckAuth(resourse.DeleteUser))
	mux.HandleFunc("/users/{id}/photo", auth.CheckAuth(resourse.UpdateUserPhoto))

	mux.HandleFunc("/articles", auth.CheckAuth(resourse.GetArticles))
	mux.HandleFunc("GET /articles/{id}", resourse.GetArticleById)
	mux.HandleFunc("POST /articles", resourse.CreateArticle)
	mux.HandleFunc("PUT /articles/{id}", auth.CheckAuth(resourse.UpdateArticle))
	mux.HandleFunc("DELETE /articles/{id}", auth.CheckAuth(resourse.DeleteArticle))
	mux.HandleFunc("GET /users/{id}/articles", resourse.GetArticlesByAuthorId)
	mux.HandleFunc("GET /companies/{id}/articles", resourse.GetArticlesByCompanyId)

	mux.HandleFunc("GET /companies", resourse.GetCompanies)
	mux.HandleFunc("GET /companies/{id}", resourse.GetCompanyById)
	mux.HandleFunc("/companies", resourse.CreateCompany)
	mux.HandleFunc("PUT /companies/{id}", auth.CheckAuth(resourse.UpdateCompany))
	mux.HandleFunc("DELETE /companies/{id}", auth.CheckAuth(resourse.DeleteCompany))
	mux.HandleFunc("/join-company", auth.CheckAuth(resourse.JoinCompany))

	http.ListenAndServe(":8080", mux)
}

type User struct {
	Username string
	Email    string
	Password string
}

type SignInResponse struct {
	Token string `json:"token"`
}
