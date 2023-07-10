package integration_test

import (
	"context"
	"encoding/json"
	"io"

	"github.com/amaurybrisou/gateway/src/database/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func (s *gwTestSuite) TestStripeSubscriptionCreated() {
	t := s.T()

	service, err := s.DB.CreateService(context.Background(), models.Service{
		ID:            uuid.MustParse("2ea74ace-7f70-4997-9eab-2e5c094543bd"),
		Name:          "check-certs",
		Prefix:        "/check-certs",
		Host:          "http://localhost:127.0.0.1:50001",
		RequiredRoles: []models.Role{"check-certs"},
	})

	require.NoError(t, err)

	resp, err := s.PostWebhook("application/json", s.ReadFile("fixtures/checkout.session.completed.json"))
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	t.Log(string(body))

	var user models.UserRole
	err = json.Unmarshal(body, &user)
	require.NoError(t, err)

	require.Equal(t, user.Role, service.RequiredRoles[0])
	require.Nil(t, user.ExpiresAt)
	// require.Equal(t, map[string]string{"max_domains": "20"}, user.Metadata)

	resp, err = s.PostWebhook("application/json", s.ReadFile("fixtures/customer.subscription.updated.json"))
	require.NoError(t, err)
	resp.Body.Close()

	hasRole, err := s.DB.HasRole(context.Background(), user.UserID, service.RequiredRoles...)
	require.NoError(t, err)
	require.True(t, hasRole)
}
