package service

import (
	"errors"
	"testing"
	"time"

	"github.com/serverscom/serverscom-ingress-controller/internal/ingress/controller/store"
	"github.com/serverscom/serverscom-ingress-controller/internal/mocks"
	"golang.org/x/net/context"

	. "github.com/onsi/gomega"
	serverscom "github.com/serverscom/serverscom-go-client/pkg"
	"go.uber.org/mock/gomock"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/record"
)

var (
	scIngressClassName    = "serverscom"
	scCertManagerPrefix   = "sc-certmgr-cert-id-"
	nonScIngressClassName = "not-sc-ingress"
	namespace             = "default"
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
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	storeHandler := mocks.NewMockStorer(mockCtrl)
	lbManagerHandler := mocks.NewMockLBManagerInterface(mockCtrl)
	tlsManagerHandler := mocks.NewMockTLSManagerInterface(mockCtrl)
	syncManagerHandler := mocks.NewMockSyncer(mockCtrl)
	recorder := record.NewFakeRecorder(10)
	fakeClient := fake.NewSimpleClientset()

	srv := New(fakeClient, tlsManagerHandler, lbManagerHandler, storeHandler, syncManagerHandler, recorder, scIngressClassName, scCertManagerPrefix, namespace)
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

		select {
		case e := <-recorder.Events:
			expectedEvent := "Warning Sync fetching object with key ingress from store failed: fetch error"
			g.Expect(e).To(BeEquivalentTo(expectedEvent))
		case <-time.After(time.Second * 1):
			t.Fatal("Timeout waiting for event")
		}
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
		syncManagerHandler.EXPECT().SyncTLS(gomock.Any(), gomock.Any()).Return(nil, errors.New("TLS sync error"))

		err := srv.SyncToPortal("ingress")
		g.Expect(err).To(HaveOccurred())

		select {
		case e := <-recorder.Events:
			expectedEvent := "Warning Sync syncing tls for ingress 'ingress' failed: TLS sync error"
			g.Expect(e).To(BeEquivalentTo(expectedEvent))
		case <-time.After(time.Second * 1):
			t.Fatal("Timeout waiting for event")
		}
	})

	t.Run("Error translating Ingress to LB", func(t *testing.T) {
		g := NewWithT(t)

		storeHandler.EXPECT().GetIngress("ingress").Return(scIngress, nil)
		syncManagerHandler.EXPECT().SyncTLS(gomock.Any(), gomock.Any()).Return(map[string]string{}, nil)
		lbManagerHandler.EXPECT().TranslateIngressToLB(gomock.Any(), gomock.Any()).Return(nil, errors.New("Translate error"))

		err := srv.SyncToPortal("ingress")
		g.Expect(err).To(HaveOccurred())

		select {
		case e := <-recorder.Events:
			expectedEvent := "Warning Translate translate ingress 'ingress' to LB failed: Translate error"
			g.Expect(e).To(BeEquivalentTo(expectedEvent))
		case <-time.After(time.Second * 1):
			t.Fatal("Timeout waiting for event")
		}
	})

	t.Run("Error syncing L7 LB", func(t *testing.T) {
		g := NewWithT(t)

		lbInput := new(serverscom.L7LoadBalancerCreateInput)
		storeHandler.EXPECT().GetIngress("ingress").Return(scIngress, nil)
		syncManagerHandler.EXPECT().SyncTLS(gomock.Any(), gomock.Any()).Return(map[string]string{}, nil)
		lbManagerHandler.EXPECT().TranslateIngressToLB(gomock.Any(), gomock.Any()).Return(lbInput, nil)
		syncManagerHandler.EXPECT().SyncL7LB(gomock.Any()).Return(nil, errors.New("LB sync error"))

		err := srv.SyncToPortal("ingress")
		g.Expect(err).To(HaveOccurred())

		select {
		case e := <-recorder.Events:
			expectedEvent := "Warning Sync syncing LB for ingress 'ingress' failed: LB sync error"
			g.Expect(e).To(BeEquivalentTo(expectedEvent))
		case <-time.After(time.Second * 1):
			t.Fatal("Timeout waiting for event")
		}
	})

	t.Run("Update status error", func(t *testing.T) {
		g := NewWithT(t)

		lbInput := new(serverscom.L7LoadBalancerCreateInput)
		storeHandler.EXPECT().GetIngress("ingress").Return(scIngress, nil)
		syncManagerHandler.EXPECT().SyncTLS(gomock.Any(), gomock.Any()).Return(map[string]string{}, nil)
		lbManagerHandler.EXPECT().TranslateIngressToLB(gomock.Any(), gomock.Any()).Return(lbInput, nil)
		syncManagerHandler.EXPECT().SyncL7LB(gomock.Any()).Return(&serverscom.L7LoadBalancer{ExternalAddresses: []string{"1.2.3.4"}}, nil)

		err := srv.SyncToPortal("ingress")
		g.Expect(err).To(BeNil())

		select {
		case e := <-recorder.Events:
			expectedEvent := `Warning UpdateStatus ingresses.networking.k8s.io "test-ingress" not found`
			g.Expect(e).To(BeEquivalentTo(expectedEvent))
		case <-time.After(time.Second * 1):
			t.Fatal("Timeout waiting for event")
		}
	})

	t.Run("Successful sync", func(t *testing.T) {
		g := NewWithT(t)

		_, err := fakeClient.NetworkingV1().Ingresses(namespace).Create(context.Background(), scIngress, metav1.CreateOptions{})
		g.Expect(err).To(BeNil())

		lbInput := new(serverscom.L7LoadBalancerCreateInput)
		storeHandler.EXPECT().GetIngress("ingress").Return(scIngress, nil)
		syncManagerHandler.EXPECT().SyncTLS(gomock.Any(), gomock.Any()).Return(map[string]string{}, nil)
		lbManagerHandler.EXPECT().TranslateIngressToLB(gomock.Any(), gomock.Any()).Return(lbInput, nil)
		syncManagerHandler.EXPECT().SyncL7LB(gomock.Any()).Return(&serverscom.L7LoadBalancer{ExternalAddresses: []string{"1.2.3.4"}}, nil)

		err = srv.SyncToPortal("ingress")
		g.Expect(err).To(BeNil())

		ing, err := fakeClient.NetworkingV1().Ingresses(namespace).Get(context.Background(), "test-ingress", metav1.GetOptions{})
		g.Expect(err).To(BeNil())
		g.Expect(ing.Status.LoadBalancer.Ingress[0].IP).To(BeEquivalentTo("1.2.3.4"))

		select {
		case e := <-recorder.Events:
			expectedEvent := "Normal Synced Successfully synced to portal"
			g.Expect(e).To(BeEquivalentTo(expectedEvent))
		case <-time.After(time.Second * 1):
			t.Fatal("Timeout waiting for event")
		}
	})
}
