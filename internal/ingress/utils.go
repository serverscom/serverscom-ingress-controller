package ingress

import v1 "k8s.io/api/networking/v1"

const (
	IngressClassKey = "kubernetes.io/ingress.class"
)

// IsScIngress checks if Ingress belongs to a specified class
func IsScIngress(i *v1.Ingress, class string) bool {
	if i.Spec.IngressClassName != nil {
		return *i.Spec.IngressClassName == class
	}

	if c, ok := i.Annotations[IngressClassKey]; ok {
		return c == class
	}

	return false
}
