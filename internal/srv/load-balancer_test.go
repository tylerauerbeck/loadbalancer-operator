package srv

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.infratographer.com/loadbalancer-manager-haproxy/pkg/lbapi"

	"go.infratographer.com/loadbalanceroperator/internal/utils/mock"

	"go.infratographer.com/x/gidx"
	"go.uber.org/zap"
)

var (
	dummyLB = gidx.MustNewID("loadbal")
)

func (suite *srvTestSuite) TestGetLBFromAddSubjs() { //nolint:govet
	type testCase struct {
		name   string
		adds   []gidx.PrefixedID
		expect bool
	}

	testCases := []testCase{
		{
			name:   "no additional subjects",
			adds:   []gidx.PrefixedID{},
			expect: false,
		},
		{
			name: "additional subjects, no loadbalancer",
			adds: []gidx.PrefixedID{
				gidx.MustNewID("testsub"),
			},
			expect: false,
		},
		{
			name: "additional subjects, loadbalancer",
			adds: []gidx.PrefixedID{
				dummyLB,
			},
			expect: true,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			check, subs := getLBFromAddSubjs(tc.adds)

			assert.Equal(t, tc.expect, check)

			if tc.expect {
				assert.Equal(t, dummyLB, subs)
			}
		})
	}
}

func (suite *srvTestSuite) TestIsLoadBalancer() { //nolint:govet
	type testCase struct {
		name   string
		subj   gidx.PrefixedID
		adds   []gidx.PrefixedID
		lbType int
	}

	testCases := []testCase{
		{
			name: "lb subject",
			subj: dummyLB,
			adds: []gidx.PrefixedID{
				gidx.MustNewID("testsub"),
			},
			lbType: typeLB,
		},
		{
			name: "lb additional subject",
			subj: gidx.MustNewID("testsub"),
			adds: []gidx.PrefixedID{
				dummyLB,
			},
			lbType: typeAssocLB,
		},
		{
			name: "no lb",
			subj: gidx.MustNewID("testsub"),
			adds: []gidx.PrefixedID{
				gidx.MustNewID("dummysb"),
			},
			lbType: typeNoLB,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			l := new(loadBalancer)

			assert.Equal(t, l.loadBalancerID.String(), "")

			l.isLoadBalancer(tc.subj, tc.adds)

			assert.Equal(t, tc.lbType, l.lbType)

			if tc.lbType != typeNoLB {
				assert.Equal(t, dummyLB, l.loadBalancerID)
			} else {
				assert.Equal(t, l.loadBalancerID.String(), "")
			}
		})
	}
}

func (suite *srvTestSuite) TestNewLoadBalancer() { //nolint:govet
	type testCase struct {
		name   string
		subj   gidx.PrefixedID
		adds   []gidx.PrefixedID
		lbType int
	}

	testCases := []testCase{
		{
			name:   "lb subject",
			subj:   dummyLB,
			adds:   []gidx.PrefixedID{},
			lbType: typeLB,
		},
		{
			name:   "lb additional subject",
			subj:   gidx.MustNewID("testsub"),
			adds:   []gidx.PrefixedID{dummyLB},
			lbType: typeAssocLB,
		},
		{
			name:   "no lb",
			subj:   gidx.MustNewID("testsub"),
			adds:   []gidx.PrefixedID{gidx.MustNewID("dummysb")},
			lbType: typeNoLB,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			server := mock.DummyAPI(tc.subj.String())
			server.Start()

			defer server.Close()

			srv := &Server{
				APIClient: lbapi.NewClient(server.URL),
				Logger:    zap.NewNop().Sugar(),
				Context:   context.TODO(),
			}

			lb, err := srv.newLoadBalancer(context.TODO(), tc.subj, tc.adds)

			assert.Equal(t, lb.lbType, tc.lbType)
			assert.Nil(t, err)

			if tc.lbType != typeNoLB {
				assert.NotNil(t, lb.lbData)
				assert.Equal(t, lb.loadBalancerID, dummyLB)
			} else {
				assert.Nil(t, lb.lbData)
				assert.Equal(t, lb.loadBalancerID.String(), "")
			}
		})
	}
}

func (suite *srvTestSuite) TestNewLoadBalancer_InvalidAPI() { //nolint:govet
	srv := &Server{
		APIClient: lbapi.NewClient("http://localhost:9999"),
		Logger:    zap.NewNop().Sugar(),
		Context:   context.TODO(),
	}

	lb, err := srv.newLoadBalancer(context.TODO(), dummyLB, []gidx.PrefixedID{})
	assert.Nil(suite.T(), lb)
	assert.NotNil(suite.T(), err)
}
