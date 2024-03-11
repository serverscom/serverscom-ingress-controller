package annotations

import (
	"context"
	"errors"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	serverscom "github.com/serverscom/serverscom-go-client/pkg"
	"github.com/serverscom/serverscom-ingress-controller/internal/mocks"
	"go.uber.org/mock/gomock"
)

func TestGetRegionIDByCode(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	cloudRegionsHandler := mocks.NewMockCloudComputingRegionsService(mockCtrl)
	collectionHandler := mocks.NewMockCollection[serverscom.CloudComputingRegion](mockCtrl)

	cloudRegionsHandler.EXPECT().
		Collection().
		Return(collectionHandler).
		AnyTimes()

	t.Run("Regions service returns error", func(t *testing.T) {
		g := NewWithT(t)
		collectionHandler.EXPECT().
			Collect(gomock.Any()).
			Return(nil, errors.New("error"))

		id, err := GetRegionIDByCode(context.Background(), cloudRegionsHandler, "")
		g.Expect(err).To(BeEquivalentTo(errors.New("error")))
		g.Expect(id).To(BeEquivalentTo(0))
	})
	t.Run("Region not found", func(t *testing.T) {
		g := NewWithT(t)
		collectionHandler.EXPECT().
			Collect(gomock.Any()).
			Return(nil, nil)

		id, err := GetRegionIDByCode(context.Background(), cloudRegionsHandler, "notexist")
		expectedError := fmt.Errorf("region with code 'notexist' not found")
		g.Expect(err).To(BeEquivalentTo(expectedError))
		g.Expect(id).To(BeEquivalentTo(0))
	})

	t.Run("Region found", func(t *testing.T) {
		g := NewWithT(t)
		cloudRegion := serverscom.CloudComputingRegion{ID: 1, Name: "test", Code: "test1"}
		collectionHandler.EXPECT().
			Collect(gomock.Any()).
			Return([]serverscom.CloudComputingRegion{cloudRegion}, nil)

		id, err := GetRegionIDByCode(context.Background(), cloudRegionsHandler, "test1")
		g.Expect(err).To(BeNil())
		g.Expect(id).To(BeEquivalentTo(1))
	})
}
