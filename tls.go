package qsse

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
)

const protocol = "PROTOCOL_QUIC"

// GetDefaultTLSConfig returns a tls.Config with the default settings for server.
func GetDefaultTLSConfig() *tls.Config {
	size := 2048

	key, err := rsa.GenerateKey(rand.Reader, size)
	if err != nil {
		panic(err)
	}

	template := x509.Certificate{SerialNumber: big.NewInt(1)} //nolint:exhaustruct

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}) //nolint:lll,exhaustruct
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})                             //nolint:lll,exhaustruct

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}

	return &tls.Config{ //nolint:exhaustruct,gosec
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{protocol},
	}
}

// GetSimpleTLS returns a tls.Config with the default settings for client.
func GetSimpleTLS() *tls.Config {
	return &tls.Config{ //nolint:exhaustruct
		InsecureSkipVerify: true, //nolint:gosec
		NextProtos:         []string{protocol},
	}
}
