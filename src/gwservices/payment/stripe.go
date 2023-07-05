package payment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/amaurybrisou/ablib/cryptlib"
	"github.com/amaurybrisou/ablib/jwtlib"
	"github.com/amaurybrisou/ablib/mailcli"
	coremodels "github.com/amaurybrisou/ablib/models"
	"github.com/amaurybrisou/gateway/src/database"
	"github.com/amaurybrisou/gateway/src/database/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/client"
	"github.com/stripe/stripe-go/v72/webhook"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	*client.API
	db            *database.Database
	jwt           *jwtlib.JWT
	mailcli       *mailcli.MailClient
	stripeKey     string
	successURL    string
	webHookSecret string
	cancelURL     string
}

type Config struct {
	StripeKey, StripeSuccessURL, StripeCancelURL, StripeWebHookSecret string
}

func NewService(db *database.Database, jwt *jwtlib.JWT, mail *mailcli.MailClient, cfg Config) Service {
	stripe.Key = cfg.StripeKey

	// stripeClient := &client.API{}
	// stripeClient.Init(cfg.StripeKey, nil)

	return Service{
		db:            db,
		jwt:           jwt,
		mailcli:       mail,
		stripeKey:     cfg.StripeKey,
		successURL:    cfg.StripeSuccessURL,
		cancelURL:     cfg.StripeCancelURL,
		webHookSecret: cfg.StripeWebHookSecret,
	}
}

func (s Service) StripeWebhook(w http.ResponseWriter, r *http.Request) {
	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("failed to read request body")
		http.Error(w, "failed to read request body", http.StatusInternalServerError)
		return
	}

	// Verify and parse the webhook event
	event, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), s.webHookSecret)
	if err != nil {
		log.Error().Err(err).Msg("failed to verify webhook event")
		http.Error(w, "failed to verify webhook event", http.StatusBadRequest)
		return
	}

	var sessionEvent stripe.Event
	err = json.Unmarshal(event.Data.Raw, &sessionEvent)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to unmarshal checkout session event")
		http.Error(w, "failed to process webhook event", http.StatusInternalServerError)
		return
	}

	// https://stripe.com/docs/api/events/types.
	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		user, err := s.RegisterUser(r.Context(), &session)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		serviceIDstring := session.ClientReferenceID
		serviceID, err := uuid.Parse(serviceIDstring)
		if err != nil {
			log.Error().Err(err).Msg("failed parse client_reference_id")
			http.Error(w, "failed parse client_reference_id", http.StatusInternalServerError)
			return
		}

		service, err := s.db.GetServiceByID(r.Context(), serviceID)
		if err != nil {
			log.Error().Err(err).Msg("failed to fetch service")
			http.Error(w, "failed to fetch service", http.StatusInternalServerError)
			return
		}

		subID := ""
		if session.Subscription != nil {
			subID = session.Subscription.ID
		}

		role := models.EmptyRole
		if len(service.RequiredRoles) > 0 {
			role = service.RequiredRoles[0]
		}

		_, err = s.db.AddRole(r.Context(), user.ID, subID, role, nil)
		if err != nil {
			log.Error().Err(err).
				Any("service", service).
				Any("user_id", user.ID).
				Msg("failed to add user roles")
			http.Error(w, "failed to add user roles", http.StatusInternalServerError)
			return
		}

		// Payment is successful and the subscription is created.
		// You should provision the subscription and save the customer ID to your database.
	case "customer.subscription.deleted":
		var sub stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &sub)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = s.DeleteRole(r.Context(), &sub)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

	case "customer.subscription.updated":
		fmt.Fprintf(os.Stderr, "Unhandled event type: %s\n", event.Type)
		var sub stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &sub)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		switch sub.PauseCollection.Behavior {
		case "":
			_, err = s.db.UpdateRoleExpiration(r.Context(), sub.ID, nil)
			if err != nil {
				log.Error().Err(err).
					Any("subscription_id", sub.ID).
					Msg("failed to update user role")
				http.Error(w, "failed to update user roles", http.StatusInternalServerError)
				return
			}
		default:
			fmt.Println(sub.PauseCollection.Behavior)
			currentPeriodendAt := time.Unix(sub.CurrentPeriodEnd, 0)
			_, err = s.db.UpdateRoleExpiration(r.Context(), sub.ID, &currentPeriodendAt)
			if err != nil {
				log.Error().Err(err).
					Any("subscription_id", sub.ID).
					Any("ends_at", currentPeriodendAt).
					Msg("failed to update user role")
				http.Error(w, "failed to update user roles", http.StatusInternalServerError)
				return
			}
		}

	case "subscription_schedule.canceled":
		fmt.Fprintf(os.Stderr, "Unhandled event type: %s\n", event.Type)
		var subSchedule stripe.SubscriptionSchedule
		err := json.Unmarshal(event.Data.Raw, &subSchedule)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = s.DeleteRole(r.Context(), subSchedule.Subscription)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	default:
		fmt.Fprintf(os.Stderr, "Unhandled event type: %s\n", event.Type)
	}

	// Return a success response to Stripe
	w.WriteHeader(http.StatusOK)
}

func (s Service) DeleteRole(ctx context.Context, sub *stripe.Subscription) error {
	deleted, err := s.db.DelRoleBySubscriptionID(ctx, sub.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)

		return err
	}

	if !deleted {
		log.Ctx(ctx).Debug().
			Err(errors.New("role could not be deleted")).
			Str("subscription_id", sub.ID).
			Send()
		return errors.New("role could not be deleted")
	}

	return nil
}

func (s Service) RegisterUser(ctx context.Context, session *stripe.CheckoutSession) (models.User, error) {
	u, err := s.db.GetFullUserByEmail(ctx, session.CustomerDetails.Email)
	if err != nil {
		return u, err
	}

	if u.ID != uuid.Nil {
		return u, nil
	}

	password, err := cryptlib.GenerateRandomPassword(16)
	if err != nil {
		return u, err
	}

	fmt.Println(strings.Repeat("#", 100))
	fmt.Println("Password:", password)
	fmt.Println(strings.Repeat("#", 100))

	hashedPassword, err := cryptlib.GenerateHash(password, bcrypt.DefaultCost)
	if err != nil {
		return u, err
	}

	u = models.User{
		ID:         uuid.New(),
		ExternalID: session.Customer.ID,
		Email:      session.CustomerDetails.Email,
		Firstname:  session.CustomerDetails.Name,
		Password:   hashedPassword,
		Role:       coremodels.USER,
		CreatedAt:  time.Now(),
	}

	u, err = s.db.CreateUser(ctx, u)
	if err != nil {
		return u, err
	}

	go func() {
		err := s.mailcli.SendPasswordEmail(u.Email, hashedPassword)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("error sending auto generated password email")
		}
		log.Ctx(ctx).Debug().Any("email", u.Email).Msg("auto generated pasword email send")
	}()

	return u, nil
}
