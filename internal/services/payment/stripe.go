package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/amaurybrisou/gateway/internal/db"
	"github.com/amaurybrisou/gateway/internal/db/models"
	"github.com/amaurybrisou/gateway/pkg/core/jwtlib"
	coremodels "github.com/amaurybrisou/gateway/pkg/core/models"
	coremiddleware "github.com/amaurybrisou/gateway/pkg/http/middleware"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/checkout/session"
	"github.com/stripe/stripe-go/v72/webhook"
)

type Service struct {
	db            *db.Database
	stripeKey     string
	successURL    string
	webHookSecret string
	cancelURL     string
}

type Config struct {
	StripeKey, StripeSuccessURL, StripeCancelURL, StripeWebHookSecret string
}

func NewService(db *db.Database, cfg Config) *Service {
	stripe.Key = cfg.StripeKey

	return &Service{
		db:            db,
		stripeKey:     cfg.StripeKey,
		successURL:    cfg.StripeSuccessURL,
		cancelURL:     cfg.StripeCancelURL,
		webHookSecret: cfg.StripeWebHookSecret,
	}
}

func (s Service) BuyServiceHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the required parameters from the request body
	var request struct {
		PlanID         uuid.UUID `json:"plan_id"`
		IdempotencyKey uuid.UUID `json:"idempotency_key"`
	}

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		log.Error().Err(err).Msg("failed to parse request")
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	user := coremiddleware.User(r.Context())

	payment, err := s.db.GetFullPaymentByID(r.Context(), request.IdempotencyKey)
	if err != nil {
		log.Error().Err(err).Msg("get payment")
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}

	if payment.ID != request.IdempotencyKey {
		payment, err = s.db.CreatePayment(r.Context(), request.IdempotencyKey, user.GetID(), request.PlanID)
		if err != nil {
			log.Error().Err(err).Msg("failed to fetch payment")
			http.Error(w, "failed to create payment intent", http.StatusInternalServerError)
			return
		}

		plan, err := s.db.GetPlanByID(r.Context(), payment.PlanID)
		if err != nil {
			log.Error().Err(err).Msg("failed to fetch payment")
			http.Error(w, "failed to create payment intent", http.StatusInternalServerError)
			return
		}

		payment.Plan = plan
	}

	sessionParams := &stripe.CheckoutSessionParams{
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Quantity: stripe.Int64(1),
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					UnitAmount: stripe.Int64(int64(payment.Plan.Price)),
					Currency:   stripe.String(payment.Plan.Currency),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String(payment.Plan.Name),
						Description: stripe.String(payment.Plan.Description),
					},
				},
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(s.successURL), // service domain
		CancelURL:  stripe.String(s.cancelURL),  // home
		// PaymentIntentData: &stripe.CheckoutSessionPaymentIntentDataParams{
		// 	Metadata: map[string]string{
		// 		"payment_id": payment.ID.String(),
		// 	},
		// 	Params: stripe.Params{
		// 		IdempotencyKey: stripe.String(payment.ID.String()),
		// 	},
		// },
	}

	checkoutSession, err := session.New(sessionParams)
	if err != nil {
		log.Error().Err(err).Msg("failed to create checkout session")
		http.Error(w, "failed to create checkout session", http.StatusInternalServerError)
		return
	}

	// Return the checkout session ID to the client
	response := struct {
		SessionID   string `json:"session_id"`
		RedirectURL string `json:"redirect_url"`
	}{
		SessionID:   checkoutSession.ID,
		RedirectURL: checkoutSession.URL,
	}

	json.NewEncoder(w).Encode(response) //nolint
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

	switch event.Type {
	case "checkout.session.completed":
		fmt.Fprintf(os.Stderr, "Unhandled event type: %s\n", event.Type)
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

		_, err = s.db.AddRole(r.Context(), user.ID, service.RequiredRoles[0], time.Now().Add(time.Hour))
		if err != nil {
			log.Error().Err(err).Msg("failed to add user roles")
			http.Error(w, "failed to add user roles", http.StatusInternalServerError)
			return
		}

		// Payment is successful and the subscription is created.
		// You should provision the subscription and save the customer ID to your database.
	case "invoice.paid":
		fmt.Fprintf(os.Stderr, "Unhandled event type: %s\n", event.Type)

		// Continue to provision the subscription as payments continue to be made.
		// Store the status in your database and check when a user accesses your service.
		// This approach helps you avoid hitting rate limits.
	case "invoice.payment_failed":
		fmt.Fprintf(os.Stderr, "Unhandled event type: %s\n", event.Type)

		// The payment failed or the customer does not have a valid payment method.
		// The subscription becomes past_due. Notify your customer and send them to the
		// customer portal to update their payment information.

	case "payment_intent.succeeded":
		var paymentIntent stripe.PaymentIntent
		err := json.Unmarshal(event.Data.Raw, &paymentIntent)
		if err != nil {
			log.Ctx(r.Context()).Error().Err(err).Msg("failed to unmarshal webhook event")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Get the user and service IDs from the metadata
		paymentID, err := uuid.Parse(paymentIntent.Metadata["payment_id"])
		if err != nil {
			log.Error().Err(err).Msg("failed to parse user ID from webhook event")
			http.Error(w, "failed to process webhook event", http.StatusInternalServerError)
			return
		}

		payment, err := s.db.GetFullPaymentByID(r.Context(), paymentID)
		if err != nil {
			log.Error().Err(err).Msg("failed to fetch payment")
			http.Error(w, "failed to fetch payment", http.StatusInternalServerError)
			return
		}

		service, err := s.db.GetServiceByID(r.Context(), payment.Plan.ServiceID)
		if err != nil {
			log.Error().Err(err).Msg("failed to fetch service")
			http.Error(w, "failed to fetch service", http.StatusInternalServerError)
			return
		}

		// Update the payment status in the database to "PAID" // receipt_url
		err = s.db.UpdatePayment(r.Context(), paymentID, models.PaymentPaid)
		if err != nil {
			log.Error().Err(err).Msg("failed to update payment status")
			http.Error(w, "failed to update payment status", http.StatusInternalServerError)
			return
		}

		// Assign the required roles to the user for the purchased service
		_, err = s.db.AddRole(r.Context(), payment.UserID, service.RequiredRoles[0], time.Now().Add(time.Duration(payment.Plan.Duration)))
		if err != nil {
			log.Error().Err(err).Msg("failed to add user roles")
			http.Error(w, "failed to add user roles", http.StatusInternalServerError)
			return
		}
	case "payment_intent.created":
		// set payment to CREATED
	default:
		fmt.Fprintf(os.Stderr, "Unhandled event type: %s\n", event.Type)
	}

	// Return a success response to Stripe
	w.WriteHeader(http.StatusOK)
}

func (s Service) RegisterUser(ctx context.Context, session *stripe.CheckoutSession) (models.User, error) {
	l := jwtlib.New("secret")

	u := models.User{
		ID:        uuid.New(),
		Email:     session.Customer.Email,
		Firstname: session.Customer.Name,
		Role:      coremodels.USER,
		CreatedAt: time.Now(),
	}

	t, err := l.GenerateToken(u.ID.String(), time.Now().Add(time.Hour))
	if err != nil {
		return u, err
	}

	token := models.AccessToken{
		UserID:    u.ID,
		Token:     t,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	err = s.db.CreateUserAndToken(ctx, u, token)
	if err != nil {
		return u, err
	}

	return u, nil
}
