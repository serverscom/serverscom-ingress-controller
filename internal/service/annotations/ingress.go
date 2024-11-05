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
	LBClusterID             = "servers.com/cluster-id"
)

// FillLBWithIngressAnnotations prepares the LB input based on annotations.
func FillLBWithIngressAnnotations(lbInput *serverscom.L7LoadBalancerCreateInput, annotations map[string]string) (*serverscom.L7LoadBalancerCreateInput, error) {
	// LBStoreLogsRegionCode annotation
	if value, ok := annotations[LBStoreLogsRegionCode]; ok {
		if regionID, found := GetStorageRegionIDByCode(value); found {
			lbInput.StoreLogsRegionID = &regionID
			storeLogs := true
			lbInput.StoreLogs = &storeLogs
		}
	}

	// LBGeoIPEnabled annotation
	if value, ok := annotations[LBGeoIPEnabled]; ok {
		val, err := strconv.ParseBool(value)
		if err != nil {
			return lbInput, err
		}
		lbInput.Geoip = &val
	}

	// LBMinTLSVersion annotation
	if value, ok := annotations[LBMinTLSVersion]; ok {
		for i := range lbInput.UpstreamZones {
			lbInput.UpstreamZones[i].TLSPreset = &value
		}
	}

	// LBClusterID annotation
	if id, ok := annotations[LBClusterID]; ok && id != "" {
		lbInput.ClusterID = &id
	}

	return lbInput, nil
}
