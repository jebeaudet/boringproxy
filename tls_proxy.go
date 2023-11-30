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
	"time"

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

	deadline := time.Now().Add(3 * time.Hour)
	conn.SetDeadline(deadline)
	upstreamConn.SetDeadline(deadline)

	var wg sync.WaitGroup
	wg.Add(2)

	// Copy request to upstream
	go func() {
		defer wg.Done()
		defer upstreamConn.Close()
		defer conn.Close()
		_, err := io.Copy(upstreamConn, conn)
		if err != nil {
			log.Printf("Error when copying request to upstream (%s:%d): %s", upstreamAddr, port, err)
		}
	}()

	// Copy response to downstream
	go func() {
		defer wg.Done()
		defer upstreamConn.Close()
		defer conn.Close()
		_, err := io.Copy(conn, upstreamConn)
		if err != nil {
			log.Printf("Error when copying response to downstream (%s:%d): %s", upstreamAddr, port, err)
		}
	}()
	defer upstreamConn.Close()
	defer conn.Close()
	wg.Wait()
}
