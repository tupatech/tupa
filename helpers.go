package tupa

import (
	"crypto/rand"
	"math/big"
)

func GenerateRandomStringHelper(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	charsetLen := big.NewInt(int64(len(charset)))

	randomString := make([]byte, length)
	for i := range randomString {
		randomIndex, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", err
		}
		randomString[i] = charset[randomIndex.Int64()]
	}
	return string(randomString), nil
}
