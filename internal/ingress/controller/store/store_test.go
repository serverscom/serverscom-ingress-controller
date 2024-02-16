package store

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	scIngressClassName    = "sc-ingress"
	nonScIngressClassName = "not-sc-ingress"
	scIngress             = &networkv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ingress",
		},
		Spec: networkv1.IngressSpec{
			IngressClassName: &scIngressClassName,
		},
	}
	nonScIngress = &networkv1.Ingress{
		Spec: networkv1.IngressSpec{
			IngressClassName: &nonScIngressClassName,
		},
	}
)

func TestGetIngress(t *testing.T) {
	g := NewWithT(t)
	s := New("", time.Second, nil, "", nil, nil)
	s.listers.Ingress.Add(scIngress)

	ingress, err := s.GetIngress("test-ingress")
	g.Expect(err).To(BeNil())
	g.Expect(ingress).To(Equal(scIngress))
}

func TestGetSecret(t *testing.T) {
	g := NewWithT(t)
	s := New("", time.Second, nil, "", nil, nil)

	testSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
	}
	s.listers.Secret.Add(testSecret)

	secret, err := s.GetSecret("default/test-secret")
	g.Expect(err).To(BeNil())
	g.Expect(secret).To(Equal(testSecret))
}

func TestListIngress(t *testing.T) {
	g := NewWithT(t)
	s := New("", time.Second, nil, "", nil, nil)

	s.listers.Ingress.Add(scIngress)
	s.listers.Ingress.Add(nonScIngress)

	ingresses := s.ListIngress()
	g.Expect(ingresses).To(HaveLen(2))
	g.Expect(ingresses).To(ContainElements(scIngress, nonScIngress))
}

func TestGetService(t *testing.T) {
	g := NewWithT(t)
	s := New("", time.Second, nil, "", nil, nil)

	testService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
		},
	}
	s.listers.Service.Add(testService)

	service, err := s.GetService("default/test-service")
	g.Expect(err).To(BeNil())
	g.Expect(service).To(Equal(testService))
}

func TestGetNodesIpList(t *testing.T) {
	g := NewWithT(t)
	s := New("", time.Second, nil, "", nil, nil)

	masterNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "master",
			Labels: map[string]string{
				MasterNodeAnnotationKey: "",
			},
		},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				{Type: corev1.NodeInternalIP, Address: "192.168.1.1"},
			},
		},
	}
	s.listers.Node.Add(masterNode)

	workerNode1 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node1",
		},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				{Type: corev1.NodeInternalIP, Address: "192.168.1.2"},
			},
		},
	}
	s.listers.Node.Add(workerNode1)

	workerNode2 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node2",
		},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				{Type: corev1.NodeInternalIP, Address: "192.168.1.3"},
			},
		},
	}
	s.listers.Node.Add(workerNode2)

	nodesIpList := s.GetNodesIpList()
	g.Expect(nodesIpList).NotTo(ContainElement("192.168.1.1"))
	g.Expect(nodesIpList).To(ConsistOf("192.168.1.2", "192.168.1.3"))
}

func TestGetIngressServiceInfo(t *testing.T) {
	g := NewWithT(t)
	s := New("", time.Second, nil, "", nil, nil)

	node1 := &corev1.Node{
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{{Type: corev1.NodeInternalIP, Address: "192.168.1.1"}},
		},
	}
	s.listers.Node.Add(node1)

	serviceName := "test-service"
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: "default",
			Annotations: map[string]string{
				"key": "value",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:     80,
					NodePort: 30000,
				},
			},
		},
	}
	s.listers.Service.Add(service)

	ingress := &networkv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: "default",
		},
		Spec: networkv1.IngressSpec{
			Rules: []networkv1.IngressRule{
				{
					Host: "example.com",
					IngressRuleValue: networkv1.IngressRuleValue{
						HTTP: &networkv1.HTTPIngressRuleValue{
							Paths: []networkv1.HTTPIngressPath{
								{
									Path: "/",
									Backend: networkv1.IngressBackend{
										Service: &networkv1.IngressServiceBackend{
											Name: serviceName,
											Port: networkv1.ServiceBackendPort{Number: 80},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	serviceInfo, err := s.GetIngressServiceInfo(ingress)
	g.Expect(err).To(BeNil())

	g.Expect(serviceInfo).To(HaveKey(serviceName))
	g.Expect(serviceInfo[serviceName].Hosts).To(ContainElement("example.com"))
	g.Expect(serviceInfo[serviceName].NodePort).To(Equal(30000))
	g.Expect(serviceInfo[serviceName].NodeIps).To(ConsistOf("192.168.1.1"))
	g.Expect(serviceInfo[serviceName].Annotations).To(HaveKeyWithValue("key", "value"))
}
