package annotations

import (
	"testing"

	. "github.com/onsi/gomega"
	serverscom "github.com/serverscom/serverscom-go-client/pkg"
)

func TestFillLBWithIngressAnnotations(t *testing.T) {
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
		result, err := FillLBWithIngressAnnotations(lbInput, annotations)
		g.Expect(err).To(BeNil())
		g.Expect(result.StoreLogs).To(BeNil())
		g.Expect(result.StoreLogsRegionID).To(BeNil())
	})

	t.Run("Invalid geoipEnabled value", func(t *testing.T) {
		g := NewWithT(t)
		annotations := map[string]string{
			LBGeoIPEnabled: "invalid",
		}
		result, err := FillLBWithIngressAnnotations(lbInput, annotations)
		g.Expect(err.Error()).To(BeEquivalentTo(`strconv.ParseBool: parsing "invalid": invalid syntax`))
		g.Expect(result).NotTo(BeNil())
	})

	t.Run("Valid annotations", func(t *testing.T) {
		g := NewWithT(t)
		annotations := map[string]string{
			LBStoreLogsRegionCode: "US01",
			LBGeoIPEnabled:        "true",
			LBMinTLSVersion:       "TLSv1.3",
			LBClusterID:           "123",
		}

		result, err := FillLBWithIngressAnnotations(lbInput, annotations)
		g.Expect(err).To(BeNil())
		g.Expect(result).NotTo(BeNil())

		g.Expect(*result.StoreLogsRegionID).To(Equal(1))
		g.Expect(*result.Geoip).To(BeTrue())
		g.Expect(*result.ClusterID).To(Equal("123"))
		for _, uz := range result.UpstreamZones {
			g.Expect(*uz.TLSPreset).To(Equal("TLSv1.3"))
		}
	})
}
