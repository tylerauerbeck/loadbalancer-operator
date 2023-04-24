package srv

import (
	"context"
	"fmt"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"

	"go.uber.org/zap"

	"go.infratographer.com/x/echox"

	"go.infratographer.com/loadbalanceroperator/internal/utils"
)

func (suite srvTestSuite) TestConfigureSubscribers() { //nolint:govet
	type testCase struct {
		name        string
		subjects    []string
		expectError bool
	}

	js := utils.GetJetstreamConnection(suite.NATSServer)

	_, _ = js.AddStream(&nats.StreamConfig{
		Name:     "TestConfigureSubscribers",
		Subjects: []string{"thing.foo", "thing.bar"},
		MaxBytes: 1024,
	})

	testCases := []testCase{
		{
			name:        "single subject",
			subjects:    []string{"foo"},
			expectError: false,
		},
		{
			name:        "multiple subjects",
			subjects:    []string{"foo", "bar"},
			expectError: false,
		},
		{
			name:        "no subjects",
			subjects:    []string{},
			expectError: false,
		},
		{
			name:        "invalid subject",
			subjects:    []string{"boom", "bar", "baz"},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			s := &Server{
				Subjects:        tc.subjects,
				StreamName:      "TestConfigureSubscribers",
				Prefix:          "thing",
				JetstreamClient: js,
				Logger:          zap.NewNop().Sugar(),
			}

			err := s.configureSubscribers()

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func (suite srvTestSuite) TestRun() { //nolint:govet
	type testCase struct {
		name        string
		s           *Server
		expectError bool
	}

	js := utils.GetJetstreamConnection(suite.NATSServer)

	_, err := js.AddStream(&nats.StreamConfig{
		Name:     "TestRunner",
		Subjects: []string{"run.foo", "run.bar"},
		MaxBytes: 1024,
	})
	fmt.Println(err)

	testCases := []testCase{
		{
			name: "valid run",
			s: &Server{
				Echo:            echox.NewServer(zap.NewNop(), echox.Config{}, nil),
				Context:         context.TODO(),
				Subjects:        []string{"foo"},
				StreamName:      "TestRunner",
				Prefix:          "run",
				JetstreamClient: js,
				Logger:          zap.NewNop().Sugar(),
			},
			// hcport:      ":8900",
			expectError: false,
		},
		{
			name: "bad subject",
			s: &Server{
				Echo:            echox.NewServer(zap.NewNop(), echox.Config{}, nil),
				Context:         context.TODO(),
				Subjects:        []string{"foo", "bar", "baz"},
				StreamName:      "TestRunner",
				Prefix:          "run",
				JetstreamClient: js,
				Logger:          zap.NewNop().Sugar(),
			},
			// hcport:      ":8901",
			expectError: true,
		},
		// {
		// 	name:   "bad healthcheck port",
		// 	hcport: "8675309",
		// 	s: &Server{
		// 		Echo:            echox.NewServer(zap.NewNop(), echox.Config{}, nil),
		// 		Context:         context.TODO(),
		// 		Subjects:        []string{"foo"},
		// 		StreamName:      "TestRunner",
		// 		Prefix:          "run",
		// 		JetstreamClient: js,
		// 		Logger:          zap.NewNop().Sugar(),
		// 	},
		// },
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			err := tc.s.Run(tc.s.Context)

			// // if tc.hcport != "" {
			// viper.Set("healthcheck.port", tc.hcport)
			// // }

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
