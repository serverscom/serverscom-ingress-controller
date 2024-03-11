package sync

import (
	"fmt"

	"github.com/serverscom/serverscom-ingress-controller/internal/service/tls"
	v1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
)

// SyncTLS syncs ingress tls certs stored in secrets to portal.
// Returns map of hosts to portal cert id
func (s *SyncManager) SyncTLS(ingress *networkv1.Ingress) (map[string]string, error) {
	var sslCerts = make(map[string]string)

	for _, intls := range ingress.Spec.TLS {
		sKey := ingress.Namespace + "/" + intls.SecretName
		secret, err := s.store.GetSecret(sKey)
		if err != nil {
			return nil, fmt.Errorf("fetching secret with key %s from store failed: %v", sKey, err)
		}
		cert, ok := secret.Data[v1.TLSCertKey]
		if !ok {
			return nil, fmt.Errorf("secret %v has no 'tls.crt'", sKey)
		}

		key, ok := secret.Data[v1.TLSPrivateKeyKey]
		if !ok {
			return nil, fmt.Errorf("secret %v has no 'tls.key'", sKey)
		}

		if err := tls.ValidateCertificate(cert); err != nil {
			return nil, fmt.Errorf("secret %v has invalid 'tls.crt': %s", sKey, err.Error())
		}

		primary, chain := tls.SplitCerts(cert)

		fingerprint := tls.GetPemFingerprint(primary)

		if fingerprint == "" {
			return nil, fmt.Errorf("can't calculate 'tls.crt' fingerprint for %s", string(cert))
		}

		if s.tls.HasRegistration(fingerprint) {
			certificate, err := s.tls.Get(fingerprint)

			if err != nil {
				return nil, err
			}

			for _, host := range intls.Hosts {
				sslCerts[host] = certificate.ID
			}

			continue
		}

		certificate, err := s.tls.SyncCertificate(
			fingerprint,
			intls.SecretName,
			primary,
			tls.StripSpaces(key),
			chain,
		)

		if err != nil {
			return nil, err
		}

		for _, host := range intls.Hosts {
			sslCerts[host] = certificate.ID
		}

	}
	return sslCerts, nil
}
