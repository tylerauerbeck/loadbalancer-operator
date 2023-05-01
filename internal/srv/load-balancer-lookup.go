package srv

import (
	"github.com/google/uuid"
	"go.infratographer.com/x/urnx"
)

// getAssocLBUUID allows for filtering of the additional subject URNs to find the load balancer
func (s *Server) getAssocLBUUID(subjs []string) (uuid.UUID, error) {
	for _, subj := range subjs {
		urn, err := urnx.Parse(subj)
		if err != nil {
			s.Logger.Errorw("unable to parse load-balancer URN", "error", err)
			return uuid.UUID{}, err
		}

		if urn.ResourceType == loadbalancer {
			return urn.ResourceID, nil
		}
	}

	return uuid.UUID{}, errNoAssocLB
}
