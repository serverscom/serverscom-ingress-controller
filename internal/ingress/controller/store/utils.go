package store

import (
	"fmt"

	networkv1 "k8s.io/api/networking/v1"
)

// ServiceInfo represents helper struct for ingress service
type ServiceInfo struct {
	Hosts       []string
	NodeIps     []string
	NodePort    int
	Annotations map[string]string
}

// GetIngressServiceInfo get services info from ingress
func getIngressServiceInfo(ingress *networkv1.Ingress, store Storer) (map[string]ServiceInfo, error) {
	servicesInfo := make(map[string]ServiceInfo)
	nodeIps := store.GetNodesIpList()

	for _, rule := range ingress.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}

		for _, path := range rule.HTTP.Paths {
			service, err := store.GetService(ingress.Namespace + "/" + path.Backend.Service.Name)
			if err != nil {
				return nil, fmt.Errorf("error getting service: %v", err)
			}

			for _, port := range service.Spec.Ports {
				if port.Port == path.Backend.Service.Port.Number {
					if port.NodePort != 0 {
						serviceName := path.Backend.Service.Name
						if _, ok := servicesInfo[serviceName]; !ok {
							servicesInfo[serviceName] = ServiceInfo{
								Hosts:       []string{rule.Host},
								NodePort:    int(port.NodePort),
								NodeIps:     nodeIps,
								Annotations: service.Annotations,
							}
						} else {
							sTmp := servicesInfo[serviceName]
							sTmp.Hosts = append(sTmp.Hosts, rule.Host)
							servicesInfo[serviceName] = sTmp
						}
					}
					break
				}
			}
		}
	}

	return servicesInfo, nil
}
