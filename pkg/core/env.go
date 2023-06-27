package core

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

func LookupEnv(e, d string) (r string) {
	r = os.Getenv(e)
	if r == "" {
		return d
	}
	return r
}

func LookupEnvInt(e string, d int) (r int) {
	tr := os.Getenv(e)
	if tr == "" {
		return d
	}

	r, err := strconv.Atoi(tr)
	if err != nil {
		log.Fatal().Err(err).Msg("strconv.Atoi()")
		return -1
	}
	return r
}

func init() {
	file, err := os.Open(".env")
	if err != nil {
		fmt.Println("loading environment: ", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) > 0 && !strings.HasPrefix(line, "#") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				os.Setenv(key, value)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}
}
