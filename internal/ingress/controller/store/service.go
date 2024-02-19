package store

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

// ServiceLister makes a Store that lists Services.
type ServiceLister struct {
	cache.Store
}

// ByKey returns the Service matching key in the local Service Store.
func (l *ServiceLister) ByKey(key string) (*corev1.Service, error) {
	s, exists, err := l.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, NotExistsError(key)
	}
	return s.(*corev1.Service), nil
}
