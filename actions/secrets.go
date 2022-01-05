// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package plugin_data

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func SetSecret(cipherPhrase *string) error {
	splits := strings.SplitN(*cipherPhrase, ":", 2)
	if splits == nil { // Just plain text, nothing to do
		return nil
	}
	var fetchMethod, fetchArg string
	var err error

	fetchMethod = splits[0]
	fetchArg = splits[1]

	switch fetchMethod {
	case "file": // Look up secret according to file path eg Kubernetes secrets
		*cipherPhrase, err = fetchFromFile(fetchArg)
	case "env":
		*cipherPhrase = os.Getenv(fetchArg)
	default:
		return fmt.Errorf("Unable to decode secret for auth password: %s", *cipherPhrase)
	}

	return err
}

func fetchFromFile(filename string) (string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("Unable to read secret from file %s: %s", filename, err)
	}
	return string(data), nil
}
