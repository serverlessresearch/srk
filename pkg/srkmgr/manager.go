package srkmgr

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	awslambda "github.com/serverlessresearch/srk/pkg/aws-lambda"
	"github.com/serverlessresearch/srk/pkg/openlambda"
	"github.com/serverlessresearch/srk/pkg/srk"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type SrkManager struct {
	Provider *srk.Provider
	Logger   srk.Logger
	Cfg      *viper.Viper
}

func NewManager(userCfg map[string]interface{}) (*SrkManager, error) {
	var err error
	mgr := &SrkManager{}

	if cfgPathRaw, ok := userCfg["config-file"]; ok {
		if cfgPath, ok := cfgPathRaw.(string); ok {
			err = mgr.initConfig(&cfgPath)
		} else {
			return nil, errors.New("option 'config-file' must be of type string")
		}
	} else {
		err = mgr.initConfig(nil)
	}
	if err != nil {
		return nil, err
	}

	if loggerRaw, ok := userCfg["logger"]; ok {
		if logger, ok := loggerRaw.(srk.Logger); ok {
			mgr.Logger = logger
		} else {
			return nil, errors.New("option 'logger' must satisfy srk.Logger")
		}
	} else {
		mgr.Logger = logrus.New()
	}

	mgr.Provider = &srk.Provider{}
	err = mgr.initFunctionService()
	if err != nil {
		return nil, err
	}

	return mgr, nil
}

func (self *SrkManager) Destroy() {
	self.Provider.Faas.Destroy()
}

// Returns a path to the raw directory for funcName (whether it exists or not)
func (self *SrkManager) GetRawPath(funcName string) string {
	return filepath.Join(
		self.Cfg.GetString("buildDir"),
		"functions",
		funcName)
}

// Place all provider-independent objects in a raw directory that will be
// packaged by the FaaS service. Will replace any existing rawDir.
// source: is the path to the user-provided source directory
// funcName: Unique name to give this function
// includes: List of standard SRK libraries to include (just the names of the packages, not paths)
// rawDir: Path where the rawDir should be made
func (self *SrkManager) CreateRaw(source string, funcName string, includes []string) (err error) {
	rawDir := self.GetRawPath(funcName)

	//Shared global function build directory
	fBuildDir := filepath.Join(self.Cfg.GetString("buildDir"), "functions")
	err = os.MkdirAll(fBuildDir, 0775)
	if err != nil {
		return errors.Wrap(err, "Failed to create build directory at "+fBuildDir)
	}

	// Always cleanup old raw directories first
	if err := os.RemoveAll(rawDir); err != nil {
		return errors.Wrap(err, "Failed to cleanup old build directory "+rawDir)
	}

	cmd := exec.Command("cp", "-r", source, rawDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "Adding source returned error\n%v", out)
	}

	// Copy includes into the raw directory
	for _, include := range includes {
		includePath := filepath.Join(
			self.Cfg.GetString("includeDir"),
			"python",
			include)
		if _, err := os.Stat(includePath); err != nil {
			return errors.Wrap(err, "Couldn't find include: "+include)
		}
		cmd := exec.Command("cp", "-r", includePath, rawDir)
		if err := cmd.Run(); err != nil {
			return errors.Wrap(err, "Adding include returned error")
		}
	}

	return nil
}

func (self *SrkManager) initConfig(cfgPath *string) error {
	// Setup defaults and globals here. These can be overwritten in the config,
	// but aren't included by default.

	// This is a private viper context just for srk (so as not to conflict with
	// the importer's usage).
	self.Cfg = viper.New()

	// Dumping ground for all generated output. Users should be able to "rm -rf
	// build" and get a clean system.
	self.Cfg.SetDefault("buildDir", "./build")

	// Collects all srk-provided libraries.
	self.Cfg.SetDefault("includeDir", "./includes")

	// Order of precedence: ENV, srk.yaml, "us-west-2"
	self.Cfg.SetDefault("service.faas.awsLambda.region", "us-west-2")
	self.Cfg.BindEnv("service.faas.awsLambda.region", "AWS_DEFAULT_REGION")

	if cfgPath != nil {
		// Use config file from the flag.
		self.Cfg.SetConfigFile(*cfgPath)
	} else {
		// default search path for config is ./configs/srk.* (* can be json, yaml, etc)
		self.Cfg.AddConfigPath("./configs")
		self.Cfg.SetConfigName("srk")
	}

	// If a config file is found, read it in.
	if err := self.Cfg.ReadInConfig(); err != nil {
		return errors.Wrap(err, "Failed to load config")
	}
	return nil
}

func (self *SrkManager) initFunctionService() error {
	// Setup the default function service
	providerName := self.Cfg.GetString("default-provider")
	if providerName == "" {
		return errors.New("No default provider in configuration")
	}

	serviceName := self.Cfg.GetString("providers." + providerName + ".faas")
	if serviceName == "" {
		return errors.New("Provider \"" + providerName + "\" does not provide a FaaS service")
	}

	var err error = nil
	switch serviceName {
	case "openLambda":
		self.Provider.Faas, err = openlambda.NewConfig(
			self.Logger.WithField("module", "faas.openlambda"),
			self.Cfg.Sub("service.faas.openLambda"))
	case "awsLambda":
		self.Provider.Faas, err = awslambda.NewConfig(
			self.Logger.WithField("module", "faas.awslambda"),
			self.Cfg.Sub("service.faas.awsLambda"))
	default:
		return errors.New("Unrecognized FaaS service: " + serviceName)
	}

	if err != nil {
		return errors.Wrap(err, "Failed to initialize service "+serviceName)
	}
	return nil
}
