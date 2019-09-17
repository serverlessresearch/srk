// Common configuration/setup functions
package cmd

import (
	awslambda "github.com/serverlessresearch/srk/pkg/aws-lambda"
	"github.com/serverlessresearch/srk/pkg/openlambda"
	"github.com/serverlessresearch/srk/pkg/srk"
	"github.com/spf13/viper"
)

func getProvider() *srk.Provider {
	return &srk.Provider{getFunctionService()}
}

func destroyProvider(p *srk.Provider) {
	p.Faas.Destroy()
}

func getFunctionService() srk.FunctionService {
	// Setup the default function service
	providerName := viper.GetString("default-provider")
	if providerName == "" {
		panic("No default provider in configuration")
	}

	serviceName := viper.GetString("providers." + providerName + ".faas")
	if serviceName == "" {
		panic("Provider \"" + providerName + "\" does not provide a FaaS service")
	}

	var service srk.FunctionService
	switch serviceName {
	case "openLambda":
		service = openlambda.NewConfig(
			viper.GetString("service.faas.openLambda.olcmd"),
			viper.GetString("service.faas.openLambda.oldir"))
	case "awsLambda":
		service = awslambda.NewConfig(
			viper.GetString("service.faas.awsLambda.role"),
			viper.GetString("service.faas.awsLambda.vpc-config"),
			viper.GetString("service.faas.awsLambda.region"))
	default:
		panic("Unrecognized FaaS service: " + serviceName)
	}

	return service
}
