package annotations

import (
	"testing"

	. "github.com/onsi/gomega"
	serverscom "github.com/serverscom/serverscom-go-client/pkg"
)

func TestFillLBVHostZoneWithServiceAnnotations(t *testing.T) {
	g := NewWithT(t)

	vZInput := &serverscom.L7VHostZoneInput{}

	annotations := map[string]string{}
	result := FillLBVHostZoneWithServiceAnnotations(vZInput, annotations)
	g.Expect(result).NotTo(BeNil())
	g.Expect(result.HTTP2).To(BeFalse())

	annotations = map[string]string{
		AppProtocol: "http2",
		LBIPHeader:  "real_ip",
		LBIPSubnets: "192.168.1.0/24, 10.0.0.0/8",
	}

	result = FillLBVHostZoneWithServiceAnnotations(vZInput, annotations)
	g.Expect(result).NotTo(BeNil())
	g.Expect(result.HTTP2).To(BeTrue())
	g.Expect(result.RealIPHeader.Name).To(BeEquivalentTo(string(serverscom.RealIP)))
	g.Expect(result.RealIPHeader.Networks).To(BeEquivalentTo([]string{"192.168.1.0/24", "10.0.0.0/8"}))
}

func TestFillLBUpstreamZoneWithServiceAnnotations(t *testing.T) {
	g := NewWithT(t)

	uZInput := &serverscom.L7UpstreamZoneInput{}

	invalidAnnotations := map[string]string{
		AppHealthcheckCheckToFail:  "",
		AppHealthcheckChecksToPass: "",
		AppHealthcheckInterval:     "",
		AppHealthcheckJitter:       "",
	}
	result := FillLBUpstreamZoneWithServiceAnnotations(uZInput, invalidAnnotations)
	g.Expect(result.HCFails).To(BeNil())
	g.Expect(result.HCPasses).To(BeNil())
	g.Expect(result.HCInterval).To(BeNil())
	g.Expect(result.HCJitter).To(BeNil())

	annotations := map[string]string{
		LBBalancingAlgorithm:         "round_robin",
		AppHealthcheckPath:           "/health",
		AppHealthcheckDomain:         "example.com",
		AppHealthcheckRequestsMethod: "GET",
		AppHealthcheckCheckToFail:    "3",
		AppHealthcheckChecksToPass:   "2",
		AppHealthcheckInterval:       "10",
		AppHealthcheckJitter:         "5",
	}

	result = FillLBUpstreamZoneWithServiceAnnotations(uZInput, annotations)
	g.Expect(result).NotTo(BeNil())
	g.Expect(*result.Method).To(Equal("round_robin"))
	g.Expect(*result.HCPath).To(Equal("/health"))
	g.Expect(*result.HCDomain).To(Equal("example.com"))
	g.Expect(*result.HCMethod).To(Equal("GET"))
	g.Expect(*result.HCFails).To(Equal(3))
	g.Expect(*result.HCPasses).To(Equal(2))
	g.Expect(*result.HCInterval).To(Equal(10))
	g.Expect(*result.HCJitter).To(Equal(5))
}
