package auth

import (
	"auth-service/internal/keys"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func CreateToken(username string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"username": username,
			"exp":      time.Now().Add(time.Hour * 48).Unix(),
		})

	tokenString, err := token.SignedString(keys.JWT_SECRET_KEY)

	if err != nil {
		return "", fmt.Errorf("failed to sign string: %v", err)
	}

	return tokenString, nil
}

func VerifyToken(token string) error {
	jwt, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return keys.JWT_SECRET_KEY, nil
	})

	if err != nil {
		return err
	}

	if !jwt.Valid {
		return fmt.Errorf("invalid token")
	}

	return nil
}

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashedPassword), nil
}

func CheckPasswordHash(password, hashedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}
