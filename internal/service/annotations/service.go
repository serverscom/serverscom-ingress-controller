package annotations

import (
	"strconv"
	"strings"

	serverscom "github.com/serverscom/serverscom-go-client/pkg"
)

const (
	LBBalancingAlgorithm         = "servers.com/load-balancer-balancing-algorithm"
	AppProtocol                  = "servers.com/app-protocol"
	AppHealthcheckPath           = "servers.com/app-healthcheck-path"
	AppHealthcheckDomain         = "servers.com/app-healthcheck-domain"
	AppHealthcheckRequestsMethod = "servers.com/app-healthcheck-requests-method"
	AppHealthcheckCheckToFail    = "servers.com/app-healthcheck-checks-to-fail"
	AppHealthcheckChecksToPass   = "servers.com/app-healthcheck-checks-to-pass"
	AppHealthcheckInterval       = "servers.com/app-healthcheck-interval"
	AppHealthcheckJitter         = "servers.com/app-healthcheck-jitter"
	LBIPHeader                   = "servers.com/load-balancer-ip-header"
	LBIPSubnets                  = "servers.com/load-balancer-ip-subnets"
)

// FillLBVHostZoneWithServiceAnnotations prepares the LB vhost zone input based on annotations.
func FillLBVHostZoneWithServiceAnnotations(vZInput *serverscom.L7VHostZoneInput, annotations map[string]string) *serverscom.L7VHostZoneInput {
	// AppProtocol annotation
	if value, ok := annotations[AppProtocol]; ok {
		if strings.EqualFold(value, "http2") {
			vZInput.HTTP2 = true
		}
	}

	// LBIPHeader & LBIPSubnets annotations
	if value, ok := annotations[LBIPHeader]; ok {
		vZInput.RealIPHeader = new(serverscom.RealIPHeader)
		vZInput.RealIPHeader.Name = ParseRealIPHeaderName(value)
		if subnets, ok := annotations[LBIPSubnets]; ok {
			s := strings.Split(strings.ReplaceAll(subnets, " ", ""), ",")
			vZInput.RealIPHeader.Networks = s
		}
	}

	return vZInput
}

// FillLBUpstreamZoneWithServiceAnnotations prepares the LB upstream zone input based on annotations.
func FillLBUpstreamZoneWithServiceAnnotations(uZInput *serverscom.L7UpstreamZoneInput, annotations map[string]string) *serverscom.L7UpstreamZoneInput {
	// LBBalancingAlgorithm annotation
	if value, ok := annotations[LBBalancingAlgorithm]; ok {
		uZInput.Method = &value
	}

	// AppHealthcheckPath annotation
	if value, ok := annotations[AppHealthcheckPath]; ok {
		uZInput.HCPath = &value
	}

	// AppHealthcheckDomain annotation
	if value, ok := annotations[AppHealthcheckDomain]; ok {
		uZInput.HCDomain = &value
	}

	// AppHealthcheckRequestsMethod annotation
	if value, ok := annotations[AppHealthcheckRequestsMethod]; ok {
		uZInput.HCMethod = &value
	}

	// AppHealthcheckCheckToFail annotation
	if value, ok := annotations[AppHealthcheckCheckToFail]; ok {
		if val, err := strconv.Atoi(value); err == nil {
			uZInput.HCFails = &val
		}
	}

	// AppHealthcheckChecksToPass annotation
	if value, ok := annotations[AppHealthcheckChecksToPass]; ok {
		if val, err := strconv.Atoi(value); err == nil {
			uZInput.HCPasses = &val
		}
	}

	// AppHealthcheckInterval annotation
	if value, ok := annotations[AppHealthcheckInterval]; ok {
		if val, err := strconv.Atoi(value); err == nil {
			uZInput.HCInterval = &val
		}
	}

	// AppHealthcheckJitter annotation
	if value, ok := annotations[AppHealthcheckJitter]; ok {
		if val, err := strconv.Atoi(value); err == nil {
			uZInput.HCJitter = &val
		}
	}

	return uZInput
}

// ParseRealIPHeaderName parses the Real IP Header Name from annotation
func ParseRealIPHeaderName(input string) serverscom.RealIPHeaderName {
	switch input {
	case string(serverscom.RealIP):
		return serverscom.RealIP
	case string(serverscom.ForwardedFor):
		return serverscom.ForwardedFor
	default:
		return ""
	}
}
