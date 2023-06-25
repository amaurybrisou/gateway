package core

import (
	"os"
	"strconv"

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
