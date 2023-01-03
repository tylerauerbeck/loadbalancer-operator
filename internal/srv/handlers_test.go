package srv

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"

	"go.infratographer.com/x/pubsubx"

	events "go.infratographer.com/loadbalanceroperator/pkg/events/v1alpha1"
)

var (
	msg = pubsubx.Message{
		SubjectURN: uuid.NewString(),
		EventType:  "create",
		Source:     "loadbalancerapi",
		Timestamp:  time.Now(),
		ActorURN:   uuid.NewString(),
	}
)

func TestParseLBData(t *testing.T) {
	type testCase struct {
		name        string
		data        map[string]interface{}
		expectError bool
	}

	testCases := []testCase{
		{
			name:        "valid data",
			expectError: false,
			data: map[string]interface{}{
				"load_balancer_id": uuid.New(),
				"location_id":      uuid.New(),
			},
		},
		{
			name:        "unable to parse event data",
			expectError: true,
			data: map[string]interface{}{
				"load_balancer_id": 1,
				"location_id":      2,
			},
		},
		{
			name:        "unable to load event data",
			expectError: true,
			data: map[string]interface{}{
				"other field": make(chan int),
			},
		},
	}

	for _, tcase := range testCases {
		t.Run(tcase.name, func(t *testing.T) {
			lbData := events.LoadBalancerData{}
			srv := &Server{
				Logger: zap.NewNop().Sugar(),
			}
			msg.AdditionalData = tcase.data
			err := srv.parseLBData(&tcase.data, &lbData)

			if tcase.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, lbData)
			}
		})
	}
}
