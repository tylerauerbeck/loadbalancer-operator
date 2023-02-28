package srv

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"go.infratographer.com/loadbalanceroperator/internal/utils"
)

func TestSrvTestSuite(t *testing.T) {
	st := new(srvTestSuite)
	suite.Run(t, st)
}

type srvTestSuite struct {
	utils.OperatorTestSuite
}
