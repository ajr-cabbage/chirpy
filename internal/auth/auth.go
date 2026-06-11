package auth

import "github.com/alexedwards/argon2id"

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
