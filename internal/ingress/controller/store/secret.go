package store

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

// SecretLister makes a Store that lists Secrets.
type SecretLister struct {
	cache.Store
}

// ByKey returns the Secret matching key in the local Secret Store.
func (l *SecretLister) ByKey(key string) (*corev1.Secret, error) {
	s, exists, err := l.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, NotExistsError(key)
	}
	return s.(*corev1.Secret), nil
}
