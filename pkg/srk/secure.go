package srk

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"os"
	"time"
)

const certificatePath = "configs/srk.crt"
const keyPath = "configs/srk.key"

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
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
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
	template.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign
	template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth}

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

func CreateServerKeyPair(hosts []string) ([]byte, []byte, error) {
	cert, priv, err := LoadCertificatePair()
	if err != nil {
		return nil, nil, err
	}
	return createServerKeyPair(cert, priv, hosts)
}

func createServerKeyPair(parent *x509.Certificate, signingPriv *rsa.PrivateKey, hosts []string) ([]byte, []byte, error) {
	var priv *rsa.PrivateKey
	var err error

	if parent.KeyUsage&x509.KeyUsageCertSign == 0 {
		return nil, nil, errors.New("must provide a parent key with signing usage")
	}

	hasServerAuthUsage := false
	for _, u := range parent.ExtKeyUsage {
		if u == x509.ExtKeyUsageServerAuth {
			hasServerAuthUsage = true
			break
		}
	}
	if !hasServerAuthUsage {
		return nil, nil, errors.New("must provide a parent key with server auth usage")
	}

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

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, parent, &priv.PublicKey, signingPriv)
	if err != nil {
		return nil, nil, err
	}
	if err := pem.Encode(&cert, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}); err != nil {
		return nil, nil, err
	}

	parentCert, err := ioutil.ReadFile(certificatePath)
	if err != nil {
		return nil, nil, err
	}
	cert.Write(parentCert)

	keyBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, nil, err
	}
	if err := pem.Encode(&key, &pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes}); err != nil {
		return nil, nil, err
	}

	return cert.Bytes(), key.Bytes(), nil
}

func LoadKeyPair() (*tls.Certificate, error) {
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

func LoadCertificatePair() (*x509.Certificate, *rsa.PrivateKey, error) {
	err := requireCertificates()
	if err != nil {
		return nil, nil, err
	}
	certPEMBlock, err := ioutil.ReadFile(certificatePath)
	if err != nil {
		return nil, nil, err
	}
	p, _ := pem.Decode(certPEMBlock)
	cert, err := x509.ParseCertificate(p.Bytes)
	if err != nil {
		return nil, nil, err
	}

	keyPEMBlock, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, nil, err
	}
	p, _ = pem.Decode(keyPEMBlock)
	priv, err := x509.ParsePKCS8PrivateKey(p.Bytes)
	if err != nil {
		return nil, nil, err
	}
	return cert, priv.(*rsa.PrivateKey), nil

}