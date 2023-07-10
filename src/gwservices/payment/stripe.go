package payment

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/amaurybrisou/ablib/jwtlib"
	"github.com/amaurybrisou/ablib/mailcli"
	"github.com/amaurybrisou/gateway/src/database"
	"github.com/amaurybrisou/gateway/src/database/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/client"
	"github.com/stripe/stripe-go/v72/webhook"
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
	ctx := r.Context()
	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to read request body")
		http.Error(w, "failed to read request body", http.StatusInternalServerError)
		return
	}

	// Verify and parse the webhook event
	event, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), s.webHookSecret)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to verify webhook event")
		http.Error(w, "failed to verify webhook event", http.StatusBadRequest)
		return
	}

	var sessionEvent stripe.Event
	err = json.Unmarshal(event.Data.Raw, &sessionEvent)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to unmarshal stripe event")
		http.Error(w, "failed to process webhook event", http.StatusInternalServerError)
		return
	}

	log.Ctx(ctx).Debug().Any("event_type", event.Type).Msg("stripe wehbook handler")

	// https://stripe.com/docs/api/events/types.
	switch event.Type {
	case "checkout.session.completed":
		// Payment is successful and the subscription is created.
		// You should provision the subscription and save the customer ID to your database.
		var session stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to unmarshal checkout session")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		user, err := s.RegisterUser(ctx, session.Customer.ID, session.CustomerDetails.Email, session.CustomerDetails.Name)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to unmarshal checkout session")
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

		service, err := s.db.GetServiceByID(ctx, serviceID)
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

		userRole, err := s.db.AddRole(ctx, user.ID, subID, role, nil)
		if err != nil {
			log.Error().Err(err).
				Any("service", service).
				Any("user_id", user.ID).
				Msg("failed to add user roles")
			http.Error(w, "failed to add user roles", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(userRole) //nolint
	case "customer.subscription.deleted":
		var sub stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &sub)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to unmarshal subscription deleted event")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = s.DeleteRole(ctx, &sub)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to unmarshal subscription deleted event")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

	case "customer.subscription.updated":
		var sub stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &sub)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to unmarshal subscription deleted event")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		items := sub.Items.Data
		if len(items) != 1 {
			return
		}

		metaData := items[0].Plan.Metadata

		currentPeriodendAt := time.Unix(sub.CurrentPeriodEnd, 0)
		_, err = s.db.UpdateRole(ctx, sub.ID, metaData, &currentPeriodendAt)
		if err != nil {
			log.Error().Err(err).
				Any("subscription_id", sub.ID).
				Any("ends_at", currentPeriodendAt).
				Msg("failed to update user role")
			http.Error(w, "failed to update user roles", http.StatusInternalServerError)
			return
		}

	case "subscription_schedule.canceled":
		var subSchedule stripe.SubscriptionSchedule
		err := json.Unmarshal(event.Data.Raw, &subSchedule)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to unmarshal subscription canceled event")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = s.DeleteRole(ctx, subSchedule.Subscription)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to unmarshal subscription canceled event")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	default:
	}

	w.WriteHeader(http.StatusOK)
}

func (s Service) DeleteRole(ctx context.Context, sub *stripe.Subscription) error {
	deleted, err := s.db.DelRoleBySubscriptionID(ctx, sub.ID)
	if err != nil {
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
