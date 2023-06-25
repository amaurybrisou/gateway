package coremiddleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/amaurybrisou/gateway/internal/db"
	coremodels "github.com/amaurybrisou/gateway/pkg/core/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type AuthMiddlewareService struct {
	db        *db.Database
	whitelist []string
}

func NewAuthMiddleware(db *db.Database, whitelist []string) AuthMiddlewareService {
	return AuthMiddlewareService{db: db, whitelist: whitelist}
}

func (s AuthMiddlewareService) IsAdmin(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		isAdmin := IsAdmin(r.Context())
		if !isAdmin {
			log.Ctx(r.Context()).Err(errors.New("not admin")).Send()
			http.Error(w, "Unauthorized", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func (s AuthMiddlewareService) BearerAuth(next http.Handler, getUser func(context.Context, string) (coremodels.UserInterface, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		isWhitelisted := false
		for _, path := range s.whitelist {
			if r.URL.Path == path {
				isWhitelisted = true
				break
			}
		}

		// Get the Authorization header value
		authHeader := r.Header.Get("Authorization")
		bearerFound := authHeader != "" || strings.HasPrefix(authHeader, "Bearer ")
		if !bearerFound && !isWhitelisted {
			log.Ctx(r.Context()).Error().Err(errors.New("no token found")).Send()
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !bearerFound && isWhitelisted {
			next.ServeHTTP(w, r)
			return
		}

		// Extract the token from the Authorization header
		token := strings.TrimPrefix(authHeader, "Bearer ")
		hasToken, err := s.db.HasToken(r.Context(), token)
		if (err != nil || !hasToken) && !isWhitelisted {
			log.Ctx(r.Context()).Error().Err(err).Msg("token not found")
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		user, err := getUser(r.Context(), token)
		if err != nil {
			log.Ctx(r.Context()).Error().Err(err).Msg("user not found")
			http.Error(w, "user not found", http.StatusUnauthorized)
			return
		}

		ctx := createUserContext(r.Context(), user)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	}
}

func createUserContext(ctx context.Context, user coremodels.UserInterface) context.Context {
	// Add user values to the new context
	ctx = context.WithValue(ctx, UserIDCtxKey, user.GetID())
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
	UserIDCtxKey  UserCtxKey = "user_id"
	UserEmail     UserCtxKey = "user_email"
	UserFirstname UserCtxKey = "user_firstname"
	UserLastname  UserCtxKey = "user_lastname"
	UserRole      UserCtxKey = "user_role"
	UserStripeKey UserCtxKey = "user_stripe_key"
	UserCreatedAt UserCtxKey = "user_created_at"
	UserUpdatedAt UserCtxKey = "user_updated_at"
	UserDeletedAt UserCtxKey = "user_deleted_at"
)

func User(ctx context.Context) coremodels.UserInterface {
	userID := ctx.Value(UserIDCtxKey)
	if userID == nil {
		return nil
	}

	return coremodels.User{
		ID:        ctx.Value(UserIDCtxKey).(uuid.UUID),
		Email:     ctx.Value(UserEmail).(string),
		Firstname: ctx.Value(UserFirstname).(string),
		Lastname:  ctx.Value(UserLastname).(string),
		Role:      ctx.Value(UserRole).(coremodels.GatewayRole),
		StripeKey: ctx.Value(UserStripeKey).(*string),
		CreatedAt: ctx.Value(UserCreatedAt).(time.Time),
		UpdatedAt: ctx.Value(UserUpdatedAt).(*time.Time),
		DeletedAt: ctx.Value(UserDeletedAt).(*time.Time),
	}
}

func IsAdmin(ctx context.Context) bool {
	user := User(ctx)
	if user == nil {
		return false
	}

	return user.GetRole() == coremodels.ADMIN
}
