package tls

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/serverscom/serverscom-ingress-controller/internal/ingress/controller/store"

	serverscom "github.com/serverscom/serverscom-go-client/pkg"
)

//go:generate mockgen --destination ../../mocks/tls_manager.go --package=mocks --source manager.go

// ManagerInterface describes an interface to manage SSL certs
type TLSManagerInterface interface {
	HasRegistration(fingerprint string) bool
	SyncCertificate(fingerprint, name string, cert, key, chain []byte) (*serverscom.SSLCertificate, error)
	Get(fingerprint string) (*serverscom.SSLCertificate, error)
}

// Manager represents a TLS manager
type Manager struct {
	resources map[string]*SslCertificate

	lock   sync.Mutex
	client *serverscom.Client
	store  store.Storer
}

// SslCertificate represents an ssl cert object for manager
type SslCertificate struct {
	state       *serverscom.SSLCertificate
	lastRefresh time.Time
}

// NewManager creates a new TLS manager
func NewManager(client *serverscom.Client, store store.Storer) *Manager {
	return &Manager{
		resources: make(map[string]*SslCertificate),
		client:    client,
		store:     store,
	}
}

// HasRegistration checks if TLS manager has an ssl with specified fingerprint
func (m *Manager) HasRegistration(fingerprint string) bool {
	m.lock.Lock()
	defer m.lock.Unlock()

	_, ok := m.resources[fingerprint]
	return ok
}

// SyncCertificate creates an ssl in portal and add it to manager or update it in manager it it already exists in portal
func (m *Manager) SyncCertificate(fingerprint, name string, cert, key, chain []byte) (*serverscom.SSLCertificate, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	var sslCertificate SslCertificate

	list, err := m.client.SSLCertificates.
		Collection().
		SetParam("search_pattern", fingerprint).
		SetParam("type", "custom").
		Collect(context.Background())

	if err != nil {
		return nil, fmt.Errorf("can't get ssl certificates list: %s", err.Error())
	}

	if len(list) != 0 {
		for _, certificate := range list {
			if fingerprint == certificate.Sha1Fingerprint {
				sslCertificate.state = &certificate
				sslCertificate.lastRefresh = time.Now()

				break
			}
		}
	}

	if sslCertificate.state != nil {
		m.resources[fingerprint] = &sslCertificate

		return sslCertificate.state, nil
	}

	newInput := serverscom.SSLCertificateCreateCustomInput{}
	newInput.Name = name
	newInput.PublicKey = string(cert)
	newInput.PrivateKey = string(key)

	if chain != nil {
		newInput.ChainKey = string(chain)
	}

	state, err := m.client.SSLCertificates.CreateCustom(context.Background(), newInput)

	if err != nil {
		return nil, err
	}

	sslCertificate.state = (*serverscom.SSLCertificate)(state)
	sslCertificate.lastRefresh = time.Now()

	m.resources[fingerprint] = &sslCertificate

	return (*serverscom.SSLCertificate)(state), nil
}

// Get gets an ssl from manager
func (m *Manager) Get(fingerprint string) (*serverscom.SSLCertificate, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	sslCertificate, ok := m.resources[fingerprint]

	if !ok {
		return nil, fmt.Errorf("can't find registered resource with name: %s", fingerprint)
	}

	if sslCertificate.state == nil {
		return nil, fmt.Errorf("can' find state for name: %s", fingerprint)
	}

	return sslCertificate.state, nil
}
