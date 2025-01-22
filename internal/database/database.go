package database

import (
	"auth-service/internal/entities"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(connString string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", connString)

	if err != nil {
		return nil, fmt.Errorf("opening database: %v", err)
	}

	return &PostgresStorage{
		db: db,
	}, nil
}

type UsersResource struct {
	Storage *PostgresStorage
}

//users

func (s *PostgresStorage) GetUsers() ([]entities.User, error) {
	rows, err := s.db.Query("SELECT id, email, username, fullname, position, company_id, avatar_url FROM users")
	if err != nil {
		return nil, fmt.Errorf("querying users: %v", err)
	}
	defer rows.Close()

	var users []entities.User

	for rows.Next() {
		var user entities.User
		var companyId sql.NullInt64

		err := rows.Scan(&user.Id, &user.Email, &user.Username, &user.Fullname, &user.Position, &companyId, &user.AvatarUrl)
		if err != nil {
			return nil, fmt.Errorf("scanning rows: %v", err)
		}

		if companyId.Valid {
			companyIdInt := int(companyId.Int64)
			user.CompanyId = &companyIdInt
		} else {
			user.CompanyId = nil
		}

		users = append(users, user)
	}

	return users, nil
}

func (s *PostgresStorage) InsertUser(user entities.User) (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("beginning transaction: %v", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var userId int

	err = tx.QueryRow("INSERT INTO users(email, username, password, fullname) VALUES($1, $2, $3, $4) RETURNING id", user.Email, user.Username, user.Password, user.Fullname).Scan(&userId)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" {
				if pqErr.Constraint == "users_email_key" {
					return 0, errors.New("email already exists")
				} else if pqErr.Constraint == "users_username_key" {
					return 0, errors.New("username already exists")
				}
			}
		}

		return 0, fmt.Errorf("running transaction: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		return 0, fmt.Errorf("committing transaction: %v", err)
	}

	return userId, nil
}

func (s *PostgresStorage) GetUserById(id int) (entities.User, error) {
	rows, err := s.db.Query("SELECT id, email, username, fullname, position, company_id, avatar_url FROM users WHERE id = $1", id)

	if err != nil {
		return entities.User{}, fmt.Errorf("getting user by id: %v", err)
	}

	defer rows.Close()

	var user entities.User

	if rows.Next() {
		err := rows.Scan(&user.Id, &user.Email, &user.Username, &user.Fullname, &user.Position, &user.CompanyId, &user.AvatarUrl)

		if err != nil {
			return entities.User{}, fmt.Errorf("scanning rows: %v", err)
		}
	} else {
		return entities.User{}, fmt.Errorf("user not found")
	}

	return user, nil
}

func (s *PostgresStorage) GetUserByUsername(username string) (entities.User, error) { //TODO: get whole user or passworl only
	rows, err := s.db.Query("SELECT * FROM users WHERE username = $1", username)

	if err != nil {
		return entities.User{}, fmt.Errorf("getting user: %v", err)
	}

	defer rows.Close()

	var user entities.User

	if rows.Next() {
		err := rows.Scan(&user.Id, &user.Username, &user.Email, &user.Password, &user.Fullname, &user.CompanyId, &user.Position, &user.AvatarUrl)

		if err != nil {
			return entities.User{}, fmt.Errorf("scanning rows: %v", err)
		}
	} else {
		return entities.User{}, fmt.Errorf("user not found")
	}

	return user, nil
}

func (s *PostgresStorage) UpdateUserCompanyInfo(userId int, companyId int, position string) error {
	_, err := s.db.Exec("UPDATE users SET company_id = $1, position = $2 WHERE id = $3", companyId, position, userId)

	if err != nil {
		return fmt.Errorf("updating user company info: %v", err)
	}

	return nil
}

func (s *PostgresStorage) UpdateUser(id int, user entities.User) error {
	_, err := s.db.Exec("UPDATE users SET email = $1, username = $2, fullname = $3, company_id = $4, avatar_url = $5 WHERE id = $6", user.Email, user.Username, user.Fullname, user.CompanyId, user.AvatarUrl, id)

	if err != nil {
		return fmt.Errorf("updating user: %v", err)
	}

	return nil
}

