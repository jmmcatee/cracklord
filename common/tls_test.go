package common

import (
	"fmt"
	"testing"
)

func TestTLSParse(t *testing.T) {
	certPath := ""
	keyPath := ""

	cert, key, err := GetCertandKey(certPath, keyPath)
	if err != nil {
		t.Errorf("Error: %s", err.Error())
	}

	fmt.Printf("Cert: %v \n\n Key: %v\n\n", cert, key)
}
