package srv

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/stretchr/testify/assert"

	events "go.infratographer.sh/loadbalanceroperator/pkg/events/v1alpha1"
	"go.infratographer.sh/loadbalanceroperator/pkg/pubsubx"
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
	t.Parallel()

	data := make(map[string]interface{})
	data["load_balancer_id"] = uuid.New()
	data["location_id"] = uuid.New()
	msg.AdditionalData = data

	lbData := events.LoadBalancerData{}
	err := parseLBData(&data, &lbData)

	assert.Nil(t, err)
	assert.Equal(t, lbData.LoadBalancerID, data["load_balancer_id"])
	assert.Equal(t, lbData.LocationID, data["location_id"])

	data["load_balancer_id"] = 1
	data["location_id"] = 2
	msg.AdditionalData = data

	err = parseLBData(&data, &lbData)

	assert.NotNil(t, err)

	data["thing"] = make(chan int)
	err = parseLBData(&data, &lbData)

	assert.NotNil(t, err)
}
