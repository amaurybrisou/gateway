package integration_test

import (
	"os"
	"testing"

	test "github.com/amaurybrisou/gateway/tests"
	"github.com/stretchr/testify/suite"
)

type gwTestSuite struct {
	test.DefaultTestSuite
}

func TestGWSuite(t *testing.T) {
	os.Setenv("DB_MIGRATIONS_PATH", "file://../../migrations")
	suite.Run(t, &gwTestSuite{})
}
