package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

func genNewCert(path string) error {
	var priv *ecdsa.PrivateKey
	var err error
	var notBefore, notAfter time.Time

	priv, err = ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		return err
	}

	notBefore = time.Now()

	notAfter = time.Now().Add(3.15569e7 * time.Second)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)

	if err != nil {
		return err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Company, Inc."},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	template.DNSNames = append(template.DNSNames, "localhost")

	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign

	pub := &priv.PublicKey
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, pub, priv)
	if err != nil {
		return err
	}

	certOut, err := os.Create(filepath.Join(path, "cert.pem"))
	if err != nil {
		return err
	}

	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	certOut.Close()

	keyOut, err := os.OpenFile("cert.key", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	b, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return err
	}

	pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: b})

	keyOut.Close()

	return nil
}

func removeGenCert(path string) {
	os.Remove(filepath.Join(path, "cert.pem"))
	os.Remove(filepath.Join(path, "cert.key"))
}
