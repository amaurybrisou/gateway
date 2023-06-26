package payment

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/amaurybrisou/gateway/internal/db"
	"github.com/amaurybrisou/gateway/internal/db/models"
	coremiddleware "github.com/amaurybrisou/gateway/pkg/http/middleware"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/checkout/session"
	"github.com/stripe/stripe-go/v72/webhook"
)

type Service struct {
	db         *db.Database
	stripeKey  string
	successURL string
	cancelURL  string
}

func NewService(db *db.Database, stripeKey, successURL, cancelURL string) *Service {
	return &Service{
		db:         db,
		stripeKey:  stripeKey,
		successURL: successURL,
		cancelURL:  cancelURL,
	}
}

func (s Service) BuyServiceHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the required parameters from the request body
	var request struct {
		ServiceID uuid.UUID     `json:"service_id"`
		Duration  time.Duration `json:"duration"`
	}

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch payment")
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	user := coremiddleware.User(r.Context())

	service, err := s.db.GetServiceByID(r.Context(), request.ServiceID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch payment")
		http.Error(w, "Service not found", http.StatusNotFound)
		return
	}

	payment, err := s.db.CreatePayment(r.Context(), user.GetID(), service.ID, request.Duration, service.Costs[models.SubDuration(request.Duration)])
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch payment")
		http.Error(w, "Failed to create payment intent", http.StatusInternalServerError)
		return
	}

	sessionParams := &stripe.CheckoutSessionParams{
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				// Provide the exact Price ID (for example, pr_1234) of the product you want to sell
				Price:    stripe.String(fmt.Sprintf("%f", payment.Amount*100)),
				Quantity: stripe.Int64(1),
				Currency: stripe.String("EUR"),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(service.Host),
		CancelURL:  stripe.String("http://localhost:8089/cancel.html"),
	}

	sessionParams.AddMetadata("user_id", string(user.GetID().String()))
	sessionParams.AddMetadata("service_id", request.ServiceID.String())
	sessionParams.AddMetadata("callback_url", s.successURL)
	sessionParams.AddMetadata("payment_id", payment.ID.String())

	sessionParams.PaymentIntentData.SetIdempotencyKey(payment.ID.String())

	checkoutSession, err := session.New(sessionParams)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch payment")
		http.Error(w, "Failed to create checkout session", http.StatusInternalServerError)
		return
	}

	// Return the checkout session ID to the client
	response := struct {
		SessionID string `json:"session_id"`
	}{
		SessionID: checkoutSession.ID,
	}

	json.NewEncoder(w).Encode(response)
}

func (s Service) StripeSuccess(w http.ResponseWriter, r *http.Request) {
	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read request body")
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	// Verify and parse the webhook event
	event, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), s.stripeKey)
	if err != nil {
		log.Error().Err(err).Msg("Failed to verify webhook event")
		http.Error(w, "Failed to verify webhook event", http.StatusBadRequest)
		return
	}

	var sessionEvent stripe.CheckoutSession
	err = json.Unmarshal(event.Data.Raw, &sessionEvent)
	if err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal checkout session event")
		http.Error(w, "Failed to process webhook event", http.StatusInternalServerError)
		return
	}

	// Get the user and service IDs from the metadata
	paymentID, err := uuid.Parse(sessionEvent.Metadata["payment_id"])
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse user ID from webhook event")
		http.Error(w, "Failed to process webhook event", http.StatusInternalServerError)
		return
	}

	payment, err := s.db.GetPaymentByID(r.Context(), paymentID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch payment")
		http.Error(w, "Failed to fetch payment", http.StatusInternalServerError)
		return
	}

	userID, err := uuid.Parse(sessionEvent.Metadata["user_id"])
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse service ID from webhook event")
		http.Error(w, "Failed to process webhook event", http.StatusInternalServerError)
		return
	}

	serviceID, err := uuid.Parse(sessionEvent.Metadata["service_id"])
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse service ID from webhook event")
		http.Error(w, "Failed to process webhook event", http.StatusInternalServerError)
		return
	}

	service, err := s.db.GetServiceByID(r.Context(), serviceID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch service")
		http.Error(w, "Failed to fetch service", http.StatusInternalServerError)
		return
	}

	// Update the payment status in the database to "PAID"
	err = s.db.UpdatePayment(r.Context(), paymentID, models.PaymentPaid)
	if err != nil {
		log.Error().Err(err).Msg("Failed to update payment status")
		http.Error(w, "Failed to update payment status", http.StatusInternalServerError)
		return
	}

	// Assign the required roles to the user for the purchased service
	_, err = s.db.AddRole(r.Context(), userID, service.RequiredRoles[0], payment.Duration)
	if err != nil {
		log.Error().Err(err).Msg("Failed to add user roles")
		http.Error(w, "Failed to add user roles", http.StatusInternalServerError)
		return
	}

	// Return a success response to Stripe
	w.WriteHeader(http.StatusOK)
}
