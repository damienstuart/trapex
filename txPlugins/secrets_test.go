// Copyright (c) 2022 Kells Kearney. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package pluginMeta

import (
	"testing"

	"github.com/rs/zerolog"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
}

func TestFileSecret(t *testing.T) {
	actualSecret := "filename:../tests/secret_data.txt"
	expectedSecret := "test secret data"

	plaintext, err := GetSecret(actualSecret)
	if err != nil {
		t.Errorf("Unable to read secret file: %s", err)
	}
	if plaintext != expectedSecret {
		t.Errorf("Secret file data: %s != %s", plaintext, expectedSecret)
	}
}
