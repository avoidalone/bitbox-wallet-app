package main

import "C"

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/shiftdevices/godbb/backend"
	backendHandlers "github.com/shiftdevices/godbb/backend/handlers"
	"github.com/shiftdevices/godbb/util/freeport"
)

const (
	// RSA key size.
	rsaBits = 2048
	// Name of the server certificate
	tlsServerCertificate = "config/server.pem"
)

// generateRSAPrivateKey generates an RSA key pair and wraps it in the type rsa.PrivateKey.
func generateRSAPrivateKey() (*rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		log.Fatalf("Failed to create private key: %s", err)
		return nil, err
	}
	return privateKey, nil
}

// createSelfSignedCertificate creates a self-signed certificate from the given rsa.PrivateKey.
func createSelfSignedCertificate(privateKey *rsa.PrivateKey) ([]byte, error) {
	serialNumber := big.Int{}
	notBefore := time.Now()
	// Invalid after one day.
	notAfter := notBefore.AddDate(0, 0, 1)
	template := x509.Certificate{
		SerialNumber: &serialNumber,
		Subject: pkix.Name{
			Country:            []string{"CH"},
			Organization:       []string{"Shift Cryptosecurity"},
			OrganizationalUnit: []string{"godbb"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1), net.ParseIP("::1")},
		DNSNames:              []string{"localhost"},
		IsCA:                  true,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, privateKey.Public(), privateKey)
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
		return nil, err
	}
	return derBytes, nil
}

// saveAsPEM saves the given PEM block as a file
func saveAsPEM(name string, pemBytes *pem.Block) error {
	certificateDir := filepath.Dir(name)
	err := os.MkdirAll(certificateDir, os.ModeDir|os.ModePerm)
	if err != nil {
		log.Fatalf("failed to create directory %s: %s", certificateDir, err)
		return err
	}
	pemFile, err := os.Create(name)
	if err != nil {
		log.Fatalf("failed to open %s for writing: %s", name, err)
		return err
	}
	err = pem.Encode(pemFile, pemBytes)
	if err != nil {
		log.Fatalf("failed to write PEM encoded file %s: %s", pemFile.Name(), err)
		return err
	}
	err = pemFile.Close()
	if err != nil {
		log.Fatalf("failed to close PEM file %s: %s", pemFile.Name(), err)
		return err
	}
	return nil
}

// derToPem wraps the givem PEM bytes and PEM type in a PEM block.
func derToPem(pemType string, pemBytes []byte) *pem.Block {
	return &pem.Block{Type: pemType, Bytes: pemBytes}
}

// Copied and adapted from package http server.go.
//
// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

// accept enables TCP keep alive and sets the period to 3 minutes.
func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

//export serve
func serve() int {
	port, err := freeport.FreePort()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Port:", port)
	handlers := backendHandlers.NewHandlers(backend.NewBackend(), port)

	privateKey, err := generateRSAPrivateKey()
	if err != nil {
		log.Fatal(err)
	}
	certificate, err := createSelfSignedCertificate(privateKey)
	if err != nil {
		log.Fatal(err)
	}
	certificatePEM := derToPem("CERTIFICATE", certificate)
	saveAsPEM(tlsServerCertificate, certificatePEM)

	var certAndKey tls.Certificate
	certAndKey.Certificate = [][]byte{certificate}
	certAndKey.PrivateKey = privateKey

	go func() {
		server := &http.Server{
			Addr:    fmt.Sprintf("localhost:%d", port),
			Handler: handlers.Router,
			TLSConfig: &tls.Config{
				NextProtos:   []string{"http/1.1"},
				Certificates: []tls.Certificate{certAndKey},
			},
		}
		listener, err := net.Listen("tcp", server.Addr)
		if err != nil {
			log.Fatal(err)
		}
		tlsListener := tls.NewListener(tcpKeepAliveListener{listener.(*net.TCPListener)}, server.TLSConfig)
		err = server.Serve(tlsListener)
		if err != nil {
			log.Fatal(err)
		}
	}()
	return port
}

// Don't remove - needed for the C compilation.
func main() {
}
