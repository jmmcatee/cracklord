package common

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
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
func WriteCertificateToFile(cert *x509.Certificate, filepath string) error {
	return writePEMFile(cert.Raw, filepath, "CERTIFICATE")
}

// WriteRSAPrivateKeyToPEM converts a RSA Private Key to PEM and writes to filepath
func WriteRSAPrivateKeyToFile(key *rsa.PrivateKey, filepath string) error {
	return writePEMFile(x509.MarshalPKCS1PrivateKey(key), filepath, "RSA PRIVATE KEY")
}

// Converts a certificate to the PEM encoding format and puts it into a string
func WriteCertificateToString(cert *x509.Certificate) (string, error) {
	var b bytes.Buffer
	err := writePEMBytes(cert.Raw, &b, "CERTIFICATE")
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

// Converts a private key to the PEM format and puts it into a string
func WriteRSAPrivateKeyToString(key *rsa.PrivateKey) (string, error) {
	var b bytes.Buffer
	err := writePEMBytes(x509.MarshalPKCS1PrivateKey(key), &b, "RSA PRIVATE KEY")
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

func writePEMBytes(raw []byte, writer io.Writer, pemtype string) error {
	pemBlock := pem.Block{
		Type:  pemtype,
		Bytes: raw,
	}

	err := pem.Encode(writer, &pemBlock)
	if err != nil {
		return err
	}

	return nil
}

func writePEMFile(raw []byte, filepath string, pemtype string) error {
	pemFile, err := os.Create(filepath)
	if err != nil {
		return err
	}

	err = writePEMBytes(raw, pemFile, pemtype)
	if err != nil {
		return err
	}

	pemFile.Close()
	return nil
}
