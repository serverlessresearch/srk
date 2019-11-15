package main

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/serverlessresearch/srk/pkg/cfbench"
	"io/ioutil"
	"log"
	"net/http"
	"net/rpc"
	"strconv"
	"time"
)

type BenchControlServer struct {
	httpServer http.Server
}

type BenchControlHandler struct {
}

func (h *BenchControlHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

}

type RunCommand struct {
	RunDuration float64
	Reference   string
}

func NewServer(launch chan cfbench.LaunchMessage, complete chan cfbench.CompletionMessage) (*BenchControlServer, error) {
	var srv = http.Server{Addr: ":3000"}

	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintf(w, "Serverless Experiment Controller")
		if err != nil {
			log.Fatal(err)
		}
	}))

	mux.Handle("/event", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Error reading body", 500)
				return
			}
			log.Printf(string(body))
			//logWriter <- string(body)
			var data map[string]interface{}
			if err := json.Unmarshal(body, &data); err != nil {
				http.Error(w, "Error parsing body", 400)
				return
			}
		Actions:
			switch data["action"] {
			case "begin":
				startTime := time.Now()
				lm := <-launch
				waitTime := time.Now().Sub(startTime)
				runDuration := lm.Duration
				if timeoutProvided, ok := data["timeout"]; ok {
					timeout := timeoutProvided.(json.Number)
					ftimeout, err := timeout.Float64()
					if err != nil {
						panic(err)
					}
					maxDuration := time.Duration(ftimeout*1000000000) - waitTime - time.Second
					if runDuration < maxDuration {
						runDuration = maxDuration
					}
				}
				var cmdJson []byte
				cmdJson, err = json.Marshal(RunCommand{
					RunDuration: runDuration.Seconds(),
					Reference:   string(lm.ReferenceId),
				})
				if err != nil {
					break
				}
				var lenWritten int
				for len(cmdJson) > 0 {
					lenWritten, err = w.Write(cmdJson)
					if err != nil {
						break Actions
					}
					cmdJson = cmdJson[lenWritten:]
				}
			case "end":
				var referenceId int
				referenceId, err = strconv.Atoi(data["Reference"].(string))
				if err != nil {
					break
				}
				complete <- cfbench.CompletionMessage{referenceId}
				_, err = fmt.Fprintf(w, "Thanks for the event.")
			}
			if err != nil {
				log.Print(err)
				http.Error(w, "error handling event", 500)
			}
		default:
			http.Error(w, "Must use POST", 400)
		}
	}))

	if err := srv.ListenAndServe(); err != nil {
		switch err {
		case http.ErrServerClosed:
			log.Printf("server shutting down")
		default:
			log.Fatal(err)
		}
	}
	return &BenchControlServer{srv}, nil
}


func main() {
	r := &cfbench.ExperimentRunner{}
	err := rpc.Register(r)
	if err != nil {
		log.Fatal(err)
	}

	cert, err := tls.LoadX509KeyPair("config/server.crt", "config/server.key")
	if err != nil {
		log.Fatalf("failed to load srk key: %s", err)
	}
	if len(cert.Certificate) != 1 {
		log.Fatal("srk.crt should have 1 certificate")
	}
	ca, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		log.Fatal(err)
	}
	certPool := x509.NewCertPool()
	certPool.AddCert(ca)
	config := tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	}
	config.Rand = rand.Reader
	listener, err := tls.Listen("tcp", "0.0.0.0:6000", &config)
	if err != nil {
		log.Fatal(err)
	}
	//noinspection GoUnhandledErrorResult
	defer listener.Close()
	rpc.Accept(listener)
}
