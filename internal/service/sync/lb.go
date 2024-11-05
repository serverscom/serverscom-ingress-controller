package sync

import (
	"context"
	"fmt"
	"time"

	serverscom "github.com/serverscom/serverscom-go-client/pkg"
	"github.com/serverscom/serverscom-ingress-controller/internal/ingress"
	"github.com/serverscom/serverscom-ingress-controller/internal/service/loadbalancer"
	"k8s.io/klog/v2"
)

const (
	LBPollInterval = 5 * time.Second
)

// SyncL7LB add or update L7 Load Balancer in portal
func (s *SyncManager) SyncL7LB(lb *serverscom.L7LoadBalancerCreateInput) (*serverscom.L7LoadBalancer, error) {
	if s.lbMgr.HasRegistration(lb.Name) {
		lbUpdateInput := &serverscom.L7LoadBalancerUpdateInput{
			Name:              lb.Name,
			StoreLogs:         lb.StoreLogs,
			StoreLogsRegionID: lb.StoreLogsRegionID,
			Geoip:             lb.Geoip,
			VHostZones:        lb.VHostZones,
			UpstreamZones:     lb.UpstreamZones,
			ClusterID:         lb.ClusterID,
		}
		if lbUpdateInput.ClusterID == nil {
			lbUpdateInput.SharedCluster = new(bool)
			*lbUpdateInput.SharedCluster = true
		}
		result, err, _ := s.lbMgr.UpdateLoadBalancer(lbUpdateInput)
		return result, err
	} else {
		result, err, _ := s.lbMgr.NewLoadBalancer(lb)
		return result, err
	}
}

// CleanupLBs deletes Load Balancers that do not have corresponding SC Ingress in portal
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
	for _, lbID := range s.lbMgr.GetIds() {
		if _, exists := validLBs[lbID]; !exists {
			err := s.lbMgr.DeleteLoadBalancer(lbID)
			if err != nil {
				return fmt.Errorf("failed to delete Load Balancer %s: %v", lbID, err)
			}
			klog.V(2).Infof("successfully deleted Load Balancer %s", lbID)
		}
	}
	return nil
}

func (s *SyncManager) SyncStatus(ctx context.Context, lb *serverscom.L7LoadBalancer) (*serverscom.L7LoadBalancer, error) {
	if loadbalancer.IsActiveStatus(lb.Status) {
		return lb, nil
	}

	for {
		select {
		case <-s.clock.After(LBPollInterval):
			tmpLB, err := s.lbMgr.GetLoadBalancer(lb.Name)
			if err != nil {
				continue
			}
			if loadbalancer.IsActiveStatus(tmpLB.Status) {
				return tmpLB, nil
			}

		case <-ctx.Done():
			return nil, fmt.Errorf("poll LB timeout reached")
		}
	}
}
