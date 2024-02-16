package tls

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/serverscom/serverscom-ingress-controller/internal/testdata"
)

func TestFindCertificate(t *testing.T) {
	t.Run("valid PEM", func(t *testing.T) {
		g := NewWithT(t)
		cert := FindCertificate([]byte(testdata.ValidPEM))
		g.Expect(cert).NotTo(BeNil())
	})

	t.Run("invalid PEM", func(t *testing.T) {
		g := NewWithT(t)
		cert := FindCertificate([]byte(testdata.InvalidPEM))
		g.Expect(cert).To(BeNil())
	})

	t.Run("empty PEM", func(t *testing.T) {
		g := NewWithT(t)
		cert := FindCertificate([]byte{})
		g.Expect(cert).To(BeNil())
	})
}

func TestStripSpaces(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name:     "Trim spaces",
			input:    []byte("  one  \ntwo  \n  three"),
			expected: []byte("one\ntwo\nthree"),
		},
		{
			name:     "Remove empty lines",
			input:    []byte("one\n\n\ntwo\n\nthree"),
			expected: []byte("one\ntwo\nthree"),
		},
		{
			name:     "No spaces",
			input:    []byte("one\ntwo\nthree"),
			expected: []byte("one\ntwo\nthree"),
		},
		{
			name:     "Only spaces",
			input:    []byte("   \n  \n "),
			expected: []byte(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			result := StripSpaces(tt.input)
			g.Expect(result).To(Equal(tt.expected))
		})
	}
}

func TestGetPemFingerprint(t *testing.T) {
	g := NewWithT(t)

	result := GetPemFingerprint([]byte(testdata.ValidPEM))
	g.Expect(result).To(Equal(testdata.ValidPEMFingerprint))

	g.Expect(GetPemFingerprint([]byte{})).To(BeEmpty())
}

func TestSplitCerts(t *testing.T) {

	fullPEM := testdata.ValidPEM + "\n" + testdata.ChainPEM

	tests := []struct {
		name          string
		input         []byte
		expectedCert  []byte
		expectedChain []byte
	}{
		{
			name:          "Only primary certificate",
			input:         []byte(testdata.ValidPEM),
			expectedCert:  []byte(testdata.ValidPEM),
			expectedChain: nil,
		},
		{
			name:          "Certificate with chain",
			input:         []byte(fullPEM),
			expectedCert:  []byte(testdata.ValidPEM),
			expectedChain: []byte(testdata.ChainPEM),
		},
		{
			name:          "Empty input",
			input:         []byte(""),
			expectedCert:  nil,
			expectedChain: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			cert, chain := SplitCerts(tt.input)
			g.Expect(cert).To(Equal(tt.expectedCert))
			if tt.expectedChain == nil {
				g.Expect(chain).To(BeNil())
			} else {
				g.Expect(chain).To(Equal(tt.expectedChain))
			}
		})
	}
}

func TestValidateCertificate(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		wantErr bool
	}{
		{
			name:    "Valid certificate",
			input:   []byte(testdata.ValidPEM),
			wantErr: false,
		},
		{
			name:    "Invalid certificate",
			input:   []byte(testdata.InvalidPEM),
			wantErr: true,
		},
		{
			name:    "Empty input",
			input:   []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			err := ValidateCertificate(tt.input)
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).To(BeNil())
			}
		})
	}
}
