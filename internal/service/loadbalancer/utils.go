package loadbalancer

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/networking/v1"
)

// GetLoadBalancerName compose a load balancer name from ingress object
func GetLoadBalancerName(ing *v1.Ingress) string {
	ret := "a" + string(ing.UID)
	ret = strings.Replace(ret, "-", "", -1)
	if len(ret) > 32 {
		ret = ret[:32]
	}
	return fmt.Sprintf("ingress-%s", ret)
}
