package store

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

const (
	MasterNodeAnnotationKey = "node-role.kubernetes.io/master"
)

// SecretLister makes a Store that lists Secrets.
type NodeLister struct {
	cache.Store
}

// NodesIpList returns nodes ips
func (l *NodeLister) NodesIpList() []string {
	var ips []string
	for _, obj := range l.List() {
		node := obj.(*corev1.Node)
		// Skip master nodes
		if _, ok := node.Labels[MasterNodeAnnotationKey]; ok {
			continue
		}
		for _, address := range node.Status.Addresses {
			if address.Type == corev1.NodeInternalIP {
				ips = append(ips, address.Address)
				break
			}
		}
	}

	return ips
}
