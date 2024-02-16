package controller

import (
	"time"

	"github.com/serverscom/serverscom-ingress-controller/internal/ingress/controller/store"
	"github.com/serverscom/serverscom-ingress-controller/internal/service"
	"github.com/serverscom/serverscom-ingress-controller/internal/service/loadbalancer"
	syncer "github.com/serverscom/serverscom-ingress-controller/internal/service/sync"
	"github.com/serverscom/serverscom-ingress-controller/internal/service/tls"

	"sync"

	serverscom "github.com/serverscom/serverscom-go-client/pkg"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/component-base/config"
	"k8s.io/klog/v2"
)

const (
	EventRecorderComponent = "sc-ingress-controller"
)

// IngressController represents an Ingress Controller
type IngressController struct {
	conf     *Configuration
	recorder record.EventRecorder
	queue    workqueue.RateLimitingInterface
	store    store.Storer
	service  *service.Service
	stopCh   chan struct{}
	stopLock sync.Mutex
	shutdown bool
}

// Configuration contains all the settings required by an Ingress controller
type Configuration struct {
	ShowVersion       bool
	Namespace         string
	LeaderElectionCfg *config.LeaderElectionConfiguration
	KubeClient        *kubernetes.Clientset
	ResyncPeriod      time.Duration
	IngressClass      string
}

// NewIngressController creates a new ingress controller
func NewIngressController(config *Configuration, client *serverscom.Client) *IngressController {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&corev1.EventSinkImpl{
		Interface: config.KubeClient.CoreV1().Events(config.Namespace),
	})

	ic := &IngressController{
		conf:  config,
		queue: workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		recorder: eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{
			Component: EventRecorderComponent,
		}),
	}
	ic.store = store.New(
		config.Namespace,
		config.ResyncPeriod,
		config.KubeClient,
		config.IngressClass,
		ic.recorder,
		ic.queue,
	)
	tlsManager := tls.NewManager(client.SSLCertificates, ic.store)
	lbManager := loadbalancer.NewManager(client.LoadBalancers, ic.store)
	ic.service = service.New(
		client,
		tlsManager,
		lbManager,
		ic.store,
		syncer.New(tlsManager, lbManager, ic.store),
		ic.recorder,
		config.IngressClass,
	)

	return ic
}

// Run runs informers and workers to process queue
func (ic *IngressController) Run(stopCh chan struct{}) {
	defer runtime.HandleCrash()
	defer ic.queue.ShutDown()
	ic.stopCh = stopCh

	go ic.store.Run(stopCh)

	go wait.Until(ic.runWorker, time.Second, stopCh)

	<-stopCh
}

// Stop gracefully stops controller
func (ic *IngressController) Stop() {
	ic.stopLock.Lock()
	defer ic.stopLock.Unlock()

	if !ic.shutdown {
		close(ic.stopCh)

		ic.queue.ShutDown()

		ic.shutdown = true
	}
}

// runWorker runs worker for processing queue
func (ic *IngressController) runWorker() {
	for ic.processNextItem() {
	}
}

// processNextItem process next item from queue
func (ic *IngressController) processNextItem() bool {
	key, quit := ic.queue.Get()
	if quit {
		return false
	}
	defer ic.queue.Done(key)

	err := ic.service.SyncToPortal(key.(string))

	ic.handleErr(err, key)
	return true
}

// handleErr checks if an error happened and makes sure we will retry later.
func (ic *IngressController) handleErr(err error, key interface{}) {
	if err == nil {
		ic.queue.Forget(key)
		return
	}

	// retries 2 times
	if ic.queue.NumRequeues(key) < 2 {
		klog.Infof("Error syncing ingress %v: %v", key, err)

		// re-enqueue the key rate limited
		ic.queue.AddRateLimited(key)
		return
	}

	ic.queue.Forget(key)

	runtime.HandleError(err)
	klog.Infof("Dropping ingress %q out of the queue: %v", key, err)
}
