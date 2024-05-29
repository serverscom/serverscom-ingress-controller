package store

import (
	"fmt"
	"reflect"
	"time"

	"github.com/serverscom/serverscom-ingress-controller/internal/ingress"

	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

//go:generate mockgen --destination ../../../mocks/store.go --package=mocks --source store.go

// NotExistsError is returned when an object does not exist in a local store.
type NotExistsError string

// Error implements the error interface.
func (e NotExistsError) Error() string {
	return fmt.Sprintf("no object matching key %q in local store", string(e))
}

// Storer is the interface that wraps the required methods to gather information
// about ingresses, services, secrets and ingress annotations.
type Storer interface {
	Run(chan struct{})
	GetSecret(key string) (*corev1.Secret, error)
	GetIngress(key string) (*networkv1.Ingress, error)
	ListIngress() []*networkv1.Ingress
	GetService(key string) (*corev1.Service, error)
	GetNodesIpList() []string
	GetIngressServiceInfo(ingress *networkv1.Ingress) (map[string]ServiceInfo, error)
}

// Store represents cache store, implements Storer
type Store struct {
	// informer contains the cache Informers
	informers *Informer

	// listers contains the cache.Store interfaces used in the ingress controller
	listers *Lister
}

// GetSecret returns the Secret matching key.
func (s *Store) GetSecret(key string) (*corev1.Secret, error) {
	return s.listers.Secret.ByKey(key)
}

// GetIngress returns the Ingress matching key.
func (s *Store) GetIngress(key string) (*networkv1.Ingress, error) {
	return s.listers.Ingress.ByKey(key)
}

// ListIngress returns a list of ingresses.
func (s *Store) ListIngress() []*networkv1.Ingress {
	return s.listers.Ingress.ListIngress()
}

// GetService returns the Service matching key.
func (s *Store) GetService(key string) (*corev1.Service, error) {
	return s.listers.Service.ByKey(key)
}

// GetNodesIpList returns nodes ips
func (s *Store) GetNodesIpList() []string {
	return s.listers.Node.NodesIpList()
}

// GetIngressServiceInfo returns ingress services info.
func (s *Store) GetIngressServiceInfo(ingress *networkv1.Ingress) (map[string]ServiceInfo, error) {
	return getIngressServiceInfo(ingress, s)
}

type Informer struct {
	Ingress cache.SharedIndexInformer
	Service cache.SharedIndexInformer
	Secret  cache.SharedIndexInformer
	Node    cache.SharedIndexInformer
}

type Lister struct {
	Ingress IngressLister
	Service ServiceLister
	Secret  SecretLister
	Node    NodeLister
}

// Run initiates the synchronization of the informers against the API server.
func (i *Informer) Run(stopCh chan struct{}) {
	go i.Secret.Run(stopCh)
	go i.Service.Run(stopCh)
	go i.Node.Run(stopCh)

	// wait for all involved caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(stopCh,
		i.Service.HasSynced,
		i.Secret.HasSynced,
		i.Node.HasSynced,
	) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
	}

	go i.Ingress.Run(stopCh)
	if !cache.WaitForCacheSync(stopCh,
		i.Ingress.HasSynced,
	) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
	}
}

// Run initiates the synchronization of the informers
func (s *Store) Run(stopCh chan struct{}) {
	s.informers.Run(stopCh)
}

