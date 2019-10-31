package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"os/signal"
	"time"

	"go.dedis.ch/example-grpc/count"
	"go.dedis.ch/example-grpc/overlay"
)

func makeCertificate() *tls.Certificate {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("Couldn't generate the private key: %+v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "localhost",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 180),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	buf, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		log.Fatalf("Couldn't create the certificate: %+v", err)
	}

	cert, err := x509.ParseCertificate(buf)
	if err != nil {
		log.Fatalf("Couldn't parse the certificate: %+v", err)
	}

	return &tls.Certificate{
		Certificate: [][]byte{buf},
		PrivateKey:  priv,
		Leaf:        cert,
	}
}

func main() {
	roster := make(overlay.Roster, 5)

	for i := range roster {
		ident := overlay.Identity{
			Port:        fmt.Sprintf(":%d", 3000+i),
			Certificate: makeCertificate(),
		}
		roster[i] = overlay.Peer{
			Port:        ident.Port,
			Certificate: ident.Certificate.Leaf,
		}

		if i == 0 {
			block := &pem.Block{
				Type:  "CERTIFICATE",
				Bytes: ident.Certificate.Certificate[0],
			}

			f, err := os.Create("cert.pem")
			if err != nil {
				log.Fatal(err)
			}
			err = pem.Encode(f, block)
			if err != nil {
				log.Fatal(err)
			}
		}

		overlay := overlay.NewOverlay(ident)
		overlay.SetRoster(roster)

		count.RegisterCountServer(overlay.Server, count.NewService(overlay))

		lis, err := net.Listen("tcp", ident.Port)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}

		go func() {
			log.Printf("Server %s is starting...\n", lis.Addr().String())
			if err := overlay.Serve(lis); err != nil {
				log.Fatalf("failed to serve: %v", err)
			}
		}()

		go func() {
			log.Printf("Proxy server %s is starting...\n", overlay.WebProxy.Addr)
			if err := overlay.WebProxy.ListenAndServeTLS("", ""); err != nil {
				log.Fatalf("failed to start https proxy for grpc-web: %v", err)
			}
		}()
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch
}
