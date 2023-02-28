package cmd

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"go.infratographer.com/loadbalanceroperator/internal/utils"
)

func TestCmdTestSuite(t *testing.T) {
	st := new(cmdTestSuite)
	suite.Run(t, st)
}

type cmdTestSuite struct {
	utils.OperatorTestSuite
}
