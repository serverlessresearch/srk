package lambdalike

import (
	"crypto/sha256"
	"encoding/base64"
	"strconv"

	"github.com/aws/aws-sdk-go/service/lambda"
)

type WorkerFunctionConfiguration struct {
	FunctionConfiguration *lambda.FunctionConfiguration
	RuntimeAddr           string
	NumInstances          int
}

// Return a hash value of key configuration information
func (wfc *WorkerFunctionConfiguration) hash() string {
	h := sha256.New()
	h.Write([]byte(wfc.RuntimeAddr))
	// h.Write([]byte(*wfc.fc.FunctionArn))
	h.Write([]byte(*wfc.FunctionConfiguration.FunctionName))
	h.Write([]byte(*wfc.FunctionConfiguration.Version))
	h.Write([]byte(*wfc.FunctionConfiguration.Handler))
	h.Write([]byte(strconv.FormatInt(*wfc.FunctionConfiguration.MemorySize, 10)))
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}
