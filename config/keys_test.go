// Copyright The Notary Project Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/notaryproject/notation-core-go/testhelper"
	"github.com/notaryproject/notation-go/dir"
)

var sampleSigningKeysInfo = SigningKeys{
	Default: Ptr("wabbit-networks"),
	Keys: []KeySuite{
		{
			Name: "wabbit-networks",
			X509KeyPair: &X509KeyPair{
				KeyPath:         "/home/demo/.config/notation/localkeys/wabbit-networks.key",
				CertificatePath: "/home/demo/.config/notation/localkeys/wabbit-networks.crt",
			},
		},
		{
			Name: "import.acme-rockets",
			X509KeyPair: &X509KeyPair{
				KeyPath:         "/home/demo/.config/notation/localkeys/import.acme-rockets.key",
				CertificatePath: "/home/demo/.config/notation/localkeys/import.acme-rockets.crt",
			},
		},
		{
			Name: "external-key",
			ExternalKey: &ExternalKey{

				ID:         "id1",
				PluginName: "pluginX",
				PluginConfig: map[string]string{
					"key": "value",
				},
			},
		},
	},
}

func TestLoadSigningKeysInfo(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		dir.UserConfigDir = "./testdata/valid"
		got, err := LoadSigningKeys()
		if err != nil {
			t.Errorf("LoadSigningKeysInfo() error = \"%v\"", err)
			return
		}

		if !reflect.DeepEqual(sampleSigningKeysInfo.Default, got.Default) {
			t.Fatal("signingKeysInfo test failed.")
		}

		if !reflect.DeepEqual(sampleSigningKeysInfo.Keys, got.Keys) {
			t.Fatal("signingKeysInfo test failed.")
		}
	})

	t.Run("DuplicateKeys", func(t *testing.T) {
		expectedErr := "malformed signingkeys.json: multiple keys with name 'wabbit-networks' found"
		dir.UserConfigDir = "./testdata/malformed-duplicate"
		_, err := LoadSigningKeys()
		if err == nil || err.Error() != expectedErr {
			t.Errorf("LoadSigningKeysInfo() error expected = \"%v\" but found = \"%v\"", expectedErr, err)
		}
	})

	t.Run("InvalidDefault", func(t *testing.T) {
		expectedErr := "malformed signingkeys.json: default key 'missing-default' not found"
		dir.UserConfigDir = "./testdata/malformed-invalid-default"
		_, err := LoadSigningKeys()
		if err == nil || err.Error() != expectedErr {
			t.Errorf("LoadSigningKeysInfo() error expected = \"%v\" but found = \"%v\"", expectedErr, err)
		}
	})
}

