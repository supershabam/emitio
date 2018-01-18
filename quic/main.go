package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"time"

	"golang.org/x/sync/errgroup"

	quic "github.com/lucas-clemente/quic-go"
)

var _ net.Listener = &listener{}

type listener struct {
	sess quic.Session
}

var _ net.Addr = &_addr{}

type _addr struct {
}

func (a *_addr) Network() string {
	return "bs"
}

func (a *_addr) String() string {
	return "wrong"
}

type _conn struct {
	s quic.Stream
}

// Read reads data from the connection.
// Read can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
func (c _conn) Read(b []byte) (n int, err error) {
	return c.s.Read(b)
}

// Write writes data to the connection.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
func (c _conn) Write(b []byte) (n int, err error) {
	return c.s.Write(b)
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (c _conn) Close() error {
	return c.s.Close()
}

// LocalAddr returns the local network address.
func (c _conn) LocalAddr() net.Addr {
	return &_addr{}
}

// RemoteAddr returns the remote network address.
func (c _conn) RemoteAddr() net.Addr {
	return &_addr{}
}

// SetDeadline sets the read and write deadlines associated
// with the connection. It is equivalent to calling both
// SetReadDeadline and SetWriteDeadline.
//
// A deadline is an absolute time after which I/O operations
// fail with a timeout (see type Error) instead of
// blocking. The deadline applies to all future and pending
// I/O, not just the immediately following call to Read or
// Write. After a deadline has been exceeded, the connection
// can be refreshed by setting a deadline in the future.
//
// An idle timeout can be implemented by repeatedly extending
// the deadline after successful Read or Write calls.
//
// A zero value for t means I/O operations will not time out.
func (c _conn) SetDeadline(t time.Time) error {
	return c.s.SetDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
// A zero value for t means Read will not time out.
func (c _conn) SetReadDeadline(t time.Time) error {
	return c.s.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (c _conn) SetWriteDeadline(t time.Time) error {
	return c.s.SetWriteDeadline(t)
}

func (l *listener) Accept() (net.Conn, error) {
	s, err := l.sess.AcceptStream()
	if err != nil {
		return nil, err
	}
	return _conn{s}, nil
}

func (l *listener) Close() error {
	return l.sess.Close(nil)
}

func (l *listener) Addr() net.Addr {
	return &_addr{}
}

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
		for {
			log.Println("server: accepting")
			sess, err := l.Accept()
			if err != nil {
				return err
			}
			hclient := &http.Client{
				Transport: &http.Transport{
					DialContext: func(context.Context, string, string) (net.Conn, error) {
						s, err := sess.OpenStream()
						if err != nil {
							return nil, err
						}
						return _conn{s}, nil
					},
				},
			}
			resp, err := hclient.Get("http://example.com/test")
			if err != nil {
				return nil
			}
			fmt.Printf("status_code=%d\n", resp.StatusCode)
			io.Copy(os.Stdout, resp.Body)
		}
	})
	eg.Go(func() error {
		log.Println("client: dial")
		sess, err := quic.DialAddr(addr, &tls.Config{InsecureSkipVerify: true}, nil)
		if err != nil {
			return err
		}
		l := &listener{sess}
		return http.Serve(l, nil)
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
