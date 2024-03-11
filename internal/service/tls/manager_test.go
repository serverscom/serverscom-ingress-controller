package tls

import (
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	serverscom "github.com/serverscom/serverscom-go-client/pkg"
	"github.com/serverscom/serverscom-ingress-controller/internal/mocks"
	"go.uber.org/mock/gomock"
)

func TestHasRegistration(t *testing.T) {
	g := NewWithT(t)

	manager := NewManager(nil, nil)

	fingerprint := "fingerprint"
	manager.resources[fingerprint] = &SslCertificate{
		state:       &serverscom.SSLCertificate{},
		lastRefresh: time.Now(),
	}

	g.Expect(manager.HasRegistration(fingerprint)).To(BeTrue())
	g.Expect(manager.HasRegistration("non-exist")).To(BeFalse())
}

func TestSyncCertificate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	sslHandler := mocks.NewMockSSLCertificatesService(mockCtrl)
	collectionHandler := mocks.NewMockCollection[serverscom.SSLCertificate](mockCtrl)

	sslHandler.EXPECT().
		Collection().
		Return(collectionHandler).
		AnyTimes()

	collectionHandler.EXPECT().
		SetParam(gomock.Any(), gomock.Any()).
		Return(collectionHandler).
		AnyTimes()

	existFingerprint := "exist-fingerprint"
	newFingerprint := "new-fingerprint"
	name := "crt-name"
	cert := []byte("cert")
	key := []byte("key")
	chain := []byte("chain")
	existingCert := serverscom.SSLCertificate{Sha1Fingerprint: existFingerprint}
	newCert := serverscom.SSLCertificateCustom{Sha1Fingerprint: newFingerprint}
	startTime := time.Now()

	client := serverscom.NewClientWithEndpoint("", "")
	client.SSLCertificates = sslHandler
	manager := NewManager(client, nil)

	t.Run("Can't get ssl certs list", func(t *testing.T) {
		g := NewWithT(t)

		collectionHandler.EXPECT().
			Collect(gomock.Any()).
			Return(nil, errors.New("error"))

		cert, err := manager.SyncCertificate("", name, cert, key, chain)
		g.Expect(cert).To(BeNil())
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("Certificate found in list", func(t *testing.T) {
		g := NewWithT(t)

		collectionHandler.EXPECT().
			Collect(gomock.Any()).
			Return([]serverscom.SSLCertificate{existingCert}, nil)

		result, err := manager.SyncCertificate(existFingerprint, name, cert, key, chain)

		g.Expect(*result).To(BeEquivalentTo(existingCert))
		g.Expect(err).To(BeNil())
		g.Expect(manager.resources[existFingerprint]).NotTo(BeNil())
		g.Expect(manager.resources[existFingerprint].lastRefresh).To(BeTemporally(">", startTime))

	})

	t.Run("Certificate not found in list and creation successful", func(t *testing.T) {
		g := NewWithT(t)

		collectionHandler.EXPECT().
			Collect(gomock.Any()).
			Return([]serverscom.SSLCertificate{}, nil)

		sslHandler.EXPECT().
			CreateCustom(gomock.Any(), serverscom.SSLCertificateCreateCustomInput{
				Name:       name,
				PublicKey:  string(cert),
				PrivateKey: string(key),
				ChainKey:   string(chain),
			}).
			Return(&newCert, nil)

		result, err := manager.SyncCertificate(newFingerprint, name, cert, key, chain)

		g.Expect(*result).To(BeEquivalentTo(newCert))
		g.Expect(err).To(BeNil())
		g.Expect(manager.resources[newFingerprint]).NotTo(BeNil())
		g.Expect(manager.resources[newFingerprint].lastRefresh).To(BeTemporally(">", startTime))

	})

	t.Run("Certificate not found in list and creation fails", func(t *testing.T) {
		g := NewWithT(t)
		collectionHandler.EXPECT().
			Collect(gomock.Any()).
			Return([]serverscom.SSLCertificate{}, nil)

		sslHandler.EXPECT().
			CreateCustom(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("error"))

		result, err := manager.SyncCertificate(newFingerprint, name, cert, key, chain)

		g.Expect(result).To(BeNil())
		g.Expect(err).To(HaveOccurred())
	})
}

func TestGet(t *testing.T) {
	manager := NewManager(nil, nil)

	fingerprint := "fingerprint"
	sslCertificate := &SslCertificate{
		state: &serverscom.SSLCertificate{ID: "id"},
	}
	manager.resources[fingerprint] = sslCertificate

	t.Run("Certificate found in collection", func(t *testing.T) {
		g := NewWithT(t)
		result, err := manager.Get(fingerprint)
		g.Expect(err).To(BeNil())
		g.Expect(result).To(Equal(sslCertificate.state))
	})

	t.Run("Certificate not found in collection", func(t *testing.T) {
		g := NewWithT(t)
		_, err := manager.Get("non-exist")
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("Certificate has missing state", func(t *testing.T) {
		g := NewWithT(t)
		manager.resources[fingerprint] = &SslCertificate{}

		_, err := manager.Get(fingerprint)
		g.Expect(err).To(HaveOccurred())
	})
}
