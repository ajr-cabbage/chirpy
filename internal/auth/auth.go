package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func HashPassword(password string) (string, error) {
	hashPwd, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", err
	}

	return hashPwd, nil
}

func CheckPasswordHash(password, hash string) (bool, error) {
	pwMatch, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false, err
	}

	return pwMatch, nil
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	claims := jwt.RegisteredClaims{
		Issuer:    "chirpy-access",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
		Subject:   userID.String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := jwt.RegisteredClaims{}

	_, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (any, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.UUID{}, err
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.UUID{}, err
	}

	return userID, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	tokenString := headers.Get("Authorization")
	if tokenString == "" {
		return "", errors.New("No auth token provided")
	}
	authHeaderParts := strings.SplitN(tokenString, " ", 2)
	if len(authHeaderParts) < 2 {
		return "", errors.New("Auth token bad format")
	}

	return authHeaderParts[1], nil
}

func MakeRefreshToken() string {
	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)

	return hex.EncodeToString(tokenBytes)
}

func GetAPIKey(headers http.Header) (string, error) {
	authHeaderString := headers.Get("Authorization")
	if authHeaderString == "" {
		return "", errors.New("No auth token provided")
	}
	authHeaderParts := strings.SplitN(authHeaderString, " ", 2)
	if len(authHeaderParts) < 2 {
		return "", errors.New("Auth token bad format")
	}

	return authHeaderParts[1], nil
}
