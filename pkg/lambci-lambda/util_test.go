package lambcilambda_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/serverlessresearch/srk/pkg/lambci-lambda"
	"github.com/stretchr/testify/assert"
)

func TestNextLayerVersion(t *testing.T) {

	assert.Equal(t, 1, lambcilambda.NextLayerVersion(nil))
	assert.Equal(t, 1, lambcilambda.NextLayerVersion([]string{}))
	assert.Equal(t, 4, lambcilambda.NextLayerVersion([]string{"layer-1", "test", "layer-3"}))
}

func TestMap2Lines(t *testing.T) {

	assert.Equal(t, "", lambcilambda.Map2Lines(nil))
	assert.Equal(t, "key1=value1\nkey2=value2\n", lambcilambda.Map2Lines(map[string]string{"key1": "value1", "key2": "value2"}))
}

func TestHttpPost(t *testing.T) {

	var received string
	data := "hello, world"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := ioutil.ReadAll(r.Body)
		received = string(data)
		w.Write(data)
	}))
	defer ts.Close()

	result, err := lambcilambda.HttpPost(ts.URL, data)
	assert.Nil(t, err)

	assert.Equal(t, data, received)
	assert.Equal(t, data, result.String())
}
