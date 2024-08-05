package sync

import (
	"fmt"
	"strings"

	tlsmanager "github.com/serverscom/serverscom-ingress-controller/internal/service/tls"
	v1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
)

const (
	TLS_ANNOTATION_PREFIX = "servers.com/certificate-"
)

// SyncTLS syncs ingress tls certs stored in secrets to portal.
// If secret name starts with certManagerPrefix-<certID> we looking for cert from API
// Due to secret name don't support upperCase for such cases we additionally checks annotations
// with TLS_ANNOTATION_PREFIX which overrides ingress tls certs for matching hosts.
// Returns map of hosts to portal cert id
func (s *SyncManager) SyncTLS(ingress *networkv1.Ingress, certManagerPrefix string) (map[string]string, error) {
	var sslCerts = make(map[string]string)

	hostsSecrets := mergeTLSWithAnnotations(ingress)
	for host, secretName := range hostsSecrets {
		if strings.HasPrefix(secretName, certManagerPrefix) {
			id := strings.TrimPrefix(secretName, certManagerPrefix)
			certificate, err := s.tlsMgr.GetByID(id)
			if err != nil {
				return nil, fmt.Errorf("fetching cert with id %q from API failed: %v", id, err)
			}
			sslCerts[host] = certificate.ID
			continue
		}
		sKey := ingress.Namespace + "/" + secretName
		secret, err := s.store.GetSecret(sKey)
		if err != nil {
			return nil, fmt.Errorf("fetching secret with key %q from store failed: %v", sKey, err)
		}
		cert, ok := secret.Data[v1.TLSCertKey]
		if !ok {
			return nil, fmt.Errorf("secret %q has no 'tls.crt'", sKey)
		}

		key, ok := secret.Data[v1.TLSPrivateKeyKey]
		if !ok {
			return nil, fmt.Errorf("secret %q has no 'tls.key'", sKey)
		}

		if err := tlsmanager.ValidateCertificate(cert); err != nil {
			return nil, fmt.Errorf("secret %q has invalid 'tls.crt': %v", sKey, err)
		}

		primary, chain := tlsmanager.SplitCerts(cert)

		fingerprint := tlsmanager.GetPemFingerprint(primary)
		if fingerprint == "" {
			return nil, fmt.Errorf("can't calculate 'tls.crt' fingerprint for %s", string(cert))
		}

		if s.tlsMgr.HasRegistration(fingerprint) {
			certificate, err := s.tlsMgr.Get(fingerprint)
			if err != nil {
				return nil, err
			}
			sslCerts[host] = certificate.ID
			continue
		}

		certificate, err := s.tlsMgr.SyncCertificate(
			fingerprint,
			secretName,
			primary,
			tlsmanager.StripSpaces(key),
			chain,
		)

		if err != nil {
			return nil, err
		}

		sslCerts[host] = certificate.ID
	}
	return sslCerts, nil
}

// mergeTLSWithAnnotations merge info about host and associated secret from ingress.Spec.TLS and ingress.Annotations
// returns map[host]secret
func mergeTLSWithAnnotations(ingress *networkv1.Ingress) map[string]string {
	res := make(map[string]string)

	for _, tls := range ingress.Spec.TLS {
		sName := tls.SecretName
		for _, host := range tls.Hosts {
			res[host] = sName
		}
	}

	// annotations overrides settings from tls
	for k, v := range ingress.Annotations {
		if strings.HasPrefix(k, TLS_ANNOTATION_PREFIX) {
			if host, ok := strings.CutPrefix(k, TLS_ANNOTATION_PREFIX); ok {
				res[host] = v
			}
		}
	}

	return res
}
