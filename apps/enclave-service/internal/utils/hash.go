package utils

import "golang.org/x/crypto/bcrypt"

func Hash(data []byte, salt string) string {
	hashedData, _ := bcrypt.GenerateFromPassword(append(data, []byte(salt)...), bcrypt.DefaultCost)
	return string(hashedData)
}
