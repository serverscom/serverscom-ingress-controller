package annotations

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	serverscom "github.com/serverscom/serverscom-go-client/pkg"
	"github.com/serverscom/serverscom-ingress-controller/internal/mocks"
	"go.uber.org/mock/gomock"
)

func TestFillLBWithIngressAnnotations(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	cloudRegionsHandler := mocks.NewMockCloudComputingRegionsService(mockCtrl)
	collectionHandler := mocks.NewMockCollection[serverscom.CloudComputingRegion](mockCtrl)

	cloudRegionsHandler.EXPECT().
		Collection().
		Return(collectionHandler).
		AnyTimes()

	cloudRegion := serverscom.CloudComputingRegion{ID: 1, Name: "test", Code: "test1"}
	collectionHandler.EXPECT().
		Collect(gomock.Any()).
		Return([]serverscom.CloudComputingRegion{cloudRegion}, nil).
		AnyTimes()

	client := serverscom.NewClientWithEndpoint("", "")
	client.CloudComputingRegions = cloudRegionsHandler

	lbInput := &serverscom.L7LoadBalancerCreateInput{
		UpstreamZones: []serverscom.L7UpstreamZoneInput{
			{TLSPreset: new(string)},
			{TLSPreset: new(string)},
		},
	}
	t.Run("Not existing region code", func(t *testing.T) {
		g := NewWithT(t)
		annotations := map[string]string{
			LBStoreLogsRegionCode: "notexist",
		}
		result, err := FillLBWithIngressAnnotations(client, lbInput, annotations)
		expectedError := fmt.Errorf("cloud region with code 'notexist' not found")
		g.Expect(err).To(BeEquivalentTo(expectedError))
		g.Expect(result).NotTo(BeNil())
	})

	t.Run("Invalid geoipEnabled value", func(t *testing.T) {
		g := NewWithT(t)
		annotations := map[string]string{
			LBGeoIPEnabled: "invalid",
		}
		result, err := FillLBWithIngressAnnotations(client, lbInput, annotations)
		g.Expect(err.Error()).To(BeEquivalentTo(`strconv.ParseBool: parsing "invalid": invalid syntax`))
		g.Expect(result).NotTo(BeNil())
	})

	t.Run("Valid annotations", func(t *testing.T) {
		g := NewWithT(t)
		annotations := map[string]string{
			LBStoreLogsRegionCode: "Test1",
			LBGeoIPEnabled:        "true",
			LBMinTLSVersion:       "TLSv1.3",
		}

		result, err := FillLBWithIngressAnnotations(client, lbInput, annotations)
		g.Expect(err).To(BeNil())
		g.Expect(result).NotTo(BeNil())

		g.Expect(*result.StoreLogsRegionID).To(Equal(1))
		g.Expect(*result.Geoip).To(BeTrue())
		for _, uz := range result.UpstreamZones {
			g.Expect(*uz.TLSPreset).To(Equal("TLSv1.3"))
		}
	})
}
