package annotations

import (
	"testing"

	. "github.com/onsi/gomega"
	serverscom "github.com/serverscom/serverscom-go-client/pkg"
)

func TestFillLBWithIngressAnnotations(t *testing.T) {
	g := NewWithT(t)

	lbInput := &serverscom.L7LoadBalancerCreateInput{
		UpstreamZones: []serverscom.L7UpstreamZoneInput{
			{TLSPreset: new(string)},
			{TLSPreset: new(string)},
		},
	}

	invalidAnnotations := map[string]string{
		LBStoreLogsRegionCode: "",
		LBGeoIPEnabled:        "invalid",
	}
	result := FillLBWithIngressAnnotations(lbInput, invalidAnnotations)
	g.Expect(result.StoreLogsRegionID).To(BeNil())
	g.Expect(result.Geoip).To(BeNil())

	annotations := map[string]string{
		LBStoreLogsRegionCode: "1",
		LBGeoIPEnabled:        "true",
		LBMinTLSVersion:       "TLSv1.3",
	}

	result = FillLBWithIngressAnnotations(lbInput, annotations)
	g.Expect(result).NotTo(BeNil())

	storeLogsRegionID := 1
	g.Expect(*result.StoreLogsRegionID).To(Equal(storeLogsRegionID))
	g.Expect(*result.Geoip).To(BeTrue())
	for _, uz := range result.UpstreamZones {
		g.Expect(*uz.TLSPreset).To(Equal("TLSv1.3"))
	}
}
