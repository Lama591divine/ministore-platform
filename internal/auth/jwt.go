package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenMaker struct {
	secret []byte
	issuer string
}

func NewTokenMaker(secret string) *TokenMaker {
	return &TokenMaker{
		secret: []byte(secret),
		issuer: "ministore-auth",
	}
}

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func (t *TokenMaker) New(userID, email, role string, ttl time.Duration) (string, error) {
	now := time.Now()

	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    t.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(t.secret)
}

func (t *TokenMaker) Parse(tokenStr string) (Claims, error) {
	var c Claims

	token, err := jwt.ParseWithClaims(tokenStr, &c, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return t.secret, nil
	})
	if err != nil || token == nil || !token.Valid {
		return Claims{}, errors.New("invalid token")
	}

	if c.Issuer != "" && c.Issuer != t.issuer {
		return Claims{}, errors.New("invalid issuer")
	}

	return c, nil
}