func TestSaveSigningKeys(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		root := t.TempDir()
		dir.UserConfigDir = root
		sampleSigningKeysInfo.Save()
		info, err := LoadSigningKeys()
		if err != nil {
			t.Fatal("Load signingkeys.json from temp dir failed.")
		}

		if !reflect.DeepEqual(sampleSigningKeysInfo.Default, info.Default) {
			t.Fatal("Save signingkeys.json failed.")
		}

		if !reflect.DeepEqual(sampleSigningKeysInfo.Keys, info.Keys) {
			t.Fatal("Save signingkeys.json failed.")
		}
	})

	t.Run("ValidWithoutDefault", func(t *testing.T) {
		root := t.TempDir()
		dir.UserConfigDir = root
		sampleSigningKeysInfoNoDefault := deepCopySigningKeys(sampleSigningKeysInfo)
		sampleSigningKeysInfoNoDefault.Default = nil
		sampleSigningKeysInfoNoDefault.Save()
		info, err := LoadSigningKeys()
		if err != nil {
			t.Fatal("Load signingkeys.json from temp dir failed.")
		}

		if !reflect.DeepEqual(sampleSigningKeysInfoNoDefault.Default, info.Default) {
			t.Fatal("Save signingkeys.json failed.")
		}

		if !reflect.DeepEqual(sampleSigningKeysInfoNoDefault.Keys, info.Keys) {
			t.Fatal("Save signingkeys.json failed.")
		}
	})

	t.Run("DuplicateKeys", func(t *testing.T) {
		expectedErr := "malformed signingkeys.json: multiple keys with name 'import.acme-rockets' found"
		dir.UserConfigDir = t.TempDir()
		duplicateKeySignKeysInfo := deepCopySigningKeys(sampleSigningKeysInfo)
		duplicateKeySignKeysInfo.Keys = append(duplicateKeySignKeysInfo.Keys, KeySuite{
			Name: "import.acme-rockets",
			X509KeyPair: &X509KeyPair{
				KeyPath:         "/keypath",
				CertificatePath: "/CertificatePath",
			},
		})
		err := duplicateKeySignKeysInfo.Save()
		if err == nil || err.Error() != expectedErr {
			t.Errorf("Save signingkeys.json failed, error expected = \"%v\" but found = \"%v\"", expectedErr, err)
		}
	})

	t.Run("EmptyKeyName", func(t *testing.T) {
		expectedErr := "malformed signingkeys.json: key name cannot be empty"
		dir.UserConfigDir = t.TempDir()
		emptyKeyNameSignKeysInfo := deepCopySigningKeys(sampleSigningKeysInfo)
		emptyKeyNameSignKeysInfo.Keys[0].Name = ""

		err := emptyKeyNameSignKeysInfo.Save()
		if err == nil || err.Error() != expectedErr {
			t.Errorf("Save signingkeys.json failed, error expected = \"%v\" but found = \"%v\"", expectedErr, err)
		}
	})

	t.Run("InvalidDefault", func(t *testing.T) {
		expectedErr := "malformed signingkeys.json: default key 'missing-default' not found"
		dir.UserConfigDir = t.TempDir()
		invalidDefaultSignKeysInfo := deepCopySigningKeys(sampleSigningKeysInfo)
		invalidDefaultSignKeysInfo.Default = Ptr("missing-default")
		err := invalidDefaultSignKeysInfo.Save()
		if err == nil || err.Error() != expectedErr {
			t.Errorf("Save signingkeys.json failed, error expected = \"%v\" but found = \"%v\"", expectedErr, err)
		}

		expectedErr = "malformed signingkeys.json: default key name cannot be empty"
		invalidDefaultSignKeysInfo.Default = Ptr("")
		err = invalidDefaultSignKeysInfo.Save()
		if err == nil || err.Error() != expectedErr {
			t.Errorf("Save signingkeys.json failed, error expected = \"%v\" but found = \"%v\"", expectedErr, err)
		}
	})
}

func TestAdd(t *testing.T) {
	certPath, keyPath := createTempCertKey(t)
	t.Run("WithDefault", func(t *testing.T) {
		testSigningKeys := deepCopySigningKeys(sampleSigningKeysInfo)
		expectedTestKeyName := "name1"

		if err := testSigningKeys.Add(expectedTestKeyName, keyPath, certPath, true); err != nil {
			t.Errorf("Add() failed with err= %v", err)
		}

		expectedSigningKeys := append(deepCopySigningKeys(sampleSigningKeysInfo).Keys, KeySuite{
			Name: expectedTestKeyName,
			X509KeyPair: &X509KeyPair{
				KeyPath:         keyPath,
				CertificatePath: certPath,
			},
		})

		if expectedTestKeyName != *testSigningKeys.Default {
			t.Error("Add() failed, incorrect default key")
		}
		if !reflect.DeepEqual(testSigningKeys.Keys, expectedSigningKeys) {
			t.Error("Add() failed, KeySuite mismatch")
		}
	})

	t.Run("WithoutDefault", func(t *testing.T) {
		dir.UserConfigDir = t.TempDir()

		testSigningKeys := deepCopySigningKeys(sampleSigningKeysInfo)
		expectedTestKeyName := "name2"
		certPath, keyPath := createTempCertKey(t)
		if err := testSigningKeys.Add(expectedTestKeyName, keyPath, certPath, false); err != nil {
			t.Errorf("Add() failed with err= %v", err)
		}

		expectedSigningKeys := append(deepCopySigningKeys(sampleSigningKeysInfo).Keys, KeySuite{
			Name: expectedTestKeyName,
			X509KeyPair: &X509KeyPair{
				KeyPath:         keyPath,
				CertificatePath: certPath,
			},
		})

		if *sampleSigningKeysInfo.Default != *testSigningKeys.Default {
			t.Error("Add() failed, default key changed")
		}
		if !reflect.DeepEqual(testSigningKeys.Keys, expectedSigningKeys) {
			t.Error("Add() failed, KeySuite mismatch")
		}
	})

	t.Run("InvalidCertKeyLocation", func(t *testing.T) {
		err := sampleSigningKeysInfo.Add("name1", "invalid", "invalid", true)
		if err == nil {
			t.Error("expected Add() to fail for invalid cert and key location")
		}
	})

	t.Run("InvalidName", func(t *testing.T) {
		err := sampleSigningKeysInfo.Add("", "invalid", "invalid", true)
		if err == nil {
			t.Error("expected Add() to fail for empty key name")
		}
	})

	t.Run("InvalidName", func(t *testing.T) {
		err := sampleSigningKeysInfo.Add("", "invalid", "invalid", true)
		if err == nil {
			t.Error("expected Add() to fail for empty key name")
		}
	})

	t.Run("DuplicateKey", func(t *testing.T) {
		err := sampleSigningKeysInfo.Add(sampleSigningKeysInfo.Keys[0].Name, "invalid", "invalid", true)
		if err == nil {
			t.Error("expected Add() to fail for duplicate name")
		}
	})
}

