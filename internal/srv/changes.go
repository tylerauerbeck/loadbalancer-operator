package srv

import (
	"context"
	"errors"

	"helm.sh/helm/v3/pkg/storage/driver"
)

func (s *Server) processLoadBalancerChangeCreate(lb *loadBalancer) error {
	if err := s.createDeployment(context.TODO(), lb); err != nil {
		return err
	}

	return nil
}

func (s *Server) processLoadBalancerChangeDelete(lb *loadBalancer) error {
	if err := s.removeDeployment(lb); err != nil {
		if errors.Is(err, driver.ErrReleaseNotFound) {
			// release does not exist, ack and move on
			return nil
		}

		return err
	}

	return nil
}

func (s *Server) processLoadBalancerChangeUpdate(lb *loadBalancer) error {
	if err := s.createDeployment(context.TODO(), lb); err != nil {
		if errors.Is(err, driver.ErrReleaseNotFound) {
			// release does not exist, ack and move on
			return nil
		}

		return err
	}

	return nil
}
