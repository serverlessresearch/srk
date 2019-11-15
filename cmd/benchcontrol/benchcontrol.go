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

func NewServer(launch <-chan cfbench.LaunchMessage, complete chan<- cfbench.CompletionMessage) (*BenchControlServer, error) {
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
			switch data["Action"] {
			case "begin":
				startTime := time.Now()
				log.Printf("begin action on ExecutionId: %s - WAITING\n", data["ExecutionId"])
				lm := <-launch
				log.Printf("begin action on ExecutionId: %s - RECEIVED LAUNCH COMMAND\n", data["ExecutionId"])
				waitTime := time.Now().Sub(startTime)
				runDuration := lm.Duration
				if timeoutProvided, ok := data["Timeout"]; ok {
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
				log.Printf("end action on ExecutionId: %s\n", data["ExecutionId"])
				var referenceIdInterface interface{}
				var referenceId string
				var ok bool
				if referenceIdInterface, ok = data["InvocationId"]; !ok {
					panic("xxx")
				}
				if referenceId, ok = referenceIdInterface.(string); !ok {
					panic("xxx")
				}
				//log.Printf("end action: %s\n", referenceId)
				complete <- cfbench.CompletionMessage{referenceId, true}
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

	var srv = http.Server{Addr: ":6001", Handler: mux}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			switch err {
			case http.ErrServerClosed:
				log.Printf("server shutting down")
			default:
				log.Fatal(err)
			}
		}
	}()
	return &BenchControlServer{srv}, nil
}


func main() {
	var launchChannel = make(chan cfbench.LaunchMessage)
	var completionChannel = make(chan cfbench.CompletionMessage)
	r := &cfbench.ExperimentRunner{
		LaunchChannel:     launchChannel,
		CompletionChannel: completionChannel,
	}
	_, err := NewServer(launchChannel, completionChannel)

	invoker := cfbench.NewLambdaAsyncInvoker()
	go func() {
		for {
			lm := <-launchChannel
			err := invoker.Invoke(lm.ReferenceId, "exp", "http://172.31.23.31:6001/", "sleepworkload", nil)
			if err != nil {
				completionChannel <- cfbench.CompletionMessage{lm.ReferenceId, false}
			}
		}
	}()

	err = rpc.Register(r)
	if err != nil {
		log.Fatal(err)
	}

	cert, err := tls.LoadX509KeyPair("config/server.crt", "config/server.key")
	if err != nil {
		log.Fatalf("failed to load server key: %s", err)
	}
	if len(cert.Certificate) != 2 {
		log.Fatal("server.crt should have 2 certificates")
	}
	ca, err := x509.ParseCertificate(cert.Certificate[1])
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
	listener, err := tls.Listen("tcp", ":6000", &config)
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Benchmark control server started")
	//noinspection GoUnhandledErrorResult
	defer listener.Close()
	rpc.Accept(listener)
}
