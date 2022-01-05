// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package plugin_data

import (
	"fmt"
	"strings"
)

func SetSecret(cipherPhrase *string) error {
	data := strings.SplitN(*cipherPhrase, ":", 2)
	if data == nil { // Just plain text, nothing to do
		return nil
	}
	switch data[0] {
	case "kube_file": // Look up secret according to file path eg Kubernetes secrets
	default:
		return fmt.Errorf("Unable to decode secret for auth password: %s", *cipherPhrase)
	}
	return nil
}