func (s *PostgresStorage) UpdateUserPhoto(photoUrl string, id int) error {
	_, err := s.db.Exec("UPDATE users SET avatar_url = $1 WHERE id = $2", photoUrl, id)

	if err != nil {
		return fmt.Errorf("updating user: %v", err)
	}

	return nil
}

func (s *PostgresStorage) DeleteUser(id int) error {
	_, err := s.db.Exec("DELETE FROM users WHERE id = $1", id)

	if err != nil {
		return fmt.Errorf("deleting user %v", err)
	}

	return nil
}

// articles
func (s *PostgresStorage) GetArticles(sortBy string) ([]entities.Article, error) {
	// var orderBy string

	// switch sortBy {
	// case "newest":
	// 	orderBy = "DESC"
	// case "oldest":
	// 	orderBy = "ASC"
	// }

	// rows, err := s.db.Query("SELECT id, author_id, company_id, title, text, rating FROM articles ORDER_BY created_at $1", orderBy)
	rows, err := s.db.Query("SELECT id, author_id, company_id, title, text, rating FROM articles")

	if err != nil {
		return nil, fmt.Errorf("querying articles: %v", err)
	}

	defer rows.Close()

	var articles []entities.Article

	for rows.Next() {
		var article entities.Article

		err := rows.Scan(&article.Id, &article.AuthorId, &article.CompanyId, &article.Title, &article.Text, &article.Rating)

		if err != nil {
			return nil, fmt.Errorf("scanning rows: %v", err)
		}

		articles = append(articles, article)
	}

	return articles, nil

}

func (s *PostgresStorage) InsertArticle(article entities.Article) error {
	if article.CompanyId != 0 {
		_, err := s.db.Exec("INSERT INTO articles(author_id, company_id, title, text, rating, cover_url) VALUES ($1, $2, $3, $4, $5, $6)", article.AuthorId, article.CompanyId, article.Title, article.Text, article.Rating, article.CoverUrl)

		if err != nil {
			return fmt.Errorf("inserting article: %v", err)
		}

	} else {
		_, err := s.db.Exec("INSERT INTO articles(author_id, title, text, rating, cover_url) VALUES ($1, $2, $3, $4, $5)", article.AuthorId, article.Title, article.Text, article.Rating, article.CoverUrl)

		if err != nil {
			return fmt.Errorf("inserting article: %v", err)
		}
	}

	return nil
}

func (s *PostgresStorage) GetArticleById(id int) (entities.Article, error) {
	rows, err := s.db.Query("SELECT * FROM articles WHERE id = $1", id)

	if err != nil {
		return entities.Article{}, fmt.Errorf("getting article by id: %v", err)
	}

	defer rows.Close()

	var article entities.Article

	if rows.Next() {
		err := rows.Scan(&article.Id, &article.AuthorId, &article.Title, &article.Text, &article.Rating, &article.CreatedAt)

		if err != nil {
			return entities.Article{}, fmt.Errorf("scanning rows: %v", err)
		}
	} else {
		return entities.Article{}, fmt.Errorf("article not found")
	}

	return article, nil
}

func (s *PostgresStorage) GetArticlesByAuthorId(authorId int) ([]entities.Article, error) {
	rows, err := s.db.Query("SELECT * FROM articles WHERE author_id = $1", authorId)

	if err != nil {
		return nil, fmt.Errorf("getting articles by authorId: %v", err)
	}

	defer rows.Close()

	var articles []entities.Article

	for rows.Next() {
		var article entities.Article

		err := rows.Scan(&article.Id, &article.AuthorId, &article.CompanyId, &article.Title, &article.Text, &article.Rating)

		if err != nil {
			return nil, fmt.Errorf("scanning rows: %v", err)
		}

		articles = append(articles, article)
	}

	return articles, nil
}

func (s *PostgresStorage) GetArticlesByCompanyId(authorId int) ([]entities.Article, error) {
	rows, err := s.db.Query("SELECT * FROM articles WHERE company_id = $1", authorId)

	if err != nil {
		return nil, fmt.Errorf("getting articles by companyId: %v", err)
	}

	defer rows.Close()

	var articles []entities.Article

	for rows.Next() {
		var article entities.Article

		err := rows.Scan(&article.Id, &article.AuthorId, &article.CompanyId, &article.Title, &article.Text, &article.Rating)

		if err != nil {
			return nil, fmt.Errorf("scanning rows: %v", err)
		}

		articles = append(articles, article)
	}

	return articles, nil
}

