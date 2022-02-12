package utils

import (
	"math/rand"
	"os"
)

func RandInt(min, max int) int {
	return rand.Intn(max-min+1) + min
}

func GetEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}
