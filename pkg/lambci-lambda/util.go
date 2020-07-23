package lambcilambda

import (
	"bytes"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

// determine the next version from a list of layer directories
// layer directory names are supposed to end with '-<version>'
// the function finds the highest version number and adds 1
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

// convert a string map to a list of lines in key=value format
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
