package sync

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
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

	syncManager := New(nil, lbManagerHandler, nil, nil)

	lbInput := &serverscom.L7LoadBalancerCreateInput{
		Name: "test-lb",
	}
	t.Run("Existing Load Balancer", func(t *testing.T) {
		g := NewWithT(t)
		lbManagerHandler.EXPECT().HasRegistration(lbInput.Name).Return(true)
		lbManagerHandler.EXPECT().UpdateLoadBalancer(gomock.Any()).Return(nil, nil, true)

		_, err := syncManager.SyncL7LB(lbInput)
		g.Expect(err).To(BeNil())
	})

	t.Run("New Load Balancer", func(t *testing.T) {
		g := NewWithT(t)
		lbManagerHandler.EXPECT().HasRegistration(lbInput.Name).Return(false)
		lbManagerHandler.EXPECT().NewLoadBalancer(lbInput).Return(nil, nil, true)

		_, err := syncManager.SyncL7LB(lbInput)
		g.Expect(err).To(BeNil())
	})

	t.Run("Update Load Balancer Error", func(t *testing.T) {
		g := NewWithT(t)
		lbManagerHandler.EXPECT().HasRegistration(lbInput.Name).Return(true)
		lbManagerHandler.EXPECT().UpdateLoadBalancer(gomock.Any()).Return(nil, errors.New("update error"), false)

		_, err := syncManager.SyncL7LB(lbInput)
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("New Load Balancer Error", func(t *testing.T) {
		g := NewWithT(t)
		lbManagerHandler.EXPECT().HasRegistration(lbInput.Name).Return(false)
		lbManagerHandler.EXPECT().NewLoadBalancer(lbInput).Return(nil, errors.New("creation error"), false)

		_, err := syncManager.SyncL7LB(lbInput)
		g.Expect(err).To(HaveOccurred())
	})
}

func TestCleanupLBs(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	storeHandler := mocks.NewMockStorer(mockCtrl)
	lbManagerHandler := mocks.NewMockLBManagerInterface(mockCtrl)

	syncManager := New(nil, lbManagerHandler, storeHandler, nil)

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

func TestSyncStatus(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	lbManagerHandler := mocks.NewMockLBManagerInterface(mockCtrl)

	fakeClock := clockwork.NewFakeClock()
	syncManager := New(nil, lbManagerHandler, nil, fakeClock)

	inProcessLB := &serverscom.L7LoadBalancer{
		Name:              "some-balancer",
		ExternalAddresses: []string{"1.2.3.4"},
		Status:            "in_process",
	}
	activeLB := &serverscom.L7LoadBalancer{
		Name:              "some-balancer",
		ExternalAddresses: []string{"1.2.3.4"},
		Status:            "active",
	}

	t.Run("Sync successful", func(t *testing.T) {
		g := NewWithT(t)

		lbManagerHandler.EXPECT().GetLoadBalancer(inProcessLB.Name).Return(activeLB, nil)
		var wg sync.WaitGroup
		wg.Add(1)
		var result *serverscom.L7LoadBalancer
		var err error
		go func() {
			defer wg.Done()
			result, err = syncManager.SyncStatus(ctx, inProcessLB)
		}()

		fakeClock.BlockUntil(1)
		fakeClock.Advance(10 * time.Second)
		wg.Wait()

		g.Expect(err).To(BeNil())
		g.Expect(result).To(Equal(activeLB))
	})
	t.Run("Sync successful after 1 retry", func(t *testing.T) {
		g := NewWithT(t)

		lbManagerHandler.EXPECT().GetLoadBalancer(inProcessLB.Name).Return(inProcessLB, nil)
		lbManagerHandler.EXPECT().GetLoadBalancer(inProcessLB.Name).Return(nil, errors.New("API error"))
		lbManagerHandler.EXPECT().GetLoadBalancer(inProcessLB.Name).Return(activeLB, nil)
		var wg sync.WaitGroup
		wg.Add(1)
		var result *serverscom.L7LoadBalancer
		var err error
		go func() {
			defer wg.Done()
			result, err = syncManager.SyncStatus(ctx, inProcessLB)
		}()

		fakeClock.BlockUntil(1)
		fakeClock.Advance(10 * time.Second)
		fakeClock.BlockUntil(1)
		fakeClock.Advance(10 * time.Second)
		fakeClock.BlockUntil(1)
		fakeClock.Advance(10 * time.Second)
		wg.Wait()

		g.Expect(err).To(BeNil())
		g.Expect(result).To(Equal(activeLB))
	})
	t.Run("Sync when poll timeout reached", func(t *testing.T) {
		g := NewWithT(t)

		lbManagerHandler.EXPECT().GetLoadBalancer(inProcessLB.Name).Return(inProcessLB, nil).AnyTimes()
		var wg sync.WaitGroup
		wg.Add(1)
		var result *serverscom.L7LoadBalancer
		var err error
		go func() {
			defer wg.Done()
			result, err = syncManager.SyncStatus(ctx, inProcessLB)
		}()

		fakeClock.BlockUntil(1)
		cancel()
		wg.Wait()

		g.Expect(result).To(BeNil())
		g.Expect(err).To(MatchError(fmt.Errorf("poll LB timeout reached")))
	})
}
