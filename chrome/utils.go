package chrome

import (
	"bufio"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

func GetChromeIDFromStdout(reader *bufio.Reader) string {
	ticker := time.NewTicker(1 * time.Minute)

	for {
		select {
		case <-ticker.C:
			log.Error().Msg("time out for read chrome's stdout")
			return ""
		default:
			str, err := reader.ReadString('\n')
			if err != nil {
				log.Error().Err(err).Msg("fail to read chrome's stdout")
				continue
			}

			id, err := extractChromeID(str)
			if err != nil {
				continue
			}

			return id
		}
	}
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
