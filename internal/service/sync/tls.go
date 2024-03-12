package sync

import (
	"fmt"
	"strings"

	tlsmanager "github.com/serverscom/serverscom-ingress-controller/internal/service/tls"
	v1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
)

// SyncTLS syncs ingress tls certs stored in secrets to portal.
// Returns map of hosts to portal cert id
func (s *SyncManager) SyncTLS(ingress *networkv1.Ingress, certManagerPrefix string) (map[string]string, error) {
	var sslCerts = make(map[string]string)
	for _, tls := range ingress.Spec.TLS {
		if strings.HasPrefix(tls.SecretName, certManagerPrefix) {
			id := strings.TrimPrefix(tls.SecretName, certManagerPrefix)
			certificate, err := s.tls.GetByID(id)
			if err != nil {
				return nil, fmt.Errorf("fetching cert with id '%s' from portal failed: %v", id, err)
			}
			for _, host := range tls.Hosts {
				sslCerts[host] = certificate.ID
			}
			continue
		}
		sKey := ingress.Namespace + "/" + tls.SecretName
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

		if err := tlsmanager.ValidateCertificate(cert); err != nil {
			return nil, fmt.Errorf("secret %v has invalid 'tls.crt': %s", sKey, err.Error())
		}

		primary, chain := tlsmanager.SplitCerts(cert)

		fingerprint := tlsmanager.GetPemFingerprint(primary)

		if fingerprint == "" {
			return nil, fmt.Errorf("can't calculate 'tls.crt' fingerprint for %s", string(cert))
		}

		if s.tls.HasRegistration(fingerprint) {
			certificate, err := s.tls.Get(fingerprint)

			if err != nil {
				return nil, err
			}

			for _, host := range tls.Hosts {
				sslCerts[host] = certificate.ID
			}

			continue
		}

		certificate, err := s.tls.SyncCertificate(
			fingerprint,
			tls.SecretName,
			primary,
			tlsmanager.StripSpaces(key),
			chain,
		)

		if err != nil {
			return nil, err
		}

		for _, host := range tls.Hosts {
			sslCerts[host] = certificate.ID
		}

	}
	return sslCerts, nil
}
