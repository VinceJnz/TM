package appCore

import (
	"crypto/tls"
	"log"
	"net/http"
)

type DebugWriter struct {
}

func (d DebugWriter) Write(p []byte) (int, error) {
	log.Printf("debugWrite()1 p: %+v", string(p))
	return 0, nil
}

// ...set up caCertPool, etc...
//tlsConfig := &tls.Config{
//    ClientAuth: tls.RequireAndVerifyClientCert,
//    ClientCAs:  caCertPool,
//    MinVersion: tls.VersionTLS12,
//}

// func startServer(addr, crtFile, keyFile string, handler http.Handler, isTLS bool, debugTag string) {
func StartServerHTTP(addr, crtFile, keyFile string, handler http.Handler, isTLS bool, debugTag string, tlsConfig *tls.Config) {
	go func() {
		protocol := "HTTP"
		scheme := "http"
		log.Printf("%s%s server running on %s://localhost%s", debugTag, protocol, scheme, addr)
		var err error

		err = http.ListenAndServe(addr, handler)

		if err != nil {
			log.Fatalf("%s%s server error: %v", debugTag, protocol, err)
		}
	}()
}

func StartServerHTTPS(addr, crtFile, keyFile string, handler http.Handler, isTLS bool, debugTag string, tlsConfig *tls.Config) {
	go func() {
		protocol := "HTTPS"
		scheme := "https"
		log.Printf("%s%s server running on %s://localhost%s", debugTag, protocol, scheme, addr)
		var err error
		server := &http.Server{
			Addr:      addr,
			Handler:   handler,
			TLSConfig: tlsConfig,
		}
		err = server.ListenAndServeTLS(crtFile, keyFile)

		if err != nil {
			log.Fatalf("%s%s server error: %v", debugTag, protocol, err)
		}
	}()
}
