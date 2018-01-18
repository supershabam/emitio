package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"

	"golang.org/x/sync/errgroup"

	quic "github.com/lucas-clemente/quic-go"
)

func main() {
	var (
		addr = "127.0.0.1:8080"
	)
	ctx := context.Background()
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		log.Println("server: listening")
		l, err := quic.ListenAddr(addr, generateTLSConfig(), nil)
		if err != nil {
			return err
		}
		log.Println("server: accepting")
		sess, err := l.Accept()
		if err != nil {
			return err
		}
		log.Println("server: open stream")
		s, err := sess.OpenStream()
		if err != nil {
			return err
		}
		log.Println("server: write")
		_, err = s.Write([]byte("hello"))
		if err != nil {
			return err
		}
		log.Println("server: exit")
		return nil
	})
	eg.Go(func() error {
		log.Println("client: dial")
		sess, err := quic.DialAddr(addr, &tls.Config{InsecureSkipVerify: true}, nil)
		if err != nil {
			return err
		}
		log.Println("client: accept stream")
		s, err := sess.AcceptStream()
		if err != nil {
			return err
		}
		log.Println("client: read")
		buf := make([]byte, 1024)
		nr, err := s.Read(buf)
		if nr > 0 {
			fmt.Printf("read: %s\n", buf[:nr])
		}
		if err != nil {
			return err
		}
		return nil
	})
	err := eg.Wait()
	if err != nil {
		panic(err)
	}
}

// Setup a bare-bones TLS config for the server
func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{Certificates: []tls.Certificate{tlsCert}}
}
