package service

import (
	"fmt"

	"github.com/serverscom/serverscom-ingress-controller/internal/ingress"
	"github.com/serverscom/serverscom-ingress-controller/internal/ingress/controller/store"
	"github.com/serverscom/serverscom-ingress-controller/internal/service/loadbalancer"
	"github.com/serverscom/serverscom-ingress-controller/internal/service/sync"
	"github.com/serverscom/serverscom-ingress-controller/internal/service/tls"

	serverscom "github.com/serverscom/serverscom-go-client/pkg"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
)

// Service represents a struct that implements business logic
type Service struct {
	ServerscomClient  *serverscom.Client
	tlsManager        tls.TLSManagerInterface
	lbManager         loadbalancer.LBManagerInterface
	store             store.Storer
	recorder          record.EventRecorder
	ingressClass      string
	certManagerPrefix string
	syncManager       sync.Syncer
}

// New creates a new Service
func New(client *serverscom.Client,
	tlsManager tls.TLSManagerInterface,
	lbManager loadbalancer.LBManagerInterface,
	store store.Storer,
	sync sync.Syncer,
	recorder record.EventRecorder,
	ingressClass string,
	certManagerPrefix string) *Service {
	return &Service{
		ServerscomClient:  client,
		tlsManager:        tlsManager,
		lbManager:         lbManager,
		store:             store,
		recorder:          recorder,
		ingressClass:      ingressClass,
		certManagerPrefix: certManagerPrefix,
		syncManager:       sync,
	}
}

// SyncToPortal syncs ingress configuration to portal by creating L7 load balancer
func (s *Service) SyncToPortal(key string) error {
	ing, err := s.store.GetIngress(key)
	if err != nil {
		if _, ok := err.(store.NotExistsError); ok {
			klog.V(2).Infof("ingress %q no longer exists", key)
			if err := s.syncManager.CleanupLBs(s.ingressClass); err != nil {
				s.recorder.Eventf(ing, v1.EventTypeWarning, "SyncFailed", err.Error())
				return err
			}
			return nil
		}
		e := fmt.Errorf("fetching object with key %s from store failed: %v", key, err)
		s.recorder.Eventf(ing, v1.EventTypeWarning, "SyncFailed", e.Error())
		return err
	}

	if !ingress.IsScIngress(ing, s.ingressClass) {
		klog.V(2).Infof("ingress %q class was changed, triggering remove", key)
		if err := s.syncManager.CleanupLBs(s.ingressClass); err != nil {
			return err
		}
		return nil
	}

	// get certs from ingress and sync it to portal
	sslCerts, err := s.syncManager.SyncTLS(ing, s.certManagerPrefix)
	if err != nil {
		e := fmt.Errorf("syncing tls for ingress '%s' failed: %v", key, err)
		s.recorder.Eventf(ing, v1.EventTypeWarning, "SyncFailed", e.Error())
		return err
	}

	// generate lb input from ingress
	lbInput, err := s.lbManager.TranslateIngressToLB(ing, sslCerts)
	if err != nil {
		e := fmt.Errorf("translate ingress '%s' to LB failed: %v", key, err)
		s.recorder.Eventf(ing, v1.EventTypeWarning, "SyncFailed", e.Error())
		return err
	}

	if err := s.syncManager.SyncL7LB(lbInput); err != nil {
		e := fmt.Errorf("syncing LB for ingress '%s' failed: %v", key, err)
		s.recorder.Eventf(ing, v1.EventTypeWarning, "SyncFailed", e.Error())
		return err
	}
	s.recorder.Eventf(ing, v1.EventTypeNormal, "Synced", "Successfully synced to portal")

	return nil
}
