package store

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
)

// PathInfo represents info about a path in the ingress controller
type PathInfo struct {
	Path     string
	Service  *corev1.Service
	NodePort int
	NodeIps  []string
}

// HostInfo represents info about a host in the ingress controller
type HostInfo struct {
	Host  string
	Paths []PathInfo
}

// getIngressHostsInfo get hosts info from ingress
func getIngressHostsInfo(ingress *networkv1.Ingress, store Storer) (map[string]HostInfo, error) {
	hostsInfo := make(map[string]HostInfo)
	nodeIps := store.GetNodesIpList()

	for _, rule := range ingress.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}

		hInfo := hostsInfo[rule.Host]
		hInfo.Host = rule.Host

		for _, path := range rule.HTTP.Paths {
			svc, err := store.GetService(ingress.Namespace + "/" + path.Backend.Service.Name)
			if err != nil {
				return nil, fmt.Errorf("error getting service: %v", err)
			}

			var nodePort int32
			found := false
			for _, port := range svc.Spec.Ports {
				if port.Port == path.Backend.Service.Port.Number {
					if port.NodePort == 0 {
						return nil, fmt.Errorf("service %s has no NodePort (only NodePort/LoadBalancer supported)", svc.Name)
					}
					nodePort = port.NodePort
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("service %s: port %d not found", svc.Name, path.Backend.Service.Port.Number)
			}

			hInfo.Paths = append(hInfo.Paths, PathInfo{
				Path:     path.Path,
				Service:  svc,
				NodePort: int(nodePort),
				NodeIps:  nodeIps,
			})
		}

		hostsInfo[rule.Host] = hInfo
	}

	return hostsInfo, nil
}
