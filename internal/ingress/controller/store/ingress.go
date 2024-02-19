package store

import (
	networkv1 "k8s.io/api/networking/v1"
	"k8s.io/client-go/tools/cache"
)

// IngressLister makes a Store that lists Ingress.
type IngressLister struct {
	cache.Store
}

// ByKey returns the Ingress matching key in the local Ingress Store.
func (l IngressLister) ByKey(key string) (*networkv1.Ingress, error) {
	i, exists, err := l.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, NotExistsError(key)
	}
	return i.(*networkv1.Ingress), nil
}

// ListIngress returns a list of ingresses.
func (l IngressLister) ListIngress() []*networkv1.Ingress {
	var ings []*networkv1.Ingress
	items := l.List()
	for _, ing := range items {
		ings = append(ings, ing.(*networkv1.Ingress))
	}
	return ings
}
