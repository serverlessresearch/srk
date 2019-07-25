package cfbench

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

func ExperimentServer(progress *progress, logWriter chan string, alldone chan struct{}) {
	var srv = http.Server{Addr: ":3000"}
	var shutdownRequested = make(chan struct{})
	go func() {
		<-shutdownRequested
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
		close(alldone)
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintf(w, "Serverless Experiment Controller")
		if err != nil {
			log.Fatal(err)
		}
	})

	checkAllDone := func() {
		if progress.allDone() {
			log.Printf("finished processing responses for experiment %s", progress.experimentId)
			close(shutdownRequested)
		}
	}

	http.HandleFunc("/event", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Error reading body", 500)
				return
			}
			logWriter <- string(body)
			var data map[string]interface{}
			if err := json.Unmarshal(body, &data); err != nil {
				http.Error(w, "Error parsing body", 400)
				return
			}
			switch data["action"] {
			case "begin":
				progress.setRunning(data["uuid"].(string))
			case "end":
				progress.setDone(data["uuid"].(string))
				checkAllDone()
			}
			//log.Print(data)
			_, err = fmt.Fprintf(w, "Thanks for the event.")
			if err != nil {
				log.Fatal(err)
			}
		default:
			http.Error(w, "Must use POST", 400)
		}
	})

	http.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Error reading body", 500)
				return
			}
			logWriter <- string(body)
			var data map[string]interface{}
			if err := json.Unmarshal(body, &data); err != nil {
				http.Error(w, "Error parsing body", 400)
				return
			}
			progress.setData(data["uuid"].(string))
			//log.Print(data)
			checkAllDone()
			_, err = fmt.Fprintf(w, "Thanks for the data.")
			if err != nil {
				log.Fatal(err)
			}
		default:
			http.Error(w, "Must use POST", 400)
		}
	})

	log.Print("starting server")
	if err := srv.ListenAndServe(); err != nil {
		switch err {
		case http.ErrServerClosed:
			log.Printf("server shutting down")
		default:
			log.Fatal(err)
		}
	}
}
