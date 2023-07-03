package integration_test

import (
	"net/http"

	"github.com/stretchr/testify/require"
)

func (s *gwTestSuite) TestLogin() {
	t := s.T()

	resp, err := s.Post("/login", "application/json", `{
		"email": "gateway@gateway.com",
		"password":  "w9oHDCAlPxT12WbH"
	}`)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	resp, err = s.Post("/login", "application/json", `{
		"email": "gateway@gateway.com",
		"password":  "wrongpassword"
	}`)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
