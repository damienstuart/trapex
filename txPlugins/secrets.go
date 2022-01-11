// Copyright (c) 2022 Kells Kearney. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package pluginMeta

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func GetSecret(cipherPhrase string) (string, error) {
	splits := strings.SplitN(cipherPhrase, ":", 2)
	if splits == nil || len(splits) != 2 { // Just plain text, nothing to do
		return cipherPhrase, nil
	}
	var err error
	var plaintext string

	fetchMethod := splits[0]
	fetchArg := splits[1]

	switch fetchMethod {
	case "filename": // Look up secret according to file path eg Kubernetes secrets
		plaintext, err = fetchFromFile(fetchArg)
	case "env":
		plaintext = os.Getenv(fetchArg)
	default:
		return "", fmt.Errorf("Unable to decode secret for %s password: %s", fetchMethod, fetchArg)
	}

	return plaintext, err
}

func fetchFromFile(filename string) (string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("Unable to read secret from file %s: %s", filename, err)
	}
	return strings.TrimSuffix(string(data), "\n"), nil
}
