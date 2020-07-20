package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	_"github.com/pkg/errors"
	"webhook-injector/pkg/infector"

	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)



func LoadToken(path string) (string, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data[:]), nil
}

func LoadCA(caFile string) (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	if ca, err := ioutil.ReadFile(caFile); err != nil {
		return nil, err
	} else {
		pool.AppendCertsFromPEM(ca)
	}
	return pool, nil
}

func main() {
	var parameters infector.WhSvrParameters

	// get command line parameters
	flag.IntVar(&parameters.Port, "port", 443, "Webhook server port.")
	flag.StringVar(&parameters.CertFile, "tlsCertFile", "/etc/webhook/certs/cert.pem", "File containing the x509 Certificate for HTTPS.")
	flag.StringVar(&parameters.KeyFile, "tlsKeyFile", "/etc/webhook/certs/key.pem", "File containing the x509 private key to --tlsCertFile.")
	flag.StringVar(&parameters.Token, "authToken", "/var/run/secrets/kubernetes.io/serviceaccount/token", "Token to communicate with apiServer")
	flag.StringVar(&parameters.Crt, "ca", "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt", "crt to verify api server")
	flag.Parse()

	// load apiServer ca.crt
	certPool, err := LoadCA(parameters.Crt)
	if err != nil {
		log.Panicf("Filed to load crt: %v", err)
	}

	// load tls key pair
	pair, err := tls.LoadX509KeyPair(parameters.CertFile, parameters.KeyFile)
	if err != nil {
		log.Panicf("Filed to load key pair: %v", err)
	}

	// build server
	whsvr := infector.WebhookServer{
			Client: &http.Client{Transport: &http.Transport{
				TLSClientConfig: &tls.Config{RootCAs: certPool},
			},
		},
		Server: &http.Server{
			Addr:      fmt.Sprintf(":%v", parameters.Port),
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
		},
	}

	// define http server and server handler
	mux := http.NewServeMux()
	mux.HandleFunc("/mutate", whsvr.ServeInject)
	whsvr.Server.Handler = mux

	// start webhook server in new rountine
	go func() {
		if err := whsvr.Server.ListenAndServeTLS("", ""); err != nil {
			log.Panicf("Filed to listen and serve webhook server: %v", err)
		}
	}()

	// listening OS shutdown singal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	fmt.Printf("Got OS shutdown signal, shutting down wenhook server gracefully...")
	whsvr.Server.Shutdown(context.Background())
}
