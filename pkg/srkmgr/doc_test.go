package srkmgr

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

func Example() {
	mgrArgs := map[string]interface{}{}
	// ./srk.yaml is an srk configuration that's been setup for your environment
	mgrArgs["config-file"] = "./srk.yaml"

	// Adding a custom logger is optional
	srkLogger := logrus.New()
	srkLogger.SetLevel(logrus.WarnLevel)
	mgrArgs["logger"] = srkLogger

	mgr, err := NewManager(mgrArgs)
	if err != nil {
		fmt.Printf("Failed to initialize: %v\n", err)
		os.Exit(1)
	}
	defer mgr.Destroy()

	// ./helloWorld points to a directory containing a hello world FaaS function
	if err = mgr.CreateRawFunction("./helloWorld", "hello", nil, nil); err != nil {
		fmt.Printf("Failed to create raw directory for lambda: %v\n", err)
		os.Exit(1)
	}
	rawDir := mgr.GetRawFunctionPath("hello")

	// Create the provider-specific representation of our function
	_, err = mgr.Provider.Faas.Package(rawDir)
	if err != nil {
		fmt.Printf("Packaging failed: %v\n", err)
		os.Exit(1)
	}

	// Upload the function to the provider
	if err := mgr.Provider.Faas.Install(rawDir, nil, ""); err != nil {
		fmt.Printf("Installation failed: %v\n", err)
		os.Exit(1)
	}

	// Actual synchronous invokation
	resp, err := mgr.Provider.Faas.Invoke("hello", `{"hello" : "world"}`)
	if err != nil {
		fmt.Printf("Failed to invoke function: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(resp.String())
}
