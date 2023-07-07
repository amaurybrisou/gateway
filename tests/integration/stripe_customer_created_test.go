package integration_test

import (
	"context"
	"encoding/json"
	"io"

	"github.com/amaurybrisou/gateway/src/database/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func (s *gwTestSuite) TestStripeCustomerCreated() {
	t := s.T()

	resp, err := s.PostWebhook("application/json", s.ReadFile("fixtures/customer.created.json"))
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	t.Log(string(body))

	var user models.User
	err = json.Unmarshal(body, &user)
	require.NoError(t, err)

	ctx := context.Background()
	u, err := s.DB.GetUserByID(ctx, user.ID)
	require.NoError(t, err)

	require.NotEqual(t, uuid.Nil, u.ID)
	require.Equal(t, user.Email, u.Email)
}
