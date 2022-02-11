package main

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
)

func RandInt(min, max int) int {
	return rand.Intn(max-min+1) + min
}

func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}

func extractChromeID(chromeStdOutStr string) (string, error) {
	i := strings.Index(chromeStdOutStr, "/devtools")

	if i < 0 {
		return "", fmt.Errorf("fail to find ws")
	}

	chromeStdOutStr = chromeStdOutStr[i:]

	i = strings.Index(chromeStdOutStr, "\n")

	if i < 0 {
		return "", fmt.Errorf("invalid ws format")
	}

	chromeStdOutStr = chromeStdOutStr[:i]

	return chromeStdOutStr, nil
}
