package srk

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"os"
	"time"
)

const certificatePath = "config/srk.crt"
const keyPath = "config/srk.key"

func requireCertificates() error {
	_, errCertifiate := os.Stat(certificatePath)
	_, errKey:= os.Stat(keyPath)
	if os.IsNotExist(errCertifiate) || os.IsNotExist(errKey) {
		return createCertificates()
	}
	return nil
}


func newCertificateTemplate() x509.Certificate {
	var notBefore = time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("Failed to generate serial number: %s", err)
	}

	return x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Serverless Research Kit"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,
		BasicConstraintsValid: true,
	}
}

func createCertificates() error {
	var priv *rsa.PrivateKey
	var err error
	priv, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	template := newCertificateTemplate()
	template.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
	template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return  err
	}
	certOut, err := os.Create(certificatePath)
	if err != nil {
		return err
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}); err != nil {
		return err
	}
	if err := certOut.Close(); err != nil {
		return err
	}

	keyBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return  err
	}
	keyOut, err := os.Create(keyPath)
	if err != nil {
		return err
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes}); err != nil {
		return err
	}
	if err := keyOut.Close(); err != nil {
		return err
	}
	return nil
}


func createServerKeyPair(parent *x509.Certificate, hosts []string) ([]byte, []byte, error) {
	var priv *rsa.PrivateKey
	var err error

	priv, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	var cert, key bytes.Buffer

	template := newCertificateTemplate()
	template.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
	template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}

	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, parent, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, err
	}
	if err := pem.Encode(&cert, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}); err != nil {
		return nil, nil, err
	}

	keyBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, nil, err
	}
	if err := pem.Encode(&key, &pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes}); err != nil {
		return nil, nil, err
	}

	return cert.Bytes(), key.Bytes(), nil
}

func LoadCertificates() (*tls.Certificate, error) {
	err := requireCertificates()
	if err != nil {
		return nil, err
	}
	cert, err := tls.LoadX509KeyPair(certificatePath, keyPath)
	if err != nil {
		return nil, err
	}
	return &cert, err
}
