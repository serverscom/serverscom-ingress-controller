package annotations

import (
	"strconv"

	serverscom "github.com/serverscom/serverscom-go-client/pkg"
)

const (
	LBStoreLogsRegionCode   = "servers.com/load-balancer-store-logs-region-code"
	LBGeoIPEnabled          = "servers.com/load-balancer-geo-ip-enabled"
	LBRealIPTrustedNetworks = "servers.com/load-balancer-real-ip-trusted-networks" // TODO not implemented yet
	LBMinTLSVersion         = "servers.com/load-balancer-min-tls-version"
)

// FillLBWithIngressAnnotations prepares the LB input based on annotations.
func FillLBWithIngressAnnotations(lbInput *serverscom.L7LoadBalancerCreateInput, annotations map[string]string) *serverscom.L7LoadBalancerCreateInput {
	// LBStoreLogsRegionCode annotation
	if value, ok := annotations[LBStoreLogsRegionCode]; ok {
		val, err := strconv.Atoi(value)
		if err == nil {
			lbInput.StoreLogsRegionID = &val
		}
	}

	// LBGeoIPEnabled annotation
	if value, ok := annotations[LBGeoIPEnabled]; ok {
		val, err := strconv.ParseBool(value)
		if err == nil {
			lbInput.Geoip = &val
		}
	}

	// LBMinTLSVersion annotation
	if value, ok := annotations[LBMinTLSVersion]; ok {
		for i := range lbInput.UpstreamZones {
			lbInput.UpstreamZones[i].TLSPreset = &value
		}
	}

	return lbInput
}