func TestPluginAdd(t *testing.T) {
	config := map[string]string{"key1": "value1"}
	name := "name1"
	id := "pluginId1"
	pluginName := "pluginName1"

	t.Run("InvalidCertKeyLocation", func(t *testing.T) {
		err := sampleSigningKeysInfo.Add("name1", "invalid", "invalid", true)
		if err == nil {
			t.Error("expected AddPlugin() to fail for invalid cert and key location")
		}
	})

	t.Run("InvalidName", func(t *testing.T) {
		err := sampleSigningKeysInfo.AddPlugin(context.Background(), "", id, pluginName, config, true)
		if err == nil {
			t.Error("expected AddPlugin() to fail for empty key name")
		}
	})

	t.Run("InvalidId", func(t *testing.T) {
		err := sampleSigningKeysInfo.AddPlugin(context.Background(), name, "", pluginName, config, true)
		if err == nil {
			t.Error("expected AddPlugin() to fail for empty key name")
		}
	})

	t.Run("InvalidPluginName", func(t *testing.T) {
		err := sampleSigningKeysInfo.AddPlugin(context.Background(), name, id, "", config, true)
		if err == nil {
			t.Error("AddPlugin AddPlugin() to fail for empty plugin name")
		}
	})
}

func TestGet(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		key, err := sampleSigningKeysInfo.Get("external-key")
		if err != nil {
			t.Errorf("Get() failed with error= %v", err)
		}

		if !reflect.DeepEqual(key, sampleSigningKeysInfo.Keys[2]) {
			t.Errorf("Get() returned %v but expected %v", key, sampleSigningKeysInfo.Keys[2])
		}
	})

	t.Run("NonExistent", func(t *testing.T) {
		_, err := sampleSigningKeysInfo.Get("nonExistent")
		if err == nil {
			t.Error("expected Get() to fail for nonExistent key name")
		}
		if !errors.Is(err, KeyNotFoundError{KeyName: "nonExistent"}) {
			t.Error("expected Get() to return ErrorKeyNotFound")
		}
	})

	t.Run("EmptyName", func(t *testing.T) {
		_, err := sampleSigningKeysInfo.Get("")
		if err == nil {
			t.Error("expected Get() to fail for empty key name")
		}
		if !errors.Is(err, ErrKeyNameEmpty) {
			t.Error("expected Get() to return ErrorKeyNameEmpty")
		}
	})
}

func TestGetDefault(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		key, err := sampleSigningKeysInfo.GetDefault()
		if err != nil {
			t.Errorf("GetDefault() failed with error= %v", err)
		}

		if !reflect.DeepEqual(key.Name, *sampleSigningKeysInfo.Default) {
			t.Errorf("GetDefault() returned %s but expected %s", key.Name, *sampleSigningKeysInfo.Default)
		}
	})

	t.Run("NoDefault", func(t *testing.T) {
		testSigningKeysInfo := deepCopySigningKeys(sampleSigningKeysInfo)
		testSigningKeysInfo.Default = nil
		if _, err := testSigningKeysInfo.GetDefault(); err == nil {
			t.Error("GetDefault Get() to fail there is no defualt key")
		}
	})
}