func (s *PostgresStorage) UpdateArticle(id int, article entities.Article) error {
	_, err := s.db.Exec("UPDATE articles SET id = $1, author_id = $2, company_id = $3, title = $4, text = $5, rating = $6 WHERE id = $7", article.Id, article.AuthorId, article.CompanyId, article.Title, article.Text, article.Rating, id)

	if err != nil {
		return fmt.Errorf("updating article: %v", err)
	}

	return nil
}

func (s *PostgresStorage) DeleteArticle(id int) error {
	_, err := s.db.Exec("DELETE FROM articles WHERE id = $1", id)

	if err != nil {
		return fmt.Errorf("deleting article: %v", err)
	}

	return nil
}

// companies
func (s *PostgresStorage) GetCompanies() ([]entities.Company, error) {
	rows, err := s.db.Query("SELECT id, name, description, website, logo_url FROM companies")

	if err != nil {
		return nil, fmt.Errorf("querying companies: %v", err)
	}

	defer rows.Close()

	var companies []entities.Company

	for rows.Next() {
		var company entities.Company

		err := rows.Scan(&company.Id, &company.Name, &company.Description, &company.LogoUrl)

		if err != nil {
			return nil, fmt.Errorf("scanning rows")
		}

		companies = append(companies, company)
	}

	return companies, nil
}

func (s *PostgresStorage) InsertCompany(company entities.Company, userId int, position string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %v", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var companyId int

	err = tx.QueryRow("INSERT INTO companies (name, key, description, website) VALUES ($1, $2, $3, $4) RETURNING id", company.Name, company.Key, company.Description, company.Website).Scan(&companyId)
	if err != nil {
		return fmt.Errorf("running transaction: %v", err)
	}

	log.Debug().Msgf("position: %v", position)

	_, err = tx.Exec("UPDATE users SET company_id = $1, position = $2 WHERE id = $3", companyId, position, userId)

	if err != nil {
		return fmt.Errorf("updating user: %v", err)
	}

	err = tx.Commit()

	if err != nil {
		return fmt.Errorf("committing transaction: %v", err)
	}

	return nil
}

func (s *PostgresStorage) GetCompanyById(id int) (entities.Company, error) {
	rows, err := s.db.Query("SELECT id, name, key FROM companies WHERE id = $1", id)

	if err != nil {
		return entities.Company{}, fmt.Errorf("getting company by id")
	}

	defer rows.Close()

	var company entities.Company

	if rows.Next() {
		err := rows.Scan(&company.Id, &company.Name, &company.Key)

		if err != nil {
			return entities.Company{}, fmt.Errorf("scanning rows: %v", err)
		}
	} else {
		return entities.Company{}, fmt.Errorf("company not found")
	}

	return company, nil
}

func (s *PostgresStorage) GetCompanyByKey(key string) (entities.Company, error) {
	rows, err := s.db.Query("SELECT * FROM companies WHERE key = $1", key)

	if err != nil {
		return entities.Company{}, fmt.Errorf("getting company by key")
	}
	defer rows.Close()

	var company entities.Company

	if rows.Next() {
		err := rows.Scan(&company.Id, &company.Name, &company.Key, &company.Description, &company.Website, &company.LogoUrl)

		if err != nil {
			return entities.Company{}, fmt.Errorf("scanning rows: %v", err)
		}
	} else {
		return entities.Company{}, fmt.Errorf("company not found")
	}

	return company, nil

}

func (s *PostgresStorage) UpdateCompanyLogo(logoUrl string, id int) error {
	_, err := s.db.Exec("UPDATE companies SET logo_url = $1 WHERE id = $2", logoUrl, id)

	if err != nil {
		return fmt.Errorf("updating company: %v", err)
	}

	return nil
}

func (s *PostgresStorage) UpdateCompany(id int, company entities.Company) error {
	_, err := s.db.Exec("UPDATE companies SET id = $1, name = $2, key = $3 WHERE id = $4", company.Id, company.Name, company.Key, id)

	if err != nil {
		return fmt.Errorf("updating company: %v", err)
	}

	return nil
}

func (s *PostgresStorage) DeleteCompany(id int) error {
	_, err := s.db.Exec("DELETE FROM companies WHERE id = $1", id)

	if err != nil {
		return fmt.Errorf("deleting company: %v", err)
	}

	return nil
}
