package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"log"
	"math/big"
	"net"
	"os"
	"strings"
	"time"
)

var (
	host     = flag.String("host", "", "Comma-separated hostnames and IPs to generate a certificate for")
	validFor = flag.Duration("duration", 365*24*time.Hour, "Duration that certificate is valid for")
	isCA     = flag.Bool("ca", false, "whether this cert should be its own Certificate Authority")
	rsaBits  = flag.Int("rsa-bits", 2048, "Size of RSA key to generate. Ignored if --ecdsa-curve is set")
)

//func publicKey(priv interface{}) interface{} {
//	switch k := priv.(type) {
//	case *rsa.PrivateKey:
//		return &k.PublicKey
//	case *ecdsa.PrivateKey:
//		return &k.PublicKey
//	case ed25519.PrivateKey:
//		return k.Public().(ed25519.PublicKey)
//	default:
//		return nil
//	}
//}

/*

func generateRoot() *x509.Certificate {
	var priv *rsa.PrivateKey
	var err error
	priv, err = rsa.GenerateKey(rand.Reader, *rsaBits)
	if err != nil {
		log.Fatalf("Failed to generate private key: %s", err)
	}

	var notBefore = time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("Failed to generate serial number: %s", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Serverless Research Kit"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
	}
	saveCert("cert/root.crt", certBytes)
	saveKeyPair("cert/root", priv)
	return &template
}

func saveCert(fn string, cert []byte) {
	certOut, err := os.Create(fn)
	if err != nil {
		log.Fatalf("Failed to open cert.pem for writing: %s", err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: cert}); err != nil {
		log.Fatalf("Failed to write data to cert.pem: %s", err)
	}
	if err := certOut.Close(); err != nil {
		log.Fatalf("Error closing cert.pem: %s", err)
	}
	log.Printf("wrote %s\n", fn)
}

func saveKeyPair(prefix string, key *rsa.PrivateKey) {
	privateKeyFile := fmt.Sprintf("%s.key", prefix)
	keyOut, err := os.OpenFile(privateKeyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Failed to open %s for writing:", privateKeyFile, err)
		return
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		log.Fatalf("Unable to marshal private key: %v", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		log.Fatalf("Failed to write data %s: %s", privateKeyFile, err)
	}
	if err := keyOut.Close(); err != nil {
		log.Fatalf("Error closing %s: %s", privateKeyFile, err)
	}
	log.Printf("wrote %s\n", privateKeyFile)

	publicKeyFile := fmt.Sprintf("%s.pub", prefix)
	pubKeyOut, err := os.OpenFile(publicKeyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Failed to open %s for writing:", publicKeyFile, err)
		return
	}
	pubBytes, err := x509.MarshalPKCS8PrivateKey(&key.PublicKey)
	if err != nil {
		log.Fatalf("Unable to marshal private key: %v", err)
	}
	if err := pem.Encode(pubKeyOut, &pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes}); err != nil {
		log.Fatalf("Failed to write data %s: %s", publicKeyFile, err)
	}
	if err := keyOut.Close(); err != nil {
		log.Fatalf("Error closing %s: %s", publicKeyFile, err)
	}
	log.Printf("wrote %s\n", publicKeyFile)

}

func generateKey(parent *x509.Certificate, client bool) {
	var priv *rsa.PrivateKey
	var err error
	priv, err = rsa.GenerateKey(rand.Reader, *rsaBits)
	if err != nil {
		log.Fatalf("Failed to generate private key: %s", err)
	}

	var notBefore = time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("Failed to generate serial number: %s", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Serverless Research Kit"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,

		BasicConstraintsValid: true,
	}
	if client {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	} else {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
	}


}
*/

func main() {
	flag.Parse()

	if len(*host) == 0 {
		log.Fatalf("Missing required --host parameter")
	}

	//rootKey := generateRoot()
	//generateServerKey(rootKey)
	//generateClientKey(rootKey)

	var priv *rsa.PrivateKey
	var err error
	priv, err = rsa.GenerateKey(rand.Reader, *rsaBits)
	if err != nil {
		log.Fatalf("Failed to generate private key: %s", err)
	}

	var notBefore = time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("Failed to generate serial number: %s", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Serverless Research Kit"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	hosts := strings.Split(*host, ",")
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	if *isCA {
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
	}

	certOut, err := os.Create("cert.pem")
	if err != nil {
		log.Fatalf("Failed to open cert.pem for writing: %s", err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		log.Fatalf("Failed to write data to cert.pem: %s", err)
	}
	if err := certOut.Close(); err != nil {
		log.Fatalf("Error closing cert.pem: %s", err)
	}
	log.Print("wrote cert.pem\n")

	keyOut, err := os.OpenFile("key.pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Failed to open key.pem for writing:", err)
		return
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		log.Fatalf("Unable to marshal private key: %v", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		log.Fatalf("Failed to write data to key.pem: %s", err)
	}
	if err := keyOut.Close(); err != nil {
		log.Fatalf("Error closing key.pem: %s", err)
	}
	log.Print("wrote key.pem\n")
}
