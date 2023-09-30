// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/9/23

package internal

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"io"
	"log"
	"net"
	"time"
)

//go:embed localcerts/public.crt
var certPem []byte

//go:embed localcerts/private.key
var keyPem []byte

func NetCopy(in net.Conn, out net.Conn) error {
	defer in.Close()
	defer out.Close()
	ec := make(chan error, 2)
	go func() {
		_, err := io.Copy(in, out)
		ec <- err
	}()
	go func() {
		_, err := io.Copy(out, in)
		ec <- err
	}()
	return <-ec
}

func SetConnFlags(conn net.Conn) {
	tc, ok := conn.(*net.TCPConn)
	if !ok {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Second)
	tc.MultipathTCP()
}

var (
	ClientTlsConfig *tls.Config
	ServerTlsConfig *tls.Config
)

type Dialer interface {
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}

func init() {
	cert, err := tls.X509KeyPair(certPem, keyPem)
	if err != nil {
		log.Fatal(err)
	}
	ServerTlsConfig = &tls.Config{
		ClientAuth:   tls.VerifyClientCertIfGiven,
		Certificates: []tls.Certificate{cert},
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(certPem)
	ClientTlsConfig = &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: true,
	}
}
