package common

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io/ioutil"
	"math/big"
	"os"
	"time"
)

// GenerateResourceKeys generates the client authentication certificate and
// private key for TLS mutual authentication between the Queue and a Resource.
func GenerateResourceKeys(ca *x509.Certificate, caPrivKey *rsa.PrivateKey, cn string) (*x509.Certificate, *rsa.PrivateKey, error) {
	randInt, err := rand.Int(rand.Reader, big.NewInt(1048576))
	if err != nil {
		return &x509.Certificate{}, &rsa.PrivateKey{}, err
	}

	// Resource Template certificate
	var resTemplate = &x509.Certificate{
		SerialNumber: randInt,
		Subject: pkix.Name{
			Country:            []string{"United State of America"},
			Organization:       []string{"Cracklord, Inc."},
			OrganizationalUnit: []string{"Operations"},
			CommonName:         cn,
		},
		SignatureAlgorithm:    x509.SHA256WithRSA,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(2, 0, 0),
	}

	resPriv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return &x509.Certificate{}, &rsa.PrivateKey{}, err
	}
	resPub := &resPriv.PublicKey

	resDER, err := x509.CreateCertificate(rand.Reader, resTemplate, ca, resPub, caPrivKey)
	if err != nil {
		return &x509.Certificate{}, &rsa.PrivateKey{}, err
	}

	resCert, err := x509.ParseCertificate(resDER)
	if err != nil {
		return &x509.Certificate{}, &rsa.PrivateKey{}, err
	}

	return resCert, resPriv, nil
}

// Parse PEM encoded certificate and private key from file path locations
func GetCertandKey(certPath, keyPath string) (*x509.Certificate, *rsa.PrivateKey, error) {
	// Check if we can open the two file paths give
	certFile, err := os.Open(certPath)
	if err != nil {
		return &x509.Certificate{}, &rsa.PrivateKey{}, err
	}

	keyFile, err := os.Open(keyPath)
	if err != nil {
		return &x509.Certificate{}, &rsa.PrivateKey{}, err
	}

	// Get the bytes of the files to parse
	certBytes, err := ioutil.ReadAll(certFile)
	if err != nil {
		return &x509.Certificate{}, &rsa.PrivateKey{}, err
	}

	keyBytes, err := ioutil.ReadAll(keyFile)
	if err != nil {
		return &x509.Certificate{}, &rsa.PrivateKey{}, err
	}

	certBlock, _ := pem.Decode(certBytes)
	keyBlock, _ := pem.Decode(keyBytes)

	// Parse cert and key
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return &x509.Certificate{}, &rsa.PrivateKey{}, err
	}

	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return &x509.Certificate{}, &rsa.PrivateKey{}, err
	}

	return cert, key, nil
}

// WriteCertificateToPEM converts a certificate to PEM and writes to filepath
func WriteCertificateToPEM(cert *x509.Certificate, filepath string) error {
	return writePemFile(cert.Raw, filepath, "Certificate")
}

// WriteRSAPrivateKeyToPEM converts a RSA Private Key to PEM and writes to filepath
func WriteRSAPrivateKeyToPEM(key *rsa.PrivateKey, filepath string) error {
	return writePemFile(x509.MarshalPKCS1PrivateKey(key), filepath, "Certificate")
}

func writePemFile(raw []byte, filepath string, pemtype string) error {
	pemBlock := pem.Block{
		Type:  pemtype,
		Bytes: raw,
	}

	pemFile, err := os.Create(filepath)
	if err != nil {
		return err
	}

	err = pem.Encode(pemFile, &pemBlock)
	if err != nil {
		return err
	}

	pemFile.Close()
	return nil
}
