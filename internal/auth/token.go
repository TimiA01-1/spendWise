package auth

import(
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const tokenIssuer = "spendwise"

var ErrInvalidToken = errors.New("invalid or expired token")

// token manager signs and verifies HS256 JWTs
// the user ID is stored in the standard "sub" subject claim

type TokenManager struct{
	secret []byte
	ttl time.Duration
}

func NewTokenManager(secret string, ttl time.Duration) *TokenManager{
	return &TokenManager{secret: []byte(secret), ttl: ttl}
}

// generate returns a signed token and the time it expires
func (m *TokenManager) Generate(userID uuid.UUID)(string, time.Time, error){
	now := time.Now()
	exp := now.Add(m.ttl)

	claims := jwt.RegisteredClaims{
		Issuer: tokenIssuer,
		Subject: userID.String(),
		IssuedAt: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(exp),
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil{
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return signed, exp, nil
}

// parse checks the signature, algorithm, issuer and expiry and returns the user ID.
func (m *TokenManager) Parse(tokenStr string)(uuid.UUID, error){
	claims := &jwt.RegisteredClaims{}

	_, err := jwt.ParseWithClaims(
		tokenStr,
		claims,
		func(*jwt.Token)(any, error){ return m.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(tokenIssuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil{
		return uuid.Nil, ErrInvalidToken
	}
	id, err := uuid.Parse(claims.Subject)
	if err != nil{
		return uuid.Nil, ErrInvalidToken
	}
	return id, nil
}