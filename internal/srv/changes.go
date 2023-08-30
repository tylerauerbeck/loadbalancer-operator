package srv

import (
	"context"
	"errors"

	"helm.sh/helm/v3/pkg/storage/driver"
)

func (s *Server) processLoadBalancerChangeCreate(ctx context.Context, lb *loadBalancer) error {
	if err := s.createDeployment(ctx, lb); err != nil {
		return err
	}

	return nil
}

func (s *Server) processLoadBalancerChangeDelete(ctx context.Context, lb *loadBalancer) error {
	if err := s.removeDeployment(ctx, lb); err != nil {
		if errors.Is(err, driver.ErrReleaseNotFound) {
			// release does not exist, ack and move on
			return nil
		}

		return err
	}

	return nil
}

func (s *Server) processLoadBalancerChangeUpdate(ctx context.Context, lb *loadBalancer) error {
	if err := s.createDeployment(ctx, lb); err != nil {
		if errors.Is(err, driver.ErrReleaseNotFound) {
			// release does not exist, ack and move on
			return nil
		}

		return err
	}

	return nil
}
