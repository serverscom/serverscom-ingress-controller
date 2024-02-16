package tls

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/serverscom/serverscom-ingress-controller/internal/ingress/controller/store"

	client "github.com/serverscom/serverscom-go-client/pkg"
)

//go:generate mockgen --destination ../../mocks/tls_manager.go --package=mocks --source manager.go

// ManagerInterface describes an interface to manage SSL certs
type TLSManagerInterface interface {
	HasRegistration(fingerprint string) bool
	SyncCertificate(fingerprint, name string, cert, key, chain []byte) (*client.SSLCertificate, error)
	Get(fingerprint string) (*client.SSLCertificate, error)
}

// Manager represents a TLS manager
type Manager struct {
	resources map[string]*SslCertificate

	lock       sync.Mutex
	sslService client.SSLCertificatesService
	store      store.Storer
}

// SslCertificate represents an ssl cert object for manager
type SslCertificate struct {
	state       *client.SSLCertificate
	lastRefresh time.Time
}

// NewManager creates a new TLS manager
func NewManager(sslService client.SSLCertificatesService, store store.Storer) *Manager {
	return &Manager{
		resources:  make(map[string]*SslCertificate),
		sslService: sslService,
		store:      store,
	}
}

// HasRegistration checks if TLS manager has an ssl with specified fingerprint
func (col *Manager) HasRegistration(fingerprint string) bool {
	col.lock.Lock()
	defer col.lock.Unlock()

	_, ok := col.resources[fingerprint]
	return ok
}

// SyncCertificate creates an ssl in portal and add it to manager or update it in manager it it already exists in portal
func (col *Manager) SyncCertificate(fingerprint, name string, cert, key, chain []byte) (*client.SSLCertificate, error) {
	col.lock.Lock()
	defer col.lock.Unlock()

	var sslCertificate SslCertificate

	list, err := col.sslService.
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
		col.resources[fingerprint] = &sslCertificate

		return sslCertificate.state, nil
	}

	newInput := client.SSLCertificateCreateCustomInput{}
	newInput.Name = name
	newInput.PublicKey = string(cert)
	newInput.PrivateKey = string(key)

	if chain != nil {
		newInput.ChainKey = string(chain)
	}

	state, err := col.sslService.CreateCustom(context.Background(), newInput)

	if err != nil {
		return nil, err
	}

	sslCertificate.state = (*client.SSLCertificate)(state)
	sslCertificate.lastRefresh = time.Now()

	col.resources[fingerprint] = &sslCertificate

	return (*client.SSLCertificate)(state), nil
}

// Get gets an ssl from manager
func (col *Manager) Get(fingerprint string) (*client.SSLCertificate, error) {
	col.lock.Lock()
	defer col.lock.Unlock()

	sslCertificate, ok := col.resources[fingerprint]

	if !ok {
		return nil, fmt.Errorf("can't find registered resource with name: %s", fingerprint)
	}

	if sslCertificate.state == nil {
		return nil, fmt.Errorf("can' find state for name: %s", fingerprint)
	}

	return sslCertificate.state, nil
}
