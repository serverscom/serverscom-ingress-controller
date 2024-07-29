package sync

import (
	"errors"
	"fmt"
	"testing"

	"github.com/serverscom/serverscom-ingress-controller/internal/mocks"

	. "github.com/onsi/gomega"
	client "github.com/serverscom/serverscom-go-client/pkg"
	serverscom "github.com/serverscom/serverscom-go-client/pkg"
	"github.com/serverscom/serverscom-ingress-controller/internal/testdata"
	"go.uber.org/mock/gomock"
	v1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	scCertManagerPrefix = "sc-certmgr-cert-id-"
)

func TestSyncTLS(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tlsManagerHandler := mocks.NewMockTLSManagerInterface(mockCtrl)
	storeHandler := mocks.NewMockStorer(mockCtrl)

	syncManager := New(tlsManagerHandler, nil, storeHandler, nil)

	ingress := &networkv1.Ingress{
		Spec: networkv1.IngressSpec{
			TLS: []networkv1.IngressTLS{
				{
					Hosts:      []string{"example.com"},
					SecretName: "test-secret",
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				TLS_ANNOTATION_PREFIX + "example1.com": "test-secret",
			},
		},
	}
	ingress.Namespace = "default"

	secretData := make(map[string][]byte)
	secretData[v1.TLSCertKey] = []byte(testdata.ValidPEM)
	secretData[v1.TLSPrivateKeyKey] = []byte("valid-key")

	secret := &v1.Secret{
		Data: secretData,
	}

	t.Run("Successfully syncing TLS", func(t *testing.T) {
		g := NewWithT(t)
		storeHandler.EXPECT().GetSecret("default/test-secret").Return(secret, nil).Times(2)

		tlsManagerHandler.EXPECT().HasRegistration(testdata.ValidPEMFingerprint).Return(false).Times(2)

		expectedCert := &client.SSLCertificate{ID: "cert-id"}
		tlsManagerHandler.EXPECT().SyncCertificate(
			gomock.Any(),
			gomock.Any(),
			gomock.Any(),
			gomock.Any(),
			gomock.Any()).
			Return(expectedCert, nil).Times(2)

		result, err := syncManager.SyncTLS(ingress, scCertManagerPrefix)
		g.Expect(err).To(BeNil())
		g.Expect(result).To(HaveKeyWithValue("example.com", "cert-id"))
		g.Expect(result).To(HaveKeyWithValue("example1.com", "cert-id"))
	})

	t.Run("Error fetching secret", func(t *testing.T) {
		g := NewWithT(t)
		storeHandler.EXPECT().GetSecret("default/test-secret").Return(nil, errors.New("error fetching secret"))

		_, err := syncManager.SyncTLS(ingress, scCertManagerPrefix)
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("Secret missing tls.crt", func(t *testing.T) {
		g := NewWithT(t)
		missingCertData := map[string][]byte{
			v1.TLSPrivateKeyKey: []byte("valid-key"),
		}
		storeHandler.EXPECT().GetSecret("default/test-secret").Return(&v1.Secret{Data: missingCertData}, nil)

		_, err := syncManager.SyncTLS(ingress, scCertManagerPrefix)
		g.Expect(err).To(MatchError(fmt.Errorf(`secret "default/test-secret" has no 'tls.crt'`)))
	})

	t.Run("Secret missing tls.key", func(t *testing.T) {
		g := NewWithT(t)
		missingCertData := map[string][]byte{
			v1.TLSCertKey: []byte("valid-crt"),
		}
		storeHandler.EXPECT().GetSecret("default/test-secret").Return(&v1.Secret{Data: missingCertData}, nil)

		_, err := syncManager.SyncTLS(ingress, scCertManagerPrefix)
		g.Expect(err).To(MatchError(fmt.Errorf(`secret "default/test-secret" has no 'tls.key'`)))
	})

	t.Run("Invalid tls.crt data", func(t *testing.T) {
		g := NewWithT(t)
		invalidCertData := map[string][]byte{
			v1.TLSCertKey:       []byte("invalid-cert"),
			v1.TLSPrivateKeyKey: []byte("valid-key"),
		}
		storeHandler.EXPECT().GetSecret("default/test-secret").Return(&v1.Secret{Data: invalidCertData}, nil)

		_, err := syncManager.SyncTLS(ingress, scCertManagerPrefix)
		g.Expect(err).To(MatchError(fmt.Errorf(`secret "default/test-secret" has invalid 'tls.crt': can't find certificate, please verify your tls.crt section`)))
	})

	t.Run("Error syncing certificate", func(t *testing.T) {
		g := NewWithT(t)
		storeHandler.EXPECT().GetSecret("default/test-secret").Return(&v1.Secret{Data: secretData}, nil)
		tlsManagerHandler.EXPECT().HasRegistration(testdata.ValidPEMFingerprint).Return(false)
		tlsManagerHandler.EXPECT().SyncCertificate(
			gomock.Any(),
			gomock.Any(),
			gomock.Any(),
			gomock.Any(),
			gomock.Any()).
			Return(nil, errors.New("error syncing certificate"))

		_, err := syncManager.SyncTLS(ingress, scCertManagerPrefix)
		g.Expect(err).To(MatchError(fmt.Errorf("error syncing certificate")))
	})

	t.Run("Cert manager prefix", func(t *testing.T) {
		g := NewWithT(t)

		ingress.Spec.TLS[0].SecretName = scCertManagerPrefix + "someid"
		ingress.Annotations[TLS_ANNOTATION_PREFIX+"example1.com"] = scCertManagerPrefix + "someid"
		tlsManagerHandler.EXPECT().
			GetByID("someid").
			Return(&serverscom.SSLCertificate{ID: "someid"}, nil).Times(2)

		result, err := syncManager.SyncTLS(ingress, scCertManagerPrefix)
		g.Expect(err).To(BeNil())
		g.Expect(result).To(HaveKeyWithValue("example.com", "someid"))
		g.Expect(result).To(HaveKeyWithValue("example1.com", "someid"))
	})
}
