package service

import (
	"fmt"
	"time"

	"github.com/serverscom/serverscom-ingress-controller/internal/ingress"
	"github.com/serverscom/serverscom-ingress-controller/internal/ingress/controller/store"
	"github.com/serverscom/serverscom-ingress-controller/internal/service/loadbalancer"
	"github.com/serverscom/serverscom-ingress-controller/internal/service/sync"
	"github.com/serverscom/serverscom-ingress-controller/internal/service/tls"
	"golang.org/x/net/context"

	v1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
)

const (
	LBPollTimeout = 30 * time.Minute
)

// Service represents a struct that implements business logic
type Service struct {
	KubeClient        kubernetes.Interface
	tlsManager        tls.TLSManagerInterface
	lbManager         loadbalancer.LBManagerInterface
	store             store.Storer
	recorder          record.EventRecorder
	ingressClass      string
	certManagerPrefix string
	namespace         string
	syncManager       sync.Syncer
}

// New creates a new Service
func New(
	kubeClient kubernetes.Interface,
	tlsManager tls.TLSManagerInterface,
	lbManager loadbalancer.LBManagerInterface,
	store store.Storer,
	sync sync.Syncer,
	recorder record.EventRecorder,
	ingressClass string,
	certManagerPrefix string,
	namespace string) *Service {
	return &Service{
		KubeClient:        kubeClient,
		tlsManager:        tlsManager,
		lbManager:         lbManager,
		store:             store,
		recorder:          recorder,
		ingressClass:      ingressClass,
		certManagerPrefix: certManagerPrefix,
		syncManager:       sync,
		namespace:         namespace,
	}
}

// SyncToPortal syncs ingress configuration to portal by creating L7 load balancer
func (s *Service) SyncToPortal(key string) error {
	ing, err := s.store.GetIngress(key)
	if err != nil {
		if _, ok := err.(store.NotExistsError); ok {
			klog.V(2).Infof("ingress %q no longer exists", key)
			if err := s.syncManager.CleanupLBs(s.ingressClass); err != nil {
				s.recorder.Eventf(ing, v1.EventTypeWarning, "Sync", err.Error())
				return err
			}
			return nil
		}
		e := fmt.Errorf("fetching object with key %q from store failed: %v", key, err)
		s.recorder.Eventf(ing, v1.EventTypeWarning, "Sync", e.Error())
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
	klog.V(2).Infof("start syncing tls for ingress %q", key)
	sslCerts, err := s.syncManager.SyncTLS(ing, s.certManagerPrefix)
	if err != nil {
		e := fmt.Errorf("syncing tls for ingress %q failed: %v", key, err)
		s.recorder.Eventf(ing, v1.EventTypeWarning, "Sync", e.Error())
		return err
	}

	// generate lb input from ingress
	klog.V(2).Infof("start translating ingress %q to load balancer", key)
	lbInput, err := s.lbManager.TranslateIngressToLB(ing, sslCerts)
	if err != nil {
		e := fmt.Errorf("translate ingress %q to LB failed: %v", key, err)
		s.recorder.Eventf(ing, v1.EventTypeWarning, "Translate", e.Error())
		return err
	}

	klog.V(2).Infof("start syncing load balancer %q to portal", lbInput.Name)
	lb, err := s.syncManager.SyncL7LB(lbInput)
	if err != nil {
		e := fmt.Errorf("syncing LB for ingress %q failed: %v", key, err)
		s.recorder.Eventf(ing, v1.EventTypeWarning, "Sync", e.Error())
		return err
	}
	if lb == nil {
		e := fmt.Errorf("no LB returned after syncing for ingress %q", key)
		s.recorder.Eventf(ing, v1.EventTypeWarning, "Sync", e.Error())
		return e
	}

	// update ingress status
	klog.V(2).Infof("start updating ingress %q status with load balancer IPs", key)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), LBPollTimeout)
		defer cancel()
		activeLB, err := s.syncManager.SyncStatus(ctx, lb)
		if err != nil {
			s.recorder.Eventf(ing, v1.EventTypeWarning, "SyncStatus", err.Error())
			return
		}

		var ingress []networkv1.IngressLoadBalancerIngress
		for _, ip := range activeLB.ExternalAddresses {
			ingress = append(ingress, networkv1.IngressLoadBalancerIngress{IP: ip})
		}

		ing.Status = networkv1.IngressStatus{
			LoadBalancer: networkv1.IngressLoadBalancerStatus{
				Ingress: ingress,
			},
		}
		ingClient := s.KubeClient.NetworkingV1().Ingresses(ing.Namespace)
		_, err = ingClient.UpdateStatus(ctx, ing, metav1.UpdateOptions{})
		if err != nil {
			s.recorder.Eventf(ing, v1.EventTypeWarning, "UpdateStatus", err.Error())
			return
		}

		s.recorder.Eventf(ing, v1.EventTypeNormal, "Synced", "Successfully synced")
	}()

	s.recorder.Eventf(ing, v1.EventTypeNormal, "Created", "Successfully created")

	return nil
}
