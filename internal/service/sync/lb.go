package sync

import (
	serverscom "github.com/serverscom/serverscom-go-client/pkg"
	"github.com/serverscom/serverscom-ingress-controller/internal/ingress"
	"github.com/serverscom/serverscom-ingress-controller/internal/service/loadbalancer"
	"k8s.io/klog/v2"
)

// SyncL7LB add or update L7 Load Balancer in portal
func (s *SyncManager) SyncL7LB(lb *serverscom.L7LoadBalancerCreateInput) error {
	if s.lb.HasRegistration(lb.Name) {
		lbUpdateInput := &serverscom.L7LoadBalancerUpdateInput{
			Name:              lb.Name,
			StoreLogs:         lb.StoreLogs,
			StoreLogsRegionID: lb.StoreLogsRegionID,
			Geoip:             lb.Geoip,
			VHostZones:        lb.VHostZones,
			UpstreamZones:     lb.UpstreamZones,
		}
		_, err, _ := s.lb.UpdateLoadBalancer(lbUpdateInput)
		return err
	} else {
		_, err, _ := s.lb.NewLoadBalancer(lb)
		return err
	}
}

// CleanupLBs deletes Load Balancers that do not have corresponding SC Ingress from portal
func (s *SyncManager) CleanupLBs(ingressClass string) error {
	allIngresses := s.store.ListIngress()

	// LB is valid if it has corresponding SC Ingress
	validLBs := make(map[string]struct{})
	for _, ing := range allIngresses {
		if ingress.IsScIngress(ing, ingressClass) {
			lbName := loadbalancer.GetLoadBalancerName(ing)
			validLBs[lbName] = struct{}{}
		}
	}

	// delete LBs not associated with Ingress objects
	for _, lbID := range s.lb.GetIds() {
		if _, exists := validLBs[lbID]; !exists {
			err := s.lb.DeleteLoadBalancer(lbID)
			if err != nil {
				klog.Errorf("failed to delete Load Balancer %s: %v", lbID, err)
				return err
			} else {
				klog.V(2).Infof("successfully deleted Load Balancer %s", lbID)
			}
		}
	}
	return nil
}
