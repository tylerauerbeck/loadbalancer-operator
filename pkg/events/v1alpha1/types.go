// Package events contains data structures that will be used to define
// the structure of a deployed load balancer
package events

import "github.com/google/uuid"

const (
	// EVENTCREATE is the event type to handle creation events
	EVENTCREATE = "create"
	// EVENTUPDATE is the event type to handle update events
	EVENTUPDATE = "update"
	// EVENTDELETE is the event type to handle delete events
	EVENTDELETE = "delete"
)

type LoadBalancerResources struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

type LoadBalancerData struct {
	LoadBalancerID uuid.UUID             `json:"load_balancer_id"`
	LocationID     uuid.UUID             `json:"location_id"`
	Resources      LoadBalancerResources `json:"resources"`
	QueryURL       string                `json:"query_url"`
}
