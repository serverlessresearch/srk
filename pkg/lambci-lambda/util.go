package lambcilambda

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	maxRetries = 3
	retryDelay = 1 * time.Second
)

func NextLayerVersion(layers []string) int {

	maxVersion := 0

	for _, layer := range layers {

		if layer == "" {
			continue
		}
		name := strings.Split(layer, "-")

		version, err := strconv.Atoi(name[len(name)-1])
		if err != nil {
			// warn but do not break on wrongly named layers
			log.Warnf("could not find version in layer name '%s'", layer)
			continue
		}

		if version > maxVersion {
			maxVersion = version
		}
	}

	return maxVersion + 1
}

func Map2Lines(m map[string]string) string {

	var lines bytes.Buffer
	for key, value := range m {
		lines.WriteString(key)
		lines.WriteString("=")
		lines.WriteString(value)
		lines.WriteString("\n")
	}
	return lines.String()
}

func HttpPost(url, data string) (*bytes.Buffer, error) {

	doPost := func() (*bytes.Buffer, error) {

		response, err := http.Post(url, "application/json", strings.NewReader(data))
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("POST %s (%s) returned status %s", url, data, response.Status)
		}

		result := new(bytes.Buffer)
		_, err = result.ReadFrom(response.Body)
		if err != nil {
			return nil, err
		}

		return result, nil
	}

	var err error
	var result *bytes.Buffer

	retries := 0
	for {

		result, err = doPost()
		if err == nil || retries >= maxRetries {
			break
		}

		time.Sleep(retryDelay)
		retries++
	}

	return result, err
}
