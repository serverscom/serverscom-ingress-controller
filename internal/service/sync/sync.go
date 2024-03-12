package sync

import (
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
	SyncL7LB(lb *serverscom.L7LoadBalancerCreateInput) error
	CleanupLBs(ingressClass string) error
}

// SyncManager represents a sync manager
type SyncManager struct {
	tls   tls.TLSManagerInterface
	lb    loadbalancer.LBManagerInterface
	store store.Storer
}

// New creates a new sync manager
func New(tlsManager tls.TLSManagerInterface, lbManager loadbalancer.LBManagerInterface, store store.Storer) *SyncManager {
	return &SyncManager{
		tls:   tlsManager,
		lb:    lbManager,
		store: store,
	}
}
