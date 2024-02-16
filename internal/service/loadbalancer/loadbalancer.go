package loadbalancer

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	client "github.com/serverscom/serverscom-go-client/pkg"
)

// LoadBalancer represents a load balancer object for manager
type LoadBalancer struct {
	id string

	state *client.L7LoadBalancer

	createInput   *client.L7LoadBalancerCreateInput
	currentInput  *client.L7LoadBalancerUpdateInput
	previousInput *client.L7LoadBalancerUpdateInput

	deleted bool

	lastRefresh time.Time

	lBService client.LoadBalancersService
}

// NewLoadBalancer creates a new load balancer object
func NewLoadBalancer(lBService client.LoadBalancersService, input *client.L7LoadBalancerCreateInput) *LoadBalancer {
	return &LoadBalancer{
		createInput: input,

		deleted: false,

		lastRefresh: time.Now(),

		lBService: lBService,
	}
}

// Find finds load balancer in portal and sets lb id if found
func (lb *LoadBalancer) Find(name string) bool {
	var query = lb.lBService.
		Collection().
		SetParam("search_pattern", name).
		SetParam("type", "l7")

	if lb.currentInput != nil && lb.createInput.LocationID != 0 {
		query = query.SetParam("location_id", strconv.FormatInt(lb.createInput.LocationID, 10))
	}

	list, err := query.Collect(context.Background())

	if err != nil {
		return false
	}

	for _, candidate := range list {
		if candidate.Name == name {
			lb.id = candidate.ID
			break
		}
	}

	return lb.id != ""
}

// Copy makes a copy of load balancer
func (lb *LoadBalancer) Copy() *LoadBalancer {
	return &LoadBalancer{
		id:            lb.id,
		state:         lb.state,
		currentInput:  lb.currentInput,
		previousInput: lb.previousInput,
		deleted:       lb.deleted,
		lBService:     lb.lBService,
	}
}

// IsChanged returns true if newInput don't match currentInput
func (lb *LoadBalancer) IsChanged(newInput *client.L7LoadBalancerUpdateInput) bool {
	newPayload, err := json.Marshal(newInput)

	if err != nil {
		return false
	}

	currentPayload, err := json.Marshal(lb.currentInput)

	if err != nil {
		return true
	}

	return string(newPayload) != string(currentPayload)
}

// Sync create/update/delete load balancer depending on it state
func (lb *LoadBalancer) Sync() (*client.L7LoadBalancer, error) {
	if lb.deleted {
		return nil, lb.delete()
	}

	if lb.id == "" {
		return lb.create()
	}

	return lb.update()
}

// MarkAsDeleted marks load balancer as deleted
func (lb *LoadBalancer) MarkAsDeleted() {
	lb.deleted = true
}

// UpdateInput saves current input to previous and updates current input with newInput
func (lb *LoadBalancer) UpdateInput(newInput *client.L7LoadBalancerUpdateInput) {
	lb.previousInput = lb.currentInput
	lb.currentInput = newInput
}

// delete deletes load balancer from portal
func (lb *LoadBalancer) delete() error {
	if err := lb.lBService.DeleteL7LoadBalancer(context.Background(), lb.id); err != nil {
		return err
	}

	return nil
}

// create creates load balancer in portal
func (lb *LoadBalancer) create() (*client.L7LoadBalancer, error) {
	l7, err := lb.lBService.CreateL7LoadBalancer(context.Background(), *lb.createInput)

	if err != nil {
		return nil, err
	}

	lb.id = l7.ID
	lb.state = l7
	lb.lastRefresh = time.Now()

	return l7, nil
}

// update updates load balancer in portal
func (lb *LoadBalancer) update() (*client.L7LoadBalancer, error) {
	if lb.currentInput == nil {
		lb.currentInput = &client.L7LoadBalancerUpdateInput{
			Name:              lb.createInput.Name,
			StoreLogs:         lb.createInput.StoreLogs,
			StoreLogsRegionID: lb.createInput.StoreLogsRegionID,
			Geoip:             lb.createInput.Geoip,
			VHostZones:        lb.createInput.VHostZones,
			UpstreamZones:     lb.createInput.UpstreamZones,
		}
	}
	l7, err := lb.lBService.UpdateL7LoadBalancer(context.Background(), lb.id, *lb.currentInput)

	if err != nil {
		return nil, err
	}

	lb.state = l7
	lb.lastRefresh = time.Now()

	return l7, nil
}