// New creates a new store.
// Add informers and it handlers.
func New(
	namespace string,
	resyncPeriod time.Duration,
	client *kubernetes.Clientset,
	ingressClass string,
	recorder record.EventRecorder,
	queue workqueue.RateLimitingInterface,
) *Store {
	store := &Store{
		informers: &Informer{},
		listers:   &Lister{},
	}

	factory := informers.NewSharedInformerFactoryWithOptions(client, resyncPeriod, informers.WithNamespace(namespace))

	store.informers.Ingress = factory.Networking().V1().Ingresses().Informer()
	store.listers.Ingress.Store = store.informers.Ingress.GetStore()

	store.informers.Secret = factory.Core().V1().Secrets().Informer()
	store.listers.Secret.Store = store.informers.Secret.GetStore()

	store.informers.Service = factory.Core().V1().Services().Informer()
	store.listers.Service.Store = store.informers.Service.GetStore()

	store.informers.Node = factory.Core().V1().Nodes().Informer()
	store.listers.Node.Store = store.informers.Node.GetStore()

	// add indexer 'byService' to find associated ingresses by service name
	store.informers.Ingress.AddIndexers(cache.Indexers{
		"byService": func(obj interface{}) ([]string, error) {
			ingress, ok := obj.(*networkv1.Ingress)
			if !ok {
				return nil, fmt.Errorf("unexpected type %T", obj)
			}
			var services []string
			for _, rule := range ingress.Spec.Rules {
				if rule.HTTP == nil {
					continue
				}
				for _, path := range rule.HTTP.Paths {
					services = append(services, path.Backend.Service.Name)
				}
			}
			return services, nil
		},
	})

	// add indexer 'bySecret' to find associated ingresses by secret name
	store.informers.Ingress.AddIndexers(cache.Indexers{
		"bySecret": func(obj interface{}) ([]string, error) {
			ingress, ok := obj.(*networkv1.Ingress)
			if !ok {
				return nil, fmt.Errorf("unexpected type %T", obj)
			}
			var secrets []string
			for _, tls := range ingress.Spec.TLS {
				secrets = append(secrets, tls.SecretName)
			}
			return secrets, nil
		},
	})

	// Ingress event handlers
	store.informers.Ingress.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			addIng := obj.(*networkv1.Ingress)
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				recorder.Eventf(addIng, corev1.EventTypeWarning, "CacheKey", err.Error())
				return
			}
			if !ingress.IsScIngress(addIng, ingressClass) {
				klog.V(4).Infof("Ignoring add for ingress %v based on annotation %v", key, ingress.IngressClassKey)
				return
			}
			klog.V(3).Infof("Ingress %v added, enqueuing", key)
			recorder.Eventf(addIng, corev1.EventTypeNormal, "CreateScheduled", key)
			queue.Add(key)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldIng := oldObj.(*networkv1.Ingress)
			newIng := newObj.(*networkv1.Ingress)
			if reflect.DeepEqual(oldIng, newIng) {
				return
			}
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			if err != nil {
				recorder.Eventf(newIng, corev1.EventTypeWarning, "CacheKey", err.Error())
				return
			}
			if ingress.IsScIngress(oldIng, ingressClass) || ingress.IsScIngress(newIng, ingressClass) {
				klog.V(3).Infof("Ingress %v updated, enqueuing", key)
				recorder.Eventf(newIng, corev1.EventTypeNormal, "UpdateScheduled", key)
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			delIng := obj.(*networkv1.Ingress)
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				recorder.Eventf(delIng, corev1.EventTypeWarning, "CacheKey", err.Error())
				return
			}
			if !ingress.IsScIngress(delIng, ingressClass) {
				klog.V(4).Infof("Ignoring delete for ingress %v based on annotation %v", key, ingress.IngressClassKey)
				return
			}
			klog.V(3).Infof("Ingress %v deleted, enqueueing", key)
			recorder.Eventf(delIng, corev1.EventTypeNormal, "DeleteScheduled", key)
			queue.Add(key)
		},
	})

	// Service event handlers
	store.informers.Service.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldSrv := oldObj.(*corev1.Service)
			newSrv := newObj.(*corev1.Service)
			if reflect.DeepEqual(oldSrv, newSrv) {
				return
			}
			ingresses, err := store.informers.Ingress.GetIndexer().ByIndex("byService", newSrv.Name)
			if err != nil {
				recorder.Eventf(newSrv, corev1.EventTypeWarning, "GetIndexerFailed", err.Error())
				return
			}
			sKey, err := cache.MetaNamespaceKeyFunc(newObj)
			if err != nil {
				recorder.Eventf(newSrv, corev1.EventTypeWarning, "CacheKey", err.Error())
				return
			}

			for _, ingressObj := range ingresses {
				ingress := ingressObj.(*networkv1.Ingress)
				iKey, err := cache.MetaNamespaceKeyFunc(ingressObj)
				if err != nil {
					recorder.Eventf(ingress, corev1.EventTypeWarning, "CacheKey", err.Error())
					return
				}
				klog.V(4).Infof("Service %v was changed, enqueuing associated ingress %v", sKey, iKey)
				recorder.Eventf(ingress, corev1.EventTypeNormal, "UpdateScheduled", iKey)
				queue.Add(iKey)
			}
		},
	})

	// Secret event handlers
	store.informers.Secret.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldSec := oldObj.(*corev1.Secret)
			newSec := newObj.(*corev1.Secret)
			if reflect.DeepEqual(oldSec, newSec) {
				return
			}
			ingresses, err := store.informers.Ingress.GetIndexer().ByIndex("bySecret", newSec.Name)
			if err != nil {
				recorder.Eventf(newSec, corev1.EventTypeWarning, "GetIndexerFailed", err.Error())
				return
			}
			sKey, err := cache.MetaNamespaceKeyFunc(newObj)
			if err != nil {
				recorder.Eventf(newSec, corev1.EventTypeWarning, "CacheKey", err.Error())
				return
			}
			for _, ingressObj := range ingresses {
				ingress := ingressObj.(*networkv1.Ingress)
				iKey, err := cache.MetaNamespaceKeyFunc(ingressObj)
				if err != nil {
					recorder.Eventf(ingress, corev1.EventTypeWarning, "CacheKey", err.Error())
					return
				}
				klog.V(4).Infof("Secret %v was changed, enqueuing associated ingress %v", sKey, iKey)
				recorder.Eventf(ingress, corev1.EventTypeNormal, "UpdateScheduled", iKey)
				queue.Add(iKey)
			}
		},
	})

	return store
}