func TestUpdateDefault(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		testSigningKeysInfo := deepCopySigningKeys(sampleSigningKeysInfo)
		newDefault := sampleSigningKeysInfo.Keys[1].Name
		err := testSigningKeysInfo.UpdateDefault(newDefault)
		if err != nil {
			t.Errorf("UpdateDefault() failed with error= %v", err)
		}

		if !reflect.DeepEqual(newDefault, *testSigningKeysInfo.Default) {
			t.Errorf("UpdateDefault() didn't update default key")
		}
	})

	t.Run("NonExistent", func(t *testing.T) {
		err := sampleSigningKeysInfo.UpdateDefault("nonExistent")
		if err == nil {
			t.Error("expected Get() to fail for nonExistent key name")
		}
		if !errors.Is(err, KeyNotFoundError{KeyName: "nonExistent"}) {
			t.Error("expected Get() to return ErrorKeyNotFound")
		}
	})

	t.Run("EmptyName", func(t *testing.T) {
		err := sampleSigningKeysInfo.UpdateDefault("")
		if err == nil {
			t.Error("expected Get() to fail for empty key name")
		}
		if !errors.Is(err, ErrKeyNameEmpty) {
			t.Error("expected Get() to return ErrorKeyNameEmpty")
		}
	})
}

func TestRemove(t *testing.T) {
	testKeyName := "wabbit-networks"
	testSigningKeysInfo := deepCopySigningKeys(sampleSigningKeysInfo)
	t.Run("Valid", func(t *testing.T) {
		keys, err := testSigningKeysInfo.Remove(testKeyName)
		if err != nil {
			t.Errorf("testSigningKeysInfo() failed with error= %v", err)
		}

		if _, err := testSigningKeysInfo.Get(testKeyName); err == nil {
			t.Error("Delete() filed to delete key")
		}
		if keys[0] != testKeyName {
			t.Error("Delete() deleted key name mismatch")
		}
	})

	t.Run("NonExistent", func(t *testing.T) {
		_, err := testSigningKeysInfo.Remove("nonExistent")
		if err == nil {
			t.Error("expected Get() to fail for nonExistent key name")
		}
		if !errors.Is(err, KeyNotFoundError{KeyName: "nonExistent"}) {
			t.Error("expected Get() to return ErrorKeyNotFound")
		}
	})

	t.Run("EmptyName", func(t *testing.T) {
		_, err := testSigningKeysInfo.Remove("")
		if err == nil {
			t.Error("expected Get() to fail for empty key name")
		}
		if !errors.Is(err, ErrKeyNameEmpty) {
			t.Error("expected Get() to return ErrorKeyNameEmpty")
		}
	})
}

func deepCopySigningKeys(keys SigningKeys) SigningKeys {
	cpyKeys := make([]KeySuite, len(sampleSigningKeysInfo.Keys))
	copy(cpyKeys, keys.Keys)
	cpyDefault := *keys.Default
	cpySignKeys := keys
	cpySignKeys.Default = &cpyDefault
	cpySignKeys.Keys = cpyKeys
	return cpySignKeys
}

func Ptr[T any](v T) *T {
	return &v
}

func createTempCertKey(t *testing.T) (string, string) {
	certTuple := testhelper.GetRSARootCertificate()
	certPath := filepath.Join(t.TempDir(), "cert.tmp")
	certData := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certTuple.Cert.Raw})
	if err := os.WriteFile(certPath, certData, 0600); err != nil {
		panic(err)
	}
	keyPath := filepath.Join(t.TempDir(), "key.tmp")
	keyBytes, _ := x509.MarshalPKCS8PrivateKey(certTuple.PrivateKey)
	keyData := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})
	if err := os.WriteFile(keyPath, keyData, 0600); err != nil {
		panic(err)
	}
	return certPath, keyPath
}
