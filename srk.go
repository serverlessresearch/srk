package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/serverlessresearch/srk/pkg/cfbench"
	"github.com/serverlessresearch/srk/pkg/cfpackage"
	"log"
	"net"
	"os"
	"strings"
)

// We structure the SRK command line tool as a single executable. We do this in hopes of bringing some cohesion to
// what could otherwise easily become a highly fragmented set of utilities. We use the subcommand pattern
// (http://blog.ralch.com/tutorial/golang-subcommands/) as is common for many cloud utilities.
func main() {
	// TODO this is a very basic implementation. Among other things improve error handling.
	if len(os.Args) < 2 {
		panic("not enough arguments")
	}

	switch os.Args[1] {
	case "bench":
		flags := flag.NewFlagSet("bench", flag.ContinueOnError)
		mode := flags.String("mode", "", "Mode of benchmark")
		functionName := flags.String("function-name", "", "Which function to run")
		functionArgs := flags.String("function-args", "{}", "Arguments to the function")
		benchParams := flags.String("params", "{}", "JSON arguments")
		trackingUrl := flags.String("tracking-url", "", "url for posting responses")
		logfile := flags.String("output", "log.txt", "Output file")

		if err := flags.Parse(os.Args[2:]); err != nil {
			panic(err)
		}

		var scanArgs cfbench.ConcurrencySweepArgs
		if err := json.Unmarshal([]byte(*benchParams), &scanArgs); err != nil {
			panic(err)
		}

		var functionArgsData map[string]interface{}
		if err := json.Unmarshal([]byte(*functionArgs), &functionArgsData); err != nil {
			panic(err)
		}

		if *trackingUrl == "" {
			ip := getLocalIp()
			*trackingUrl = fmt.Sprintf("http://%s:3000/", ip)
		}
		log.Printf("using tracking url %s", *trackingUrl)

		switch *mode {
		case "concurrency_scan":
			transitions := cfbench.GenSweepTransitions(scanArgs)
			cfbench.ConcurrencySweep(*functionName, functionArgsData, transitions, *trackingUrl, *logfile)
		default:
			panic("unknown mode")
		}
	case "package":
		flags := flag.NewFlagSet("package", flag.ContinueOnError)
		source := flags.String("source", "", "source directory or file")
		target := flags.String("target", "", "output zip file")
		include := flags.String("include", "", "what to include, e.g., bench")
		// TODO packaging for environments other than AWS Lambda

		if err := flags.Parse(os.Args[2:]); err != nil {
			panic(err)
		}
		includes := strings.Split(*include, ",")
		if err := cfpackage.Package(*source, *target, includes); err != nil {
			panic(err)
		}
	default:
		panic("unknown command")
	}
}

func getLocalIp() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}

	var interfaceAddrs []net.IP
	for _, i := range interfaces {
		addrs, err := i.Addrs()
		if err != nil {
			panic(err)
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && !ip.IsLoopback() {
				if ip4 := ip.To4(); ip4 != nil {
					interfaceAddrs = append(interfaceAddrs, ip4)
				}
			}
		}
	}
	for _, ip := range interfaceAddrs {
		if ip[0] == 10 {
			return ip.String()
		}
	}
	for _, ip := range interfaceAddrs {
		if ip[0] == 172 && ip[1] & 0xF0 == 16 {
			return ip.String()
		}
	}
	return interfaceAddrs[0].String()
}
