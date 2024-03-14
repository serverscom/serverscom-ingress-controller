package sync

import (
	"context"

	"github.com/jonboulle/clockwork"
	serverscom "github.com/serverscom/serverscom-go-client/pkg"
	"github.com/serverscom/serverscom-ingress-controller/internal/ingress/controller/store"
	"github.com/serverscom/serverscom-ingress-controller/internal/service/loadbalancer"
	"github.com/serverscom/serverscom-ingress-controller/internal/service/tls"
	networkv1 "k8s.io/api/networking/v1"
)

//go:generate mockgen --destination ../../mocks/sync_manager.go --package=mocks --source sync.go

// Syncer describes a sync interface
type Syncer interface {
	SyncTLS(ingress *networkv1.Ingress, certManagerPrefix string) (map[string]string, error)
	SyncL7LB(lb *serverscom.L7LoadBalancerCreateInput) (*serverscom.L7LoadBalancer, error)
	CleanupLBs(ingressClass string) error
	SyncStatus(ctx context.Context, lb *serverscom.L7LoadBalancer) (*serverscom.L7LoadBalancer, error)
}

// SyncManager represents a sync manager
type SyncManager struct {
	tlsMgr tls.TLSManagerInterface
	lbMgr  loadbalancer.LBManagerInterface
	store  store.Storer
	clock  clockwork.Clock
}

// New creates a new sync manager
func New(tlsManager tls.TLSManagerInterface,
	lbManager loadbalancer.LBManagerInterface,
	store store.Storer,
	clock clockwork.Clock) *SyncManager {
	return &SyncManager{
		tlsMgr: tlsManager,
		lbMgr:  lbManager,
		store:  store,
		clock:  clock,
	}
}
