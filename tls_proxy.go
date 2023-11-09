package boringproxy

import (
	//"errors"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"

	"github.com/caddyserver/certmagic"
)

func ProxyTcp(conn net.Conn, addr string, port int, useTls bool, certConfig *certmagic.Config) error {

	if useTls {
		tlsConfig := &tls.Config{
			GetCertificate: certConfig.GetCertificate,
		}

		tlsConfig.NextProtos = append([]string{"http/1.1", "h2", "acme-tls/1"}, tlsConfig.NextProtos...)

		tlsConn := tls.Server(conn, tlsConfig)

		tlsConn.Handshake()
		if tlsConn.ConnectionState().NegotiatedProtocol == "acme-tls/1" {
			tlsConn.Close()
			return nil
		}

		go handleConnection(tlsConn, addr, port)
	} else {
		go handleConnection(conn, addr, port)
	}

	return nil
}

func handleConnection(conn net.Conn, upstreamAddr string, port int) {

	useTls := false
	addr := upstreamAddr

	if strings.HasPrefix(upstreamAddr, "https://") {
		addr = upstreamAddr[len("https://"):]
		useTls = true
	}

	var upstreamConn net.Conn
	var err error

	if useTls {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
		}
		upstreamConn, err = tls.Dial("tcp", fmt.Sprintf("%s:%d", addr, port), tlsConfig)
	} else {
		upstreamConn, err = net.Dial("tcp", fmt.Sprintf("%s:%d", addr, port))
	}

	if err != nil {
		log.Println("Error when establishing connection:", err)
		conn.Close()
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Copy request to upstream
	go func() {
		_, err := io.Copy(upstreamConn, conn)
		if err != nil {
			log.Println("Error when copying request to upstream:", err)
		}

		wg.Done()
	}()

	// Copy response to downstream
	go func() {
		_, err := io.Copy(conn, upstreamConn)
		if err != nil {
			log.Println("Error when copying response to downstream:", err)
		}
		wg.Done()
	}()

	defer func() {
		err := upstreamConn.Close()
		if err != nil {
			log.Println("Error while closing upstream connection:", err)
		}
	}()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Println("Error while closing connection:", err)
		}
	}()
	wg.Wait()
}
