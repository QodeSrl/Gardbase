package utils

import "golang.org/x/crypto/bcrypt"

func Hash(data []byte, salt []byte) string {
	hashedData, _ := bcrypt.GenerateFromPassword(append(data, salt...), bcrypt.DefaultCost)
	return string(hashedData)
}
