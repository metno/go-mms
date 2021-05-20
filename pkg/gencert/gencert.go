/*
  Copyright 2020 MET Norway

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.
*/

package gencert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

func createPrivateKey() (*rsa.PrivateKey, error) {
	// Size of RSA key to generate
	rsaBits := 4096

	priv, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		log.Fatalf("Failed to generate private key: %v", err)
	}

	keyOut, err := os.OpenFile("key.pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Failed to open key.pem for writing: %v", err)
		return nil, err
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		log.Fatalf("Unable to marshal private key: %v", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		log.Fatalf("Failed to write data to key.pem: %v", err)
	}
	if err := keyOut.Close(); err != nil {
		log.Fatalf("Error closing key.pem: %v", err)
	}
	log.Print("wrote key.pem\n")

	return priv, nil
}

// GenerateCSR creates a signing request and stores it in a file
func GenerateCSR() func(*cli.Context) error {
	return func(ctx *cli.Context) error {

		if _, err := os.Stat("key.pem"); !os.IsNotExist(err) {
			if !ctx.Bool("overwrite") {
				log.Fatalf("The file already exists. Won't overwrite unless --overwrite is specified.")
			}
		}

		subj := pkix.Name{
			CommonName: ctx.String("common-name"),
			Country:    []string{ctx.String("country")},
		}

		template := x509.CertificateRequest{
			Subject:            subj,
			DNSNames:           strings.Split(ctx.String("alternative-names"), ","),
			SignatureAlgorithm: x509.SHA256WithRSA,
		}

		keyBytes, err := createPrivateKey()
		if err != nil {
			log.Fatalf("Failed to generate private key: %v", err)
		}
		csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &template, keyBytes)

		if err != nil {
			log.Fatalf("Failed to create certificate request: %v", err)
		}

		certOut, err := os.Create("cert.csr")
		if err != nil {
			log.Fatalf("Failed to open cert.csr for writing: %v", err)
		}
		if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes}); err != nil {
			log.Fatalf("Failed to write data to cert.pem: %v", err)
		}
		if err := certOut.Close(); err != nil {
			log.Fatalf("Error closing cert.csr: %v", err)
		}
		log.Print("wrote cert.csr\n")

		return nil
	}
}

// GenerateCertificate creates a self-signed certificate and stores it in a file
func GenerateCertificate() func(*cli.Context) error {
	return func(ctx *cli.Context) error {
		if _, err := os.Stat("key.pem"); !os.IsNotExist(err) {
			if !ctx.Bool("overwrite") {
				log.Fatalf("The file already exists. Won't overwrite unless --overwrite is specified.")
			}
		}

		// Duration the certificate is valid for
		validFor := 2 * 365 * 24 * time.Hour

		// Comma separated list of hostnames the certificate is be valid for
		host := "localhost"

		keyBytes, err := createPrivateKey()
		if err != nil {
			log.Fatalf("Failed to generate private key: %v", err)
		}

		keyUsage := x509.KeyUsageDigitalSignature
		keyUsage |= x509.KeyUsageKeyEncipherment

		notBefore := time.Now()
		notAfter := notBefore.Add(validFor)

		serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
		serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)

		if err != nil {
			log.Fatalf("Failed to generate serial number: %v", err)
		}

		template := x509.Certificate{
			SerialNumber: serialNumber,

			NotBefore: notBefore,
			NotAfter:  notAfter,

			KeyUsage:              keyUsage,
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			BasicConstraintsValid: true,
		}

		hosts := strings.Split(host, ",")
		for _, h := range hosts {
			if ip := net.ParseIP(h); ip != nil {
				template.IPAddresses = append(template.IPAddresses, ip)
			} else {
				template.DNSNames = append(template.DNSNames, h)
			}
		}

		derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &keyBytes.PublicKey, keyBytes)
		if err != nil {
			log.Fatalf("Failed to create certificate: %v", err)
		}

		certOut, err := os.Create("cert.pem")
		if err != nil {
			log.Fatalf("Failed to open cert.pem for writing: %v", err)
		}
		if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
			log.Fatalf("Failed to write data to cert.pem: %v", err)
		}
		if err := certOut.Close(); err != nil {
			log.Fatalf("Error closing cert.pem: %v", err)
		}
		log.Print("wrote cert.pem\n")

		return nil
	}
}
