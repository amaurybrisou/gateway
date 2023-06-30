package coremiddleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/amaurybrisou/gateway/pkg/core/jwtlib"
	coremodels "github.com/amaurybrisou/gateway/pkg/core/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type AuthMiddlewareService struct {
	jwt         *jwtlib.JWT
	getUserFunc func(context.Context, uuid.UUID) (coremodels.UserInterface, error)
}

func NewAuthMiddleware(jwt *jwtlib.JWT, getUserFunc func(context.Context, uuid.UUID) (coremodels.UserInterface, error)) AuthMiddlewareService {
	return AuthMiddlewareService{jwt: jwt, getUserFunc: getUserFunc}
}

func (s AuthMiddlewareService) IsAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isAdmin := IsAdmin(r.Context())
		if !isAdmin {
			log.Ctx(r.Context()).Error().Err(errors.New("not admin")).Msg("Unauthorized")
			http.Error(w, "Unauthorized", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s AuthMiddlewareService) JWTAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the Authorization header value
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.Ctx(r.Context()).Error().Err(errors.New("invalid header")).Msg("Unauthorized")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Extract the token from the Authorization header
		token := strings.TrimPrefix(authHeader, "Bearer ")

		// Verify the token
		claims, err := s.jwt.VerifyToken(token)
		if err != nil {
			log.Ctx(r.Context()).Error().Err(err).Msg("Unauthorized")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		userID, err := uuid.Parse(claims["sub"].(string))
		if err != nil {
			log.Ctx(r.Context()).Error().Err(err).Msg("Unauthorized")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		user, err := s.getUserFunc(r.Context(), userID)
		if err != nil {
			log.Ctx(r.Context()).Error().Err(err).Msg("Unauthorized")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := createUserContext(r.Context(), user)
		r = r.WithContext(ctx)

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

func createUserContext(ctx context.Context, user coremodels.UserInterface) context.Context {
	// Add user values to the new context
	ctx = context.WithValue(ctx, UserIDCtxKey, user.GetID())
	ctx = context.WithValue(ctx, ExternalIDCtxKey, user.GetExternalID())
	ctx = context.WithValue(ctx, UserEmail, user.GetEmail())
	ctx = context.WithValue(ctx, UserFirstname, user.GetFirstname())
	ctx = context.WithValue(ctx, UserLastname, user.GetLastname())
	ctx = context.WithValue(ctx, UserRole, user.GetRole())
	ctx = context.WithValue(ctx, UserStripeKey, user.GetStripeKey())
	ctx = context.WithValue(ctx, UserCreatedAt, user.GetCreatedAt())
	ctx = context.WithValue(ctx, UserUpdatedAt, user.GetUpdatedAt())
	ctx = context.WithValue(ctx, UserDeletedAt, user.GetDeletedAt())

	return ctx
}

type UserCtxKey string

const (
	UserIDCtxKey     UserCtxKey = "user_id"
	ExternalIDCtxKey UserCtxKey = "external_id"
	UserEmail        UserCtxKey = "user_email"
	UserFirstname    UserCtxKey = "user_firstname"
	UserLastname     UserCtxKey = "user_lastname"
	UserRole         UserCtxKey = "user_role"
	UserStripeKey    UserCtxKey = "user_stripe_key"
	UserCreatedAt    UserCtxKey = "user_created_at"
	UserUpdatedAt    UserCtxKey = "user_updated_at"
	UserDeletedAt    UserCtxKey = "user_deleted_at"
)

func User(ctx context.Context) coremodels.UserInterface {
	userID := ctx.Value(UserIDCtxKey)
	if userID == nil {
		return nil
	}

	u := coremodels.User{
		ID:         ctx.Value(UserIDCtxKey).(uuid.UUID),
		ExternalID: ctx.Value(ExternalIDCtxKey).(string),
		Email:      ctx.Value(UserEmail).(string),
		Firstname:  ctx.Value(UserFirstname).(string),
		Lastname:   ctx.Value(UserLastname).(string),
		Role:       ctx.Value(UserRole).(coremodels.GatewayRole),
		StripeKey:  ctx.Value(UserStripeKey).(*string),
		CreatedAt:  ctx.Value(UserCreatedAt).(time.Time),
		UpdatedAt:  ctx.Value(UserUpdatedAt).(*time.Time),
		DeletedAt:  ctx.Value(UserDeletedAt).(*time.Time),
	}

	return u
}

func IsAdmin(ctx context.Context) bool {
	user := User(ctx)
	if user == nil {
		return false
	}

	return user.GetRole() == coremodels.ADMIN
}
