package cfbench

import (
	"fmt"
	"log"
	"os"
)

func ConcurrencySweep(logfile string) {
	progress := new(Progress)
	serverWorking := make(chan struct{})

	logWriter := make(chan string)
	go func() {
		f, err := os.OpenFile(logfile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
		if err != nil {
			panic(err)
		}
		for s := range logWriter {
			_, err = fmt.Fprintf(f, "%s\n", s)
			if err != nil {
				log.Printf("Unable to log %v", err)
			}
		}
	}()

	go ExperimentServer(progress, logWriter, serverWorking)
	go invokeMulti(progress, 100)

	<-serverWorking

	// TODO is this right?
	close(logWriter)
}