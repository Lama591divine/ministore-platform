package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	defaultIssuer = "ministore-auth"
	expLeeway     = 30 * time.Second
)

var (
	ErrUnexpectedAlg  = errors.New("unexpected signing method")
	ErrInvalidToken   = errors.New("invalid token")
	ErrInvalidIssuer  = errors.New("invalid issuer")
	ErrMissingExp     = errors.New("missing exp")
	ErrTokenExpired   = errors.New("token expired")
	ErrMissingUserID  = errors.New("missing user_id")
	ErrInvalidSubject = errors.New("invalid subject")
)

type TokenMaker struct {
	secret []byte
	issuer string
}

func NewTokenMaker(secret string) *TokenMaker {
	return &TokenMaker{
		secret: []byte(secret),
		issuer: defaultIssuer,
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

func (t *TokenMaker) Parse(tokenString string) (Claims, error) {
	var c Claims

	token, err := jwt.ParseWithClaims(tokenString, &c, t.keyFunc())
	if err != nil || token == nil || !token.Valid {
		return Claims{}, ErrInvalidToken
	}

	if c.Issuer != t.issuer {
		return Claims{}, ErrInvalidIssuer
	}

	if c.ExpiresAt == nil {
		return Claims{}, ErrMissingExp
	}

	if time.Now().After(c.ExpiresAt.Time.Add(expLeeway)) {
		return Claims{}, ErrTokenExpired
	}

	if c.UserID == "" {
		return Claims{}, ErrMissingUserID
	}

	if c.Subject != "" && c.Subject != c.UserID {
		return Claims{}, ErrInvalidSubject
	}

	return c, nil
}

func (t *TokenMaker) keyFunc() jwt.Keyfunc {
	return func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, ErrUnexpectedAlg
		}
		return t.secret, nil
	}
}
