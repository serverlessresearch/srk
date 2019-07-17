package cfbench

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

func ExperimentServer(progress *Progress, logWriter chan string, alldone chan struct{}) {
	var srv = http.Server{Addr: ":3000"}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintf(w, "Serverless Experiment Controller")
		if err != nil {
			log.Fatal(err)
		}
	})

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
				progress.setRunning()
			case "end":
				progress.setDone()
			}
			log.Print(data)
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
			progress.setData()
			log.Print(data)
			if progress.allDone() {
				log.Print("all done!!")
				if err := srv.Shutdown(context.Background()); err != nil {
					log.Printf("HTTP server shutdown error: %v", err)
				}
				close(alldone)
			}
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
		log.Fatal(err)
	}
}
