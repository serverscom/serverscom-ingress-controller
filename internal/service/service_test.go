package service

import (
	"errors"
	"os"
	"testing"

	"github.com/serverscom/serverscom-ingress-controller/internal/config"
	"github.com/serverscom/serverscom-ingress-controller/internal/ingress/controller/store"
	"github.com/serverscom/serverscom-ingress-controller/internal/mocks"

	. "github.com/onsi/gomega"
	serverscom "github.com/serverscom/serverscom-go-client/pkg"
	"go.uber.org/mock/gomock"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	scIngressClassName    = "serverscom"
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

func TestSyncToPortal(t *testing.T) {
	g := NewWithT(t)

	os.Setenv("SC_ACCESS_TOKEN", "123")
	scClient, err := config.NewServerscomClient()
	g.Expect(err).To(BeNil())

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	storeHandler := mocks.NewMockStorer(mockCtrl)
	lbHandler := mocks.NewMockLoadBalancersService(mockCtrl)
	sslHandler := mocks.NewMockSSLCertificatesService(mockCtrl)
	lbManagerHandler := mocks.NewMockLBManagerInterface(mockCtrl)
	tlsManagerHandler := mocks.NewMockTLSManagerInterface(mockCtrl)
	scClient.LoadBalancers = lbHandler
	scClient.SSLCertificates = sslHandler
	syncManagerHandler := mocks.NewMockSyncer(mockCtrl)

	srv := New(scClient, tlsManagerHandler, lbManagerHandler, storeHandler, syncManagerHandler, nil, scIngressClassName)
	t.Run("Ingress does not exist", func(t *testing.T) {
		g := NewWithT(t)

		storeHandler.EXPECT().GetIngress("ingress").Return(nil, store.NotExistsError("error"))
		syncManagerHandler.EXPECT().CleanupLBs(scIngressClassName).Return(nil)

		err := srv.SyncToPortal("ingress")
		g.Expect(err).To(BeNil())
	})

	t.Run("Error fetching ingress", func(t *testing.T) {
		g := NewWithT(t)

		expectedError := errors.New("fetch error")
		storeHandler.EXPECT().GetIngress("ingress").Return(nil, expectedError)

		err := srv.SyncToPortal("ingress")
		g.Expect(err).To(MatchError(expectedError))
	})

	t.Run("Ingress class was changed", func(t *testing.T) {
		g := NewWithT(t)

		storeHandler.EXPECT().GetIngress("ingress").Return(nonScIngress, nil)
		syncManagerHandler.EXPECT().CleanupLBs(scIngressClassName).Return(nil)

		err := srv.SyncToPortal("ingress")
		g.Expect(err).To(BeNil())
	})

	t.Run("Error syncing TLS", func(t *testing.T) {
		g := NewWithT(t)

		storeHandler.EXPECT().GetIngress("ingress").Return(scIngress, nil)
		syncManagerHandler.EXPECT().SyncTLS(gomock.Any()).Return(nil, errors.New("TLS sync error"))

		err := srv.SyncToPortal("ingress")
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("Error translating Ingress to LB", func(t *testing.T) {
		g := NewWithT(t)

		storeHandler.EXPECT().GetIngress("ingress").Return(scIngress, nil)
		syncManagerHandler.EXPECT().SyncTLS(gomock.Any()).Return(map[string]string{}, nil)
		lbManagerHandler.EXPECT().TranslateIngressToLB(gomock.Any(), gomock.Any()).Return(nil, errors.New("Translate error"))

		err := srv.SyncToPortal("ingress")
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("Error syncing L7 LB", func(t *testing.T) {
		g := NewWithT(t)

		lbInput := new(serverscom.L7LoadBalancerCreateInput)
		storeHandler.EXPECT().GetIngress("ingress").Return(scIngress, nil)
		syncManagerHandler.EXPECT().SyncTLS(gomock.Any()).Return(map[string]string{}, nil)
		lbManagerHandler.EXPECT().TranslateIngressToLB(gomock.Any(), gomock.Any()).Return(lbInput, nil)
		syncManagerHandler.EXPECT().SyncL7LB(gomock.Any()).Return(errors.New("LB sync error"))

		err := srv.SyncToPortal("ingress")
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("Successful sync", func(t *testing.T) {
		g := NewWithT(t)

		lbInput := new(serverscom.L7LoadBalancerCreateInput)
		storeHandler.EXPECT().GetIngress("ingress").Return(scIngress, nil)
		syncManagerHandler.EXPECT().SyncTLS(gomock.Any()).Return(map[string]string{}, nil)
		lbManagerHandler.EXPECT().TranslateIngressToLB(gomock.Any(), gomock.Any()).Return(lbInput, nil)
		syncManagerHandler.EXPECT().SyncL7LB(gomock.Any()).Return(nil)

		err := srv.SyncToPortal("ingress")
		g.Expect(err).To(BeNil())
	})
}
