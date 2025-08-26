package loadbalancer

import (
	"errors"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	serverscom "github.com/serverscom/serverscom-go-client/pkg"
	"github.com/serverscom/serverscom-ingress-controller/internal/ingress/controller/store"
	"github.com/serverscom/serverscom-ingress-controller/internal/mocks"
	"github.com/serverscom/serverscom-ingress-controller/internal/service/annotations"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHasRegistration(t *testing.T) {
	g := NewWithT(t)

	manager := NewManager(nil, nil)

	lbName := "test-lb"
	manager.resources[lbName] = &LoadBalancer{
		state: &serverscom.L7LoadBalancer{},
	}

	g.Expect(manager.HasRegistration(lbName)).To(BeTrue())
	g.Expect(manager.HasRegistration("non-exist")).To(BeFalse())
}

func TestNewLoadBalancer(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	lbHandler := mocks.NewMockLoadBalancersService(mockCtrl)
	collectionHandler := mocks.NewMockCollection[serverscom.LoadBalancer](mockCtrl)

	lbHandler.EXPECT().
		Collection().
		Return(collectionHandler).
		AnyTimes()

	collectionHandler.EXPECT().
		SetParam(gomock.Any(), gomock.Any()).
		Return(collectionHandler).
		AnyTimes()

	lbName := "test-lb"
	lbID := "test-id"
	startedTime := time.Now()
	expectedL7LB := &serverscom.L7LoadBalancer{ID: lbID, Name: lbName}

	client := serverscom.NewClientWithEndpoint("", "")
	client.LoadBalancers = lbHandler

	manager := NewManager(client, nil)
	t.Run("Load balancer already exists", func(t *testing.T) {
		g := NewWithT(t)

		collectionHandler.EXPECT().
			Collect(gomock.Any()).
			Return([]serverscom.LoadBalancer{{ID: lbID, Name: lbName}}, nil)

		lbHandler.EXPECT().
			UpdateL7LoadBalancer(gomock.Any(), lbID, serverscom.L7LoadBalancerUpdateInput{Name: lbName}).
			Return(expectedL7LB, nil)

		_, err, _ := manager.NewLoadBalancer(&serverscom.L7LoadBalancerCreateInput{Name: lbName})

		g.Expect(err).To(BeNil())
		g.Expect(manager.resources[lbName].id).To(Equal(lbID))
		g.Expect(manager.resources[lbName].lastRefresh).To(BeTemporally(">", startedTime))
		g.Expect(manager.resources[lbName].state).To(Equal(expectedL7LB))
	})

	t.Run("Load balancer doesn't exists", func(t *testing.T) {
		g := NewWithT(t)

		collectionHandler.EXPECT().
			Collect(gomock.Any()).
			Return([]serverscom.LoadBalancer{}, nil)

		lbHandler.EXPECT().
			CreateL7LoadBalancer(gomock.Any(), serverscom.L7LoadBalancerCreateInput{Name: lbName}).
			Return(expectedL7LB, nil)

		_, err, _ := manager.NewLoadBalancer(&serverscom.L7LoadBalancerCreateInput{Name: lbName})

		g.Expect(err).To(BeNil())
		g.Expect(manager.resources[lbName].id).To(Equal(lbID))
		g.Expect(manager.resources[lbName].lastRefresh).To(BeTemporally(">", startedTime))
		g.Expect(manager.resources[lbName].state).To(Equal(expectedL7LB))
	})
}
func TestUpdateLoadBalancer(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	lbHandler := mocks.NewMockLoadBalancersService(mockCtrl)
	collectionHandler := mocks.NewMockCollection[serverscom.LoadBalancer](mockCtrl)

	lbHandler.EXPECT().
		Collection().
		Return(collectionHandler).
		AnyTimes()

	collectionHandler.EXPECT().
		SetParam(gomock.Any(), gomock.Any()).
		Return(collectionHandler).
		AnyTimes()

	lbName := "test-lb"
	lbID := "test-id"
	startedTime := time.Now()
	expectedL7LB := &serverscom.L7LoadBalancer{ID: lbID, Name: lbName}
	lbL7UpdateInput := &serverscom.L7LoadBalancerUpdateInput{Name: lbName}

	client := serverscom.NewClientWithEndpoint("", "")
	client.LoadBalancers = lbHandler

	manager := NewManager(client, nil)
	manager.resources[lbName] = &LoadBalancer{
		id:           lbID,
		state:        expectedL7LB,
		currentInput: lbL7UpdateInput,
		lBService:    lbHandler,
	}
	t.Run("Load balancer not found", func(t *testing.T) {
		g := NewWithT(t)

		_, err, _ := manager.UpdateLoadBalancer(&serverscom.L7LoadBalancerUpdateInput{Name: "not-exist"})

		g.Expect(err).To(MatchError(fmt.Errorf("can't find resource: not-exist")))
	})

	t.Run("Load balancer not changed", func(t *testing.T) {
		g := NewWithT(t)

		lb, err, updated := manager.UpdateLoadBalancer(lbL7UpdateInput)

		g.Expect(lb).To(Equal(expectedL7LB))
		g.Expect(updated).To(Equal(false))
		g.Expect(err).To(BeNil())
	})

	t.Run("Load balancer updated", func(t *testing.T) {
		g := NewWithT(t)

		geoip := true
		newUpdateInput := serverscom.L7LoadBalancerUpdateInput{Name: lbName, Geoip: &geoip}
		lbHandler.EXPECT().
			UpdateL7LoadBalancer(gomock.Any(), lbID, newUpdateInput).
			Return(expectedL7LB, nil)

		lb, err, updated := manager.UpdateLoadBalancer(&newUpdateInput)

		g.Expect(lb).To(Equal(expectedL7LB))
		g.Expect(updated).To(Equal(true))
		g.Expect(err).To(BeNil())
		g.Expect(manager.resources[lbName].lastRefresh).To(BeTemporally(">", startedTime))
	})

	t.Run("Load balancer deleted", func(t *testing.T) {
		g := NewWithT(t)

		manager.resources[lbName].deleted = true
		lb, err, updated := manager.UpdateLoadBalancer(lbL7UpdateInput)

		g.Expect(lb).To(Equal(expectedL7LB))
		g.Expect(updated).To(Equal(false))
		g.Expect(err).To(BeNil())
	})
}
func TestDeleteLoadBalancer(t *testing.T) {
	g := NewGomegaWithT(t)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	lbHandler := mocks.NewMockLoadBalancersService(mockCtrl)
	lbName := "test-lb"
	lbID := "test-id"
	expectedL7LB := &serverscom.L7LoadBalancer{ID: lbID, Name: lbName}

	client := serverscom.NewClientWithEndpoint("", "")
	client.LoadBalancers = lbHandler
	manager := NewManager(client, nil)
	manager.resources[lbName] = &LoadBalancer{
		id:        lbID,
		state:     expectedL7LB,
		lBService: lbHandler,
		deleted:   false,
	}

	t.Run("Load balancer not found", func(t *testing.T) {
		err := manager.DeleteLoadBalancer("not-exist")

		g.Expect(err).To(MatchError(fmt.Errorf("can't find resource: not-exist")))
	})

	t.Run("Error during sync", func(t *testing.T) {
		lbHandler.EXPECT().DeleteL7LoadBalancer(gomock.Any(), lbID).Return(fmt.Errorf("error"))

		err := manager.DeleteLoadBalancer(lbName)

		g.Expect(err).To(HaveOccurred())
		_, ok := manager.resources[lbName]
		g.Expect(ok).To(BeTrue())
	})

	t.Run("Load balancer deleted successfully", func(t *testing.T) {
		lbHandler.EXPECT().DeleteL7LoadBalancer(gomock.Any(), lbID).Return(nil)

		err := manager.DeleteLoadBalancer(lbName)

		g.Expect(err).To(BeNil())
		_, ok := manager.resources[lbName]
		g.Expect(ok).To(BeFalse())
	})
}

func TestGetIds(t *testing.T) {
	g := NewGomegaWithT(t)
	manager := NewManager(nil, nil)
	ids := manager.GetIds()
	g.Expect(ids).To(BeEmpty())

	manager.resources["lb1"] = &LoadBalancer{}
	manager.resources["lb2"] = &LoadBalancer{}

	ids = manager.GetIds()
	g.Expect(ids).To(ConsistOf("lb1", "lb2"))
}

func TestTranslateIngressToLB(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ingressClassName := "sc-ingress"
	ingress := &networkv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			UID:         "123",
			Name:        "test-ingress",
			Annotations: map[string]string{annotations.LBGeoIPEnabled: "true"},
		},
		Spec: networkv1.IngressSpec{
			IngressClassName: &ingressClassName,
		},
	}
	sslCerts := map[string]string{
		"example.com": "ssl-cert-id",
		"foo.com":     "ssl-cert-foo",
	}

	hostsInfo := map[string]store.HostInfo{
		"example.com": {
			Paths: []store.PathInfo{
				{
					Path:     "/api",
					NodePort: 30000,
					NodeIps:  []string{"192.168.1.1"},
					Service: &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "service-key",
							Annotations: map[string]string{annotations.LBBalancingAlgorithm: "round-robin"},
						},
						Spec: corev1.ServiceSpec{
							Ports: []corev1.ServicePort{{Port: 80, NodePort: 30000}},
						},
					},
				},
				{
					Path:     "/local",
					NodePort: 30001,
					NodeIps:  []string{"192.168.1.1"},
					Service: &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "service-key2",
							Annotations: map[string]string{annotations.LBBalancingAlgorithm: "least-connections"},
						},
						Spec: corev1.ServiceSpec{
							Ports: []corev1.ServicePort{{Port: 81, NodePort: 30001}},
						},
					},
				},
			},
		},
		"foo.com": {
			Paths: []store.PathInfo{
				{
					Path:     "/",
					NodePort: 30002,
					NodeIps:  []string{"192.168.1.2"},
					Service: &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "service-foo",
							Annotations: map[string]string{annotations.LBBalancingAlgorithm: "round-robin"},
						},
						Spec: corev1.ServiceSpec{
							Ports: []corev1.ServicePort{{Port: 80, NodePort: 30002}},
						},
					},
				},
			},
		},
	}

	storeHandler := mocks.NewMockStorer(mockCtrl)
	lbHandler := mocks.NewMockLoadBalancersService(mockCtrl)

	client := serverscom.NewClientWithEndpoint("", "")
	client.LoadBalancers = lbHandler
	manager := NewManager(client, storeHandler)

	t.Run("Translate ingress to lb input successfully", func(t *testing.T) {
		g := NewWithT(t)
		storeHandler.EXPECT().GetIngressHostsInfo(ingress).Return(hostsInfo, nil)
		lbInput, err := manager.TranslateIngressToLB(ingress, sslCerts)
		g.Expect(err).To(BeNil())
		g.Expect(lbInput).NotTo(BeNil())

		g.Expect(lbInput.VHostZones).To(HaveLen(2))

		for _, vz := range lbInput.VHostZones {
			expectedID := fmt.Sprintf("vhost-zone-%s", vz.Domains[0])
			g.Expect(vz.ID).To(Equal(expectedID))

			g.Expect(vz.Domains).ToNot(BeEmpty())
			g.Expect(vz.LocationZones).ToNot(BeEmpty())
			g.Expect(vz.SSLCertID).ToNot(BeEmpty())

			switch vz.Domains[0] {
			case "example.com":
				g.Expect(vz.SSLCertID).To(Equal("ssl-cert-id"))
				g.Expect(vz.LocationZones).To(HaveLen(2))
				g.Expect(vz.LocationZones[0].Location).To(Equal("/api"))
				g.Expect(vz.LocationZones[1].Location).To(Equal("/local"))
			case "foo.com":
				g.Expect(vz.SSLCertID).To(Equal("ssl-cert-foo"))
				g.Expect(vz.LocationZones).To(HaveLen(1))
				g.Expect(vz.LocationZones[0].Location).To(Equal("/"))
			default:
				t.Fatalf("unexpected domain %s", vz.Domains[0])
			}
		}

		expectedAlgorithmMethods := map[string]string{
			"upstream-zone-service-key-30000":  "round-robin",
			"upstream-zone-service-key2-30001": "least-connections",
			"upstream-zone-service-foo-30002":  "round-robin",
		}

		g.Expect(lbInput.UpstreamZones).To(HaveLen(3))
		upstreamIDs := make(map[string]struct{})
		for _, uz := range lbInput.UpstreamZones {
			if expected, ok := expectedAlgorithmMethods[uz.ID]; ok {
				g.Expect(uz.Method).ToNot(BeNil())
				g.Expect(*uz.Method).To(Equal(expected))
			}
			upstreamIDs[uz.ID] = struct{}{}
			for _, u := range uz.Upstreams {
				g.Expect([]string{"192.168.1.1", "192.168.1.2"}).To(ContainElement(u.IP))
				g.Expect(u.Weight).To(Equal(int32(1)))
			}
		}
		for _, host := range hostsInfo {
			for _, p := range host.Paths {
				upID := fmt.Sprintf("upstream-zone-%s-%d", p.Service.Name, p.NodePort)
				_, exists := upstreamIDs[upID]
				g.Expect(exists).To(BeTrue(), "upstream %s should exist", upID)
			}
		}

		expectedLBName := "ingress-a123"
		g.Expect(lbInput.Name).To(Equal(expectedLBName))
		g.Expect(*lbInput.Geoip).To(Equal(true))
	})

	t.Run("Services info fails", func(t *testing.T) {
		g := NewWithT(t)
		storeHandler.EXPECT().GetIngressHostsInfo(ingress).Return(nil, errors.New("error"))
		lbInput, err := manager.TranslateIngressToLB(ingress, sslCerts)
		g.Expect(err).To(HaveOccurred())
		g.Expect(lbInput).To(BeNil())
	})

	t.Run("Services info is empty", func(t *testing.T) {
		g := NewWithT(t)
		storeHandler.EXPECT().GetIngressHostsInfo(ingress).Return(make(map[string]store.HostInfo), nil)
		lbInput, err := manager.TranslateIngressToLB(ingress, sslCerts)
		expectedErr := errors.New("vhost or upstream can't be empty, can't continue")
		g.Expect(err).To(Equal(expectedErr))
		g.Expect(lbInput).To(BeNil())
	})
}

func TestGetLoadBalancer(t *testing.T) {
	g := NewGomegaWithT(t)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	lbHandler := mocks.NewMockLoadBalancersService(mockCtrl)
	lbName := "test-lb"
	lbID := "test-id"
	expectedL7LB := &serverscom.L7LoadBalancer{ID: lbID, Name: lbName}

	client := serverscom.NewClientWithEndpoint("", "")
	client.LoadBalancers = lbHandler
	manager := NewManager(client, nil)
	manager.resources[lbName] = &LoadBalancer{
		id:        lbID,
		state:     expectedL7LB,
		lBService: lbHandler,
		deleted:   false,
	}

	t.Run("Not found", func(t *testing.T) {
		_, err := manager.GetLoadBalancer("not-exist")

		g.Expect(err).To(MatchError(fmt.Errorf("can't find resource: not-exist")))
	})

	t.Run("Successfull get", func(t *testing.T) {
		lbHandler.EXPECT().GetL7LoadBalancer(gomock.Any(), lbID).Return(expectedL7LB, nil)

		result, err := manager.GetLoadBalancer(lbName)

		g.Expect(err).To(BeNil())
		g.Expect(result).To(BeEquivalentTo(expectedL7LB))
	})
}
