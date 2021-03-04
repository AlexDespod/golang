package main

import (
	"crypto/tls"
	"flag"
	"io"
	"net"
	"net/http"
	"time"
)

func handleTunneling(w http.ResponseWriter, r *http.Request) {
	destConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
	go transfer(destConn, clientConn)
	go transfer(clientConn, destConn)
}
func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}
func handleHTTP(w http.ResponseWriter, req *http.Request) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
func handlReq(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		handleTunneling(w, r)
	} else {
		handleHTTP(w, r)
	}
}

// func redirectTLS(w http.ResponseWriter, r *http.Request) {
// 	http.Redirect(w, r, "https://localhost:8888"+r.RequestURI, http.StatusMovedPermanently)
// }
func main() {
	var pemPath string
	flag.StringVar(&pemPath, "pem", "./tls/cert.pem", "path to pem file")
	var keyPath string
	flag.StringVar(&keyPath, "key", "./tls/key.pem", "path to key file")

	flag.Parse()

	server := &http.Server{
		Addr:    ":8080",
		Handler: http.HandlerFunc(handlReq),
		// Disable HTTP/2.

	}
	serverTLS := &http.Server{
		Addr:    ":8888",
		Handler: http.HandlerFunc(handlReq),
		// Disable HTTP/2.
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	go server.ListenAndServe()

	serverTLS.ListenAndServeTLS(pemPath, keyPath)

}
