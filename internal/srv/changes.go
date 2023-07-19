package srv

import "context"

func (s *Server) processLoadBalancerChangeCreate(lb *loadBalancer) error {
	// if err := s.newDeployment(lb); err != nil {
	// 	return err
	// }

	if err := s.createDeployment(context.TODO(), lb); err != nil {
		return err
	}

	return nil
}

func (s *Server) processLoadBalancerChangeDelete(lb *loadBalancer) error {
	if err := s.removeDeployment(lb); err != nil {
		return err
	}

	return nil
}

func (s *Server) processLoadBalancerChangeUpdate(lb *loadBalancer) error {
	// if err := s.updateDeployment(lb); err != nil {
	// 	return err
	// }

	if err := s.createDeployment(context.TODO(), lb); err != nil {
		return err
	}

	return nil
}
