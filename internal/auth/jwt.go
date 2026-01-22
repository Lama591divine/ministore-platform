package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenMaker struct {
	secret []byte
}

func NewTokenMaker(secret string) *TokenMaker {
	return &TokenMaker{secret: []byte(secret)}
}

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func (t *TokenMaker) New(userID, email, role string, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(t.secret)
}

func (t *TokenMaker) Parse(tokenStr string) (Claims, error) {
	var c Claims
	token, err := jwt.ParseWithClaims(tokenStr, &c, func(token *jwt.Token) (any, error) {
		return t.secret, nil
	})
	if err != nil || !token.Valid {
		return Claims{}, errors.New("invalid token")
	}
	return c, nil
}
