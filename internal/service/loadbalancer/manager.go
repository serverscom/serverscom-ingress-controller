package loadbalancer

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/serverscom/serverscom-ingress-controller/internal/config"
	"github.com/serverscom/serverscom-ingress-controller/internal/ingress/controller/store"
	"github.com/serverscom/serverscom-ingress-controller/internal/service/annotations"

	serverscom "github.com/serverscom/serverscom-go-client/pkg"
	networkv1 "k8s.io/api/networking/v1"
	"k8s.io/klog/v2"
)

//go:generate mockgen --destination ../../mocks/lb_manager.go --package=mocks --source manager.go

// ManagerInterface describes an interface to manage load balancers
type LBManagerInterface interface {
	HasRegistration(name string) bool
	NewLoadBalancer(input *serverscom.L7LoadBalancerCreateInput) (*serverscom.L7LoadBalancer, error, bool)
	DeleteLoadBalancer(name string) error
	UpdateLoadBalancer(input *serverscom.L7LoadBalancerUpdateInput) (*serverscom.L7LoadBalancer, error, bool)
	GetIds() []string
	TranslateIngressToLB(ingress *networkv1.Ingress, sslCerts map[string]string) (*serverscom.L7LoadBalancerCreateInput, error)
	GetLoadBalancer(name string) (*serverscom.L7LoadBalancer, error)
}

// Manager represents a load balancer manager
type Manager struct {
	resources map[string]*LoadBalancer

	lock   sync.Mutex
	client *serverscom.Client
	store  store.Storer
}

// NewManager creates a load balancer manager
func NewManager(client *serverscom.Client, store store.Storer) *Manager {
	return &Manager{
		resources: make(map[string]*LoadBalancer),
		client:    client,
		store:     store,
	}
}

// HasRegistration checks if lb manager has load balancer with specified name
func (m *Manager) HasRegistration(name string) bool {
	m.lock.Lock()
	defer m.lock.Unlock()

	_, ok := m.resources[name]
	return ok
}

// NewLoadBalancer creates a new load balancer in portal from input if it doesn't exists in portal, otherwise update it
// Updates load balancer state in manager
func (m *Manager) NewLoadBalancer(input *serverscom.L7LoadBalancerCreateInput) (*serverscom.L7LoadBalancer, error, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	lb := NewLoadBalancer(m.client.LoadBalancers, input)
	lb.Find(input.Name)
	l7, err := lb.Sync()

	if err != nil {
		return nil, err, false
	}

	m.resources[input.Name] = lb
	return l7, nil, true
}

// DeleteLoadBalancer deletes load balancer from portal and manager
func (m *Manager) DeleteLoadBalancer(name string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	lb, ok := m.resources[name]

	if !ok {
		return fmt.Errorf("can't find resource: %s", name)
	}

	lb.MarkAsDeleted()

	_, err := lb.Sync()

	if err != nil {
		return err
	}

	delete(m.resources, name)

	return nil
}

// UpdateLoadBalancer updates load balancer in portal and manager.
func (m *Manager) UpdateLoadBalancer(input *serverscom.L7LoadBalancerUpdateInput) (*serverscom.L7LoadBalancer, error, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	lb, ok := m.resources[input.Name]

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
		m.resources[input.Name] = lbCopy

		return nil, err, false
	}

	return l7, nil, true
}

func (m *Manager) GetIds() []string {
	var ids []string

	for k := range m.resources {
		ids = append(ids, k)
	}

	return ids
}

// TranslateIngressToLB maps an Ingress to L7 LB object and fills annotations
func (m *Manager) TranslateIngressToLB(ingress *networkv1.Ingress, sslCerts map[string]string) (*serverscom.L7LoadBalancerCreateInput, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	sInfo, err := m.store.GetIngressServiceInfo(ingress)
	if err != nil {
		return nil, err
	}

	var upstreamZones []serverscom.L7UpstreamZoneInput
	var upstreams []serverscom.L7UpstreamInput
	var vhostZones []serverscom.L7VHostZoneInput
	var locationZones []serverscom.L7LocationZoneInput

	for sKey, service := range sInfo {
		sslId := ""
		sslEnabled := false
		vhostPorts := []int32{80}
		upstreamId := fmt.Sprintf("upstream-zone-%s", sKey)
		for _, ip := range service.NodeIps {
			upstreams = append(upstreams, serverscom.L7UpstreamInput{
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
			locationZones = append(locationZones, serverscom.L7LocationZoneInput{
				Location:   "/",
				UpstreamID: upstreamId,
			})
		}

		vZInput := serverscom.L7VHostZoneInput{
			ID:            fmt.Sprintf("vhost-zone-%s", sKey),
			Domains:       service.Hosts,
			SSLCertID:     sslId,
			SSL:           sslEnabled,
			Ports:         vhostPorts,
			LocationZones: locationZones,
		}
		vZInput = *annotations.FillLBVHostZoneWithServiceAnnotations(&vZInput, service.Annotations)
		vhostZones = append(vhostZones, vZInput)

		uZInput := serverscom.L7UpstreamZoneInput{
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

	lbInput := &serverscom.L7LoadBalancerCreateInput{
		Name:          GetLoadBalancerName(ingress),
		LocationID:    int64(locId),
		UpstreamZones: upstreamZones,
		VHostZones:    vhostZones,
	}
	lbInput, err = annotations.FillLBWithIngressAnnotations(lbInput, ingress.Annotations)

	return lbInput, err
}

// GetLoadBalancer get load balancer from api
func (m *Manager) GetLoadBalancer(name string) (*serverscom.L7LoadBalancer, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	lb, ok := m.resources[name]

	if !ok {
		return nil, fmt.Errorf("can't find resource: %s", name)
	}
	return lb.Get()
}
