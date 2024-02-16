package loadbalancer

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/serverscom/serverscom-ingress-controller/internal/config"
	"github.com/serverscom/serverscom-ingress-controller/internal/ingress/controller/store"
	"github.com/serverscom/serverscom-ingress-controller/internal/service/annotations"

	client "github.com/serverscom/serverscom-go-client/pkg"
	networkv1 "k8s.io/api/networking/v1"
	"k8s.io/klog/v2"
)

//go:generate mockgen --destination ../../mocks/lb_manager.go --package=mocks --source manager.go

// ManagerInterface describes an interface to manage load balancers
type LBManagerInterface interface {
	HasRegistration(name string) bool
	NewLoadBalancer(input *client.L7LoadBalancerCreateInput) (*client.L7LoadBalancer, error, bool)
	DeleteLoadBalancer(name string) error
	UpdateLoadBalancer(input *client.L7LoadBalancerUpdateInput) (*client.L7LoadBalancer, error, bool)
	GetIds() []string
	TranslateIngressToLB(ingress *networkv1.Ingress, sslCerts map[string]string) (*client.L7LoadBalancerCreateInput, error)
}

// Manager represents a load balancer manager
type Manager struct {
	resources map[string]*LoadBalancer

	lock      sync.Mutex
	lBService client.LoadBalancersService
	store     store.Storer
}

// NewManager creates a load balancer manager
func NewManager(lBService client.LoadBalancersService, store store.Storer) *Manager {
	return &Manager{
		resources: make(map[string]*LoadBalancer),
		lBService: lBService,
		store:     store,
	}
}

// HasRegistration checks if lb manager has load balancer with specified name
func (manager *Manager) HasRegistration(name string) bool {
	manager.lock.Lock()
	defer manager.lock.Unlock()

	_, ok := manager.resources[name]
	return ok
}

// NewLoadBalancer creates a new load balancer in portal from input if it doesn't exists in portal, otherwise update it
// Updates load balancer state in manager
func (manager *Manager) NewLoadBalancer(input *client.L7LoadBalancerCreateInput) (*client.L7LoadBalancer, error, bool) {
	manager.lock.Lock()
	defer manager.lock.Unlock()

	lb := NewLoadBalancer(manager.lBService, input)
	lb.Find(input.Name)
	l7, err := lb.Sync()

	if err != nil {
		return nil, err, false
	}

	manager.resources[input.Name] = lb
	return l7, nil, true
}

// DeleteLoadBalancer deletes load balancer from portal and manager
func (manager *Manager) DeleteLoadBalancer(name string) error {
	manager.lock.Lock()
	defer manager.lock.Unlock()

	lb, ok := manager.resources[name]

	if !ok {
		return fmt.Errorf("can't find resource: %s", name)
	}

	lb.MarkAsDeleted()

	_, err := lb.Sync()

	if err != nil {
		return err
	}

	delete(manager.resources, name)

	return nil
}

// UpdateLoadBalancer updates load balancer in portal and manager.
func (manager *Manager) UpdateLoadBalancer(input *client.L7LoadBalancerUpdateInput) (*client.L7LoadBalancer, error, bool) {
	manager.lock.Lock()
	defer manager.lock.Unlock()

	lb, ok := manager.resources[input.Name]

	if !ok {
		return nil, fmt.Errorf("can't find resource: %s", input.Name), false
	}

	if lb.deleted {
		return lb.state, nil, false
	}

	if !lb.IsChanged(input) {
		return lb.state, nil, false
	}

	lbCopy := lb.Copy()

	lb.UpdateInput(input)

	l7, err := lb.Sync()

	if err != nil {
		manager.resources[input.Name] = lbCopy

		return nil, err, false
	}

	return l7, nil, true
}

func (manager *Manager) GetIds() []string {
	var ids []string

	for k := range manager.resources {
		ids = append(ids, k)
	}

	return ids
}

// TranslateIngressToLB maps an Ingress to L7 LB object and fills annotations
func (manager *Manager) TranslateIngressToLB(ingress *networkv1.Ingress, sslCerts map[string]string) (*client.L7LoadBalancerCreateInput, error) {
	sInfo, err := manager.store.GetIngressServiceInfo(ingress)
	if err != nil {
		return nil, err
	}

	var upstreamZones []client.L7UpstreamZoneInput
	var upstreams []client.L7UpstreamInput
	var vhostZones []client.L7VHostZoneInput
	var locationZones []client.L7LocationZoneInput

	for sKey, service := range sInfo {
		sslId := ""
		sslEnabled := false
		vhostPorts := []int32{80}
		upstreamId := fmt.Sprintf("upstream-zone-%s", sKey)
		for _, ip := range service.NodeIps {
			upstreams = append(upstreams, client.L7UpstreamInput{
				IP:     ip,
				Weight: 1,
				Port:   int32(service.NodePort),
			})
		}

		for _, host := range service.Hosts {
			if id, ok := sslCerts[host]; ok {
				sslId = id
				sslEnabled = true
				vhostPorts = []int32{443}
			}
			locationZones = append(locationZones, client.L7LocationZoneInput{
				Location:   "/",
				UpstreamID: upstreamId,
			})
		}

		vZInput := client.L7VHostZoneInput{
			ID:            fmt.Sprintf("vhost-zone-%s", sKey),
			Domains:       service.Hosts,
			SSLCertID:     sslId,
			SSL:           sslEnabled,
			Ports:         vhostPorts,
			LocationZones: locationZones,
		}
		vZInput = *annotations.FillLBVHostZoneWithServiceAnnotations(&vZInput, service.Annotations)
		vhostZones = append(vhostZones, vZInput)

		uZInput := client.L7UpstreamZoneInput{
			ID:        upstreamId,
			Upstreams: upstreams,
		}
		uZInput = *annotations.FillLBUpstreamZoneWithServiceAnnotations(&uZInput, service.Annotations)
		upstreamZones = append(upstreamZones, uZInput)
	}
	locIdStr := config.FetchEnv("SC_LOCATION_ID", "1")
	locId, err := strconv.Atoi(locIdStr)
	if err != nil {
		klog.Errorf("can't convert SC_LOCATION_ID to int: %v", err)
		locId = 1
	}

	lbInput := &client.L7LoadBalancerCreateInput{
		Name:          GetLoadBalancerName(ingress),
		LocationID:    int64(locId),
		UpstreamZones: upstreamZones,
		VHostZones:    vhostZones,
	}
	lbInput = annotations.FillLBWithIngressAnnotations(lbInput, ingress.Annotations)

	return lbInput, nil
}
