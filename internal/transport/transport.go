package transport

import (
	"auth-service/internal/auth"
	"auth-service/internal/cors"
	"auth-service/internal/database"
	"auth-service/internal/entities"
	"auth-service/internal/keys"
	"auth-service/internal/storage"
	"auth-service/pkg/cookie"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/rs/zerolog/log"
)

type Resourse struct {
	s *database.PostgresStorage
}

func NewResourse(s *database.PostgresStorage) *Resourse {
	return &Resourse{s: s}
}

type LoginRresponse struct {
	Token    string        `json:"token"`
	UserData entities.User `json:"userData"`
}

func (res *Resourse) Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")

	cors.EnableCors(&w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var usr entities.User

	err := json.NewDecoder(r.Body).Decode(&usr)

	if err != nil {
		http.Error(w, "Could not parse request data", http.StatusBadRequest)
		return
	}

	userData, err := res.s.GetUserByUsername(usr.Username)

	if err != nil {
		log.Error().Err(err).Msg("User not found")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !auth.CheckPasswordHash(usr.Password, userData.Password) {
		log.Error().Err(err).Msg("Invalid password")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	token, err := auth.CreateToken(usr.Username)

	if err != nil {
		http.Error(w, "Problem with generating a token", http.StatusInternalServerError)
		return
	}

	tokenCookie := cookie.NewAccessTokenCookie(w, token)

	cookie.Write(w, tokenCookie)

	json.NewEncoder(w).Encode(LoginRresponse{
		UserData: userData,
	})
}

// users
func (res *Resourse) GetUsers(w http.ResponseWriter, r *http.Request) {
	cors.EnableCors(&w)

	users, err := res.s.GetUsers()

	if err != nil {
		log.Error().Err(err).Msg("Failed to get users")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Debug().Msgf("users: %v", users)

	err = json.NewEncoder(w).Encode(users)

	if err != nil {
		log.Error().Err(err).Msg("Failed to encode")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

type CreateUserResponse struct {
	Id int `json:"id"`
}

func (res *Resourse) CreateUser(w http.ResponseWriter, r *http.Request) {
	cors.EnableCors(&w)

	w.Header().Set("Access-Control-Allow-Methods", "POST")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var reqBody entities.User

	err := json.NewDecoder(r.Body).Decode(&reqBody)

	if err != nil {
		log.Error().Err(err).Msg("Failed to decode")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	password, err := auth.HashPassword(reqBody.Password)

	if err != nil {
		log.Error().Err(err).Msg("Failed to hash user password")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	reqBody.Password = password

	id, err := res.s.InsertUser(reqBody)

	if err != nil {
		log.Error().Err(err).Msg("Failed to create user")

		if err.Error() == "email already exists" || err.Error() == "username already exists" {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		http.Error(w, "Failed to create user", http.StatusBadRequest)
		return
	}

	token, err := auth.CreateToken(reqBody.Username)

	if err != nil {
		http.Error(w, "Failed to create token", http.StatusInternalServerError)
		return
	}

	tokenCookie := cookie.NewAccessTokenCookie(w, token)

	err = cookie.Write(w, tokenCookie)

	if err != nil {
		http.Error(w, "Failed to write cookie", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(CreateUserResponse{
		Id: id,
	})

	w.WriteHeader(http.StatusOK)
}

func (res *Resourse) GetUserById(w http.ResponseWriter, r *http.Request) {
	cors.EnableCors(&w)

	idVal := r.PathValue("id")

	id, err := strconv.Atoi(idVal)

	if err != nil {
		log.Error().Err(err).Msg("Failed to convert id to integer")

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	user, err := res.s.GetUserById(id)

	if err != nil {
		log.Error().Err(err).Msg("Failed to get user by id")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = json.NewEncoder(w).Encode(user)

	if err != nil {
		log.Error().Err(err).Msg("Failed to encode")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (res *Resourse) UpdateUserPhoto(w http.ResponseWriter, r *http.Request) {
	cors.EnableCors(&w)

	w.Header().Set("Access-Control-Allow-Methods", "PUT, OPTIONS")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPut {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse multipart form")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	idVal := r.PathValue("id")

	id, err := strconv.Atoi(idVal)

	if err != nil {
		log.Error().Err(err).Msg("Failed to convert id to integer")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	file, fileHeader, err := r.FormFile("photo")

	if err != nil {
		log.Error().Err(err).Msg("Failed to get file from form")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	defer file.Close()

	avatarsFolder := "avatars"

	fileURL, err := storage.UploadFileToS3(file, avatarsFolder, fileHeader, keys.BUCKET_NAME, keys.AWS_REGION, keys.AWS_ACCESS_KEY, keys.AWS_SECRET_KEY)

	if err != nil {
		log.Error().Err(err).Msg("Failed to upload file to S3")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = res.s.UpdateUserPhoto(fileURL, id)

	if err != nil {
		log.Error().Err(err).Msg("Failed to update user's photo URL")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "User photo updated successfully", "photoURL": fileURL})
}

func (res *Resourse) UpdateUser(w http.ResponseWriter, r *http.Request) {
	cors.EnableCors(&w)

	w.Header().Set("Access-Control-Allow-Methods", "POST")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20)

	if err != nil {
		log.Error().Err(err).Msg("Failed to parse multipart form")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	idVal := r.PathValue("id")
	var reqBody entities.User

	id, err := strconv.Atoi(idVal)

	if err != nil {
		log.Error().Err(err).Msg("Failed to convert id to integer")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	file, fileHeader, err := r.FormFile("photo")
	if err != nil {
		log.Error().Err(err).Msg("Failed to get file from form")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer file.Close()

	avatarsFolder := "avatars"

	fileURL, err := storage.UploadFileToS3(file, avatarsFolder, fileHeader, keys.BUCKET_NAME, keys.AWS_REGION, keys.AWS_ACCESS_KEY, keys.AWS_SECRET_KEY)

	if err != nil {
		log.Error().Err(err).Msg("Failed to upload file to S3")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	email := r.FormValue("email")
	username := r.FormValue("username")
	fullname := r.FormValue("fullName")
	companyIdVal := r.FormValue("companyId")

	var companyId *int

	if companyIdVal != "" {
		companyIdInt, err := strconv.Atoi(companyIdVal)
		if err != nil {
			log.Error().Err(err).Msg("Failed to convert companyId to int")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		companyId = &companyIdInt
	}

	reqBody = entities.User{
		Id:        id,
		Email:     email,
		Username:  username,
		Fullname:  fullname,
		CompanyId: companyId,
		AvatarUrl: fileURL,
	}

	err = res.s.UpdateUser(id, reqBody)

	if err != nil {
		log.Error().Err(err).Msg("Failed to update user")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "User updated successfully", "photoURL": fileURL})
	err = res.s.UpdateUser(id, reqBody)

	if err != nil {
		log.Error().Err(err).Msg("Failed to update user")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

func (res *Resourse) DeleteUser(w http.ResponseWriter, r *http.Request) {
	cors.EnableCors(&w)

	idVal := r.PathValue("id")
	id, err := strconv.Atoi(idVal)

	if err != nil {
		log.Error().Err(err).Msg("Failed to convert id to integer")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = res.s.DeleteUser(id)

	if err != nil {
		log.Error().Err(err).Msg("Failed to delete user")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

//articles

func (res *Resourse) GetArticles(w http.ResponseWriter, r *http.Request) {
	cors.EnableCors(&w)

	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS, POST, PUT, PATCH")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	articles, err := res.s.GetArticles("newest")

	if err != nil {
		log.Error().Err(err).Msg("Failed to get articles")
		return
	}

	err = json.NewEncoder(w).Encode(articles)

	if err != nil {
		log.Error().Err(err).Msg("Failed to encode")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (res *Resourse) CreateArticle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, PUT")

	cors.EnableCors(&w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		log.Error().Err(err).Msg("Error parsing form data")
		http.Error(w, "Can't parse form", http.StatusBadRequest)
		return
	}

	title := r.FormValue("title")
	text := r.FormValue("text")
	userId, err := strconv.Atoi(r.FormValue("userId"))

	if err != nil {
		log.Error().Err(err).Msg("Failed to conver string to int")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	file, fileHeader, err := r.FormFile("coverUrl")

	if err != nil {
		log.Error().Err(err).Msg("Failed to get file from form")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	defer file.Close()

	fileURL, err := storage.UploadFileToS3(file, "covers", fileHeader, keys.BUCKET_NAME, keys.AWS_REGION, keys.AWS_ACCESS_KEY, keys.AWS_SECRET_KEY)

	if err != nil {
		log.Error().Err(err).Msg("Failed to upload file to S3")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	article := entities.Article{
		Title:    title,
		Text:     text,
		AuthorId: userId,
		CoverUrl: fileURL,
	}

	err = res.s.InsertArticle(article)

	if err != nil {
		log.Error().Err(err).Msg("Failed to create article")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (res *Resourse) GetArticleById(w http.ResponseWriter, r *http.Request) {
	cors.EnableCors(&w)

	idVal := r.PathValue("id")

	id, err := strconv.Atoi(idVal)

	if err != nil {
		log.Error().Err(err).Msg("Failed to convert id to integer")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	article, err := res.s.GetArticleById(id)

	if err != nil {
		log.Error().Err(err).Msg("Failed to get article by id")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = json.NewEncoder(w).Encode(article)

	if err != nil {
		log.Error().Err(err).Msg("Failed to encode")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (res *Resourse) GetArticlesByAuthorId(w http.ResponseWriter, r *http.Request) {
	cors.EnableCors(&w)

	idVal := r.PathValue("id")

	id, err := strconv.Atoi(idVal)

	if err != nil {
		log.Error().Err(err).Msg("Failed to convert id to number")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	articles, err := res.s.GetArticlesByAuthorId(id)

	if err != nil {
		log.Error().Err(err).Msg("Failed to get articles by user id")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = json.NewEncoder(w).Encode(articles)

	if err != nil {
		log.Error().Err(err).Msg("Failed to encode")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (res *Resourse) GetArticlesByCompanyId(w http.ResponseWriter, r *http.Request) {
	idVal := r.PathValue("id")

	id, err := strconv.Atoi(idVal)

	if err != nil {
		log.Error().Err(err).Msg("Failed to convert id to number")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	articles, err := res.s.GetArticlesByCompanyId(id)

	if err != nil {
		log.Error().Err(err).Msg("Failed to get articles by company id")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = json.NewEncoder(w).Encode(articles)

	if err != nil {
		log.Error().Err(err).Msg("Failed to encode")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (res *Resourse) UpdateArticle(w http.ResponseWriter, r *http.Request) {
	cors.EnableCors(&w)

	idVal := r.PathValue("id")
	var reqBody entities.Article

	id, err := strconv.Atoi(idVal)

	if err != nil {
		log.Error().Err(err).Msg("Failed to convert id to integer")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.NewDecoder(r.Body).Decode(&reqBody)

	if err != nil {
		log.Error().Err(err).Msg("Failed to decode")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = res.s.UpdateArticle(id, reqBody)

	if err != nil {
		log.Error().Err(err).Msg("Failed to update article")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

func (res *Resourse) DeleteArticle(w http.ResponseWriter, r *http.Request) {
	cors.EnableCors(&w)

	idVal := r.PathValue("id")
	id, err := strconv.Atoi(idVal)

	if err != nil {
		log.Error().Err(err).Msg("Failed to convert id to integer")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = res.s.DeleteArticle(id)

	if err != nil {
		log.Error().Err(err).Msg("Failed to delete article")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

func (res *Resourse) GetCompanies(w http.ResponseWriter, r *http.Request) {
	cors.EnableCors(&w)

	companies, err := res.s.GetCompanies()

	if err != nil {
		log.Error().Err(err).Msg("Failed to get companies")
		return
	}

	err = json.NewEncoder(w).Encode(companies)

	if err != nil {
		log.Error().Err(err).Msg("Failed to encode")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (res *Resourse) CreateCompany(w http.ResponseWriter, r *http.Request) {
	cors.EnableCors(&w)

	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		log.Error().Err(err).Msg("Error parsing form data")
		http.Error(w, "Can't parse form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")
	position := r.FormValue("position")
	website := r.FormValue("website")
	userId, err := strconv.Atoi(r.FormValue("userId"))

	if err != nil {
		log.Error().Err(err).Msg("Failed to conver string to int")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	file, fileHeader, err := r.FormFile("logoUrl")

	if err != nil {
		log.Error().Err(err).Msg("Failed to get file from form")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	defer file.Close()

	fileURL, err := storage.UploadFileToS3(file, "logos", fileHeader, keys.BUCKET_NAME, keys.AWS_REGION, keys.AWS_ACCESS_KEY, keys.AWS_SECRET_KEY)

	if err != nil {
		log.Error().Err(err).Msg("Failed to upload file to S3")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	bytes := make([]byte, 20)

	_, err = rand.Read(bytes)

	if err != nil {
		log.Error().Err(err).Msg("Failed to read bytes")
	}

	secretKey := base64.URLEncoding.EncodeToString(bytes)

	if len(secretKey) > 20 {
		secretKey = secretKey[:20]
	}

	company := entities.Company{
		Description: description,
		Website:     website,
		Name:        name,
		Key:         secretKey,
		LogoUrl:     fileURL,
	}

	err = res.s.InsertCompany(company, userId, position)

	if err != nil {
		log.Error().Err(err).Msg("Failed to create company")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (res *Resourse) GetCompanyById(w http.ResponseWriter, r *http.Request) {
	cors.EnableCors(&w)

	idVal := r.PathValue("id")

	id, err := strconv.Atoi(idVal)

	if err != nil {
		log.Error().Err(err).Msg("Failed to convert id to integer")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	company, err := res.s.GetCompanyById(id)

	if err != nil {
		log.Error().Err(err).Msg("Failed to get company by id")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = json.NewEncoder(w).Encode(company)

	if err != nil {
		log.Error().Err(err).Msg("Failed to encode")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

type JoinCompanyRequest struct {
	UserId   int    `json:"userId"`
	Key      string `json:"key"`
	Position string `json:"position"`
}

func (res *Resourse) JoinCompany(w http.ResponseWriter, r *http.Request) {
	cors.EnableCors(&w)

	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var reqBody JoinCompanyRequest

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	company, err := res.s.GetCompanyByKey(reqBody.Key)

	if err != nil {
		log.Error().Err(err).Msg("Failed to get company by its key")
	}

	if err := res.s.UpdateUserCompanyInfo(reqBody.UserId, company.Id, reqBody.Position); err != nil {
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (res *Resourse) UpdateCompanyLogo(w http.ResponseWriter, r *http.Request) {
	cors.EnableCors(&w)

	w.Header().Set("Access-Control-Allow-Methods", "PUT, OPTIONS")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPut {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20)

	if err != nil {
		log.Error().Err(err).Msg("Failed to parse multipart form")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	idVal := r.PathValue("id")

	id, err := strconv.Atoi(idVal)

	if err != nil {
		log.Error().Err(err).Msg("Failed to convert id to integer")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	file, fileHeader, err := r.FormFile("photo")

	if err != nil {
		log.Error().Err(err).Msg("Failed to get file from form")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	defer file.Close()

	avatarsFolder := "logos"

	fileURL, err := storage.UploadFileToS3(file, avatarsFolder, fileHeader, keys.BUCKET_NAME, keys.AWS_REGION, keys.AWS_ACCESS_KEY, keys.AWS_SECRET_KEY)

	if err != nil {
		log.Error().Err(err).Msg("Failed to upload file to S3")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = res.s.UpdateCompanyLogo(fileURL, id)

	if err != nil {
		log.Error().Err(err).Msg("Failed to update user's photo URL")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "User photo updated successfully", "photoURL": fileURL})
}

func (res *Resourse) UpdateCompany(w http.ResponseWriter, r *http.Request) {
	cors.EnableCors(&w)

	idVal := r.PathValue("id")
	var reqBody entities.Company

	id, err := strconv.Atoi(idVal)

	if err != nil {
		log.Error().Err(err).Msg("Failed to convert id to integer")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.NewDecoder(r.Body).Decode(&reqBody)

	if err != nil {
		log.Error().Err(err).Msg("Failed to decode")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = res.s.UpdateCompany(id, reqBody)

	if err != nil {
		log.Error().Err(err).Msg("Failed to update company")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

func (res *Resourse) DeleteCompany(w http.ResponseWriter, r *http.Request) {
	cors.EnableCors(&w)

	idVal := r.PathValue("id")
	id, err := strconv.Atoi(idVal)

	if err != nil {
		log.Error().Err(err).Msg("Failed to convert id to integer")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = res.s.DeleteCompany(id)

	if err != nil {
		log.Error().Err(err).Msg("Failed to delete company")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}
