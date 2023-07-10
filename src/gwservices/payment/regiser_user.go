package payment

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/amaurybrisou/ablib"
	"github.com/amaurybrisou/ablib/cryptlib"
	coremodels "github.com/amaurybrisou/ablib/models"
	"github.com/amaurybrisou/gateway/src/database/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

func (s Service) RegisterUser(ctx context.Context, id, email, name string) (models.User, error) {
	password, err := cryptlib.GenerateRandomPassword(16)
	if err != nil {
		return models.User{}, err
	}

	env := ablib.LookupEnv("ENV", "dev")
	if env == "dev" {
		fmt.Println(strings.Repeat("#", 100))
		fmt.Println("Password:", password)
		fmt.Println(strings.Repeat("#", 100))
	}

	hashedPassword, err := cryptlib.GenerateHash(password, bcrypt.DefaultCost)
	if err != nil {
		return models.User{}, err
	}

	u := models.User{
		ID:         uuid.New(),
		ExternalID: id,
		Email:      email,
		Firstname:  name,
		Password:   hashedPassword,
		Role:       coremodels.USER,
		CreatedAt:  time.Now(),
	}

	u, err = s.db.CreateUser(ctx, u)
	if err != nil {
		return u, err
	}

	go func() {
		if s.mailcli != nil && env == "prod" {
			err := s.mailcli.SendPasswordEmail(u.Email, hashedPassword)
			if err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("error sending auto generated password email")
				fmt.Println(strings.Repeat("#", 100))
				fmt.Println("Password:", password)
				fmt.Println(strings.Repeat("#", 100))
				return
			}
			log.Ctx(ctx).Debug().Any("email", u.Email).Msg("auto generated password email sent")
		}
	}()

	return u, nil
}
