package sync

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"
	serverscom "github.com/serverscom/serverscom-go-client/pkg"
	"github.com/serverscom/serverscom-ingress-controller/internal/mocks"
	"go.uber.org/mock/gomock"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSyncL7LB(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	lbManagerHandler := mocks.NewMockLBManagerInterface(mockCtrl)

	syncManager := New(nil, lbManagerHandler, nil)

	lbInput := &serverscom.L7LoadBalancerCreateInput{
		Name: "test-lb",
	}
	t.Run("Existing Load Balancer", func(t *testing.T) {
		g := NewWithT(t)
		lbManagerHandler.EXPECT().HasRegistration(lbInput.Name).Return(true)
		lbManagerHandler.EXPECT().UpdateLoadBalancer(gomock.Any()).Return(nil, nil, true)

		err := syncManager.SyncL7LB(lbInput)
		g.Expect(err).To(BeNil())
	})

	t.Run("New Load Balancer", func(t *testing.T) {
		g := NewWithT(t)
		lbManagerHandler.EXPECT().HasRegistration(lbInput.Name).Return(false)
		lbManagerHandler.EXPECT().NewLoadBalancer(lbInput).Return(nil, nil, true)

		err := syncManager.SyncL7LB(lbInput)
		g.Expect(err).To(BeNil())
	})

	t.Run("Update Load Balancer Error", func(t *testing.T) {
		g := NewWithT(t)
		lbManagerHandler.EXPECT().HasRegistration(lbInput.Name).Return(true)
		lbManagerHandler.EXPECT().UpdateLoadBalancer(gomock.Any()).Return(nil, errors.New("update error"), false)

		err := syncManager.SyncL7LB(lbInput)
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("New Load Balancer Error", func(t *testing.T) {
		g := NewWithT(t)
		lbManagerHandler.EXPECT().HasRegistration(lbInput.Name).Return(false)
		lbManagerHandler.EXPECT().NewLoadBalancer(lbInput).Return(nil, errors.New("creation error"), false)

		err := syncManager.SyncL7LB(lbInput)
		g.Expect(err).To(HaveOccurred())
	})
}

func TestCleanupLBs(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	storeHandler := mocks.NewMockStorer(mockCtrl)
	lbManagerHandler := mocks.NewMockLBManagerInterface(mockCtrl)

	syncManager := New(nil, lbManagerHandler, storeHandler)

	scClass := "serverscom"
	otherClass := "default"
	allIngresses := []*networkv1.Ingress{
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:  "123",
				Name: "valid-ingress"},
			Spec: networkv1.IngressSpec{
				IngressClassName: &scClass,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:  "456",
				Name: "invalid-ingress"},
			Spec: networkv1.IngressSpec{
				IngressClassName: &otherClass,
			},
		},
	}

	validLBId := "ingress-a123"

	t.Run("Cleanup LBs successfully", func(t *testing.T) {
		g := NewGomegaWithT(t)
		storeHandler.EXPECT().ListIngress().Return(allIngresses)
		lbManagerHandler.EXPECT().GetIds().Return([]string{"invalid-id", validLBId})
		lbManagerHandler.EXPECT().DeleteLoadBalancer("invalid-id").Return(nil)

		err := syncManager.CleanupLBs(scClass)
		g.Expect(err).To(BeNil())
	})

	t.Run("Fail to delete LB", func(t *testing.T) {
		g := NewGomegaWithT(t)
		storeHandler.EXPECT().ListIngress().Return(allIngresses)
		lbManagerHandler.EXPECT().GetIds().Return([]string{"invalid-lb"})
		lbManagerHandler.EXPECT().DeleteLoadBalancer("invalid-lb").Return(errors.New("delete error"))

		err := syncManager.CleanupLBs(scClass)
		g.Expect(err).To(HaveOccurred())
	})
}
