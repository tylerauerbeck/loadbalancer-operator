package srv

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"go.infratographer.com/load-balancer-operator/internal/utils"
)

func TestSrvTestSuite(t *testing.T) {
	st := new(srvTestSuite)
	suite.Run(t, st)
	st.TearDownAllSuite()
}

type srvTestSuite struct {
	utils.OperatorTestSuite
}
