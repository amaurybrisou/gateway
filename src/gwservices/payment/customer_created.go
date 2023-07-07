package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/amaurybrisou/ablib/cryptlib"
	coremodels "github.com/amaurybrisou/ablib/models"
	"github.com/amaurybrisou/gateway/src/database/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/stripe/stripe-go/v72"
	"golang.org/x/crypto/bcrypt"
)

func (s Service) customerCreated(ctx context.Context, event json.RawMessage) (models.User, error) {
	var customer stripe.Customer
	err := json.Unmarshal(event, &customer)
	if err != nil {
		return models.User{}, fmt.Errorf("creating user: %w", err)
	}

	user, err := s.RegisterUser(ctx, &customer)
	if err != nil {
		return models.User{}, fmt.Errorf("creating user: %w", err)
	}

	return user, nil
}

func (s Service) RegisterUser(ctx context.Context, customer *stripe.Customer) (models.User, error) {
	password, err := cryptlib.GenerateRandomPassword(16)
	if err != nil {
		return models.User{}, err
	}

	fmt.Println(strings.Repeat("#", 100))
	fmt.Println("Password:", password)
	fmt.Println(strings.Repeat("#", 100))

	hashedPassword, err := cryptlib.GenerateHash(password, bcrypt.DefaultCost)
	if err != nil {
		return models.User{}, err
	}

	u := models.User{
		ID:         uuid.New(),
		ExternalID: customer.ID,
		Email:      customer.Email,
		Firstname:  customer.Name,
		Password:   hashedPassword,
		Role:       coremodels.USER,
		CreatedAt:  time.Now(),
	}

	u, err = s.db.CreateUser(ctx, u)
	if err != nil {
		return u, err
	}

	go func() {
		if s.mailcli != nil {
			err := s.mailcli.SendPasswordEmail(u.Email, hashedPassword)
			if err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("error sending auto generated password email")
			}
			log.Ctx(ctx).Debug().Any("email", u.Email).Msg("auto generated pasword email send")
		}
	}()

	return u, nil
}
