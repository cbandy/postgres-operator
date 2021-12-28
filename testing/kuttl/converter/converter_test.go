/*
 Copyright 2021 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestParsing(t *testing.T) {
	file, err := os.Open(filepath.Join("testdata", "single.yaml"))
	assert.NilError(t, err)
	defer file.Close()

	decoder := yaml.NewYAMLToJSONDecoder(file)

	{
		var u unstructured.Unstructured
		assert.NilError(t, decoder.Decode(&u))

		assert.Equal(t, u.GetAPIVersion(), "v1")
		assert.Equal(t, u.GetKind(), "ConfigMap")
	}

	{
		var u unstructured.Unstructured
		assert.NilError(t, decoder.Decode(&u))

		assert.Equal(t, u.GetAPIVersion(), "testing/v1")
		assert.Equal(t, u.GetKind(), "Step")
	}

	{
		var u unstructured.Unstructured
		err := decoder.Decode(&u)

		assert.Assert(t, errors.Is(err, io.EOF), "expected EOF, got %+v", err)
	}
}

func TestWriting(t *testing.T) {
	writer := Writer{Directory: filepath.Join("testdata", "single-generated")}

	file, err := os.Open(filepath.Join("testdata", "single.yaml"))
	assert.NilError(t, err)
	defer file.Close()

	decoder := yaml.NewYAMLToJSONDecoder(file)

	{
		var u unstructured.Unstructured
		assert.NilError(t, decoder.Decode(&u))
		assert.NilError(t, writer.Write(1, u))
	}

	{
		var u unstructured.Unstructured
		assert.NilError(t, decoder.Decode(&u))
		assert.NilError(t, writer.Write(2, u))
	}
}

func TestConversion(t *testing.T) {
	source := filepath.Join("testdata", "single.yaml")
	target := filepath.Join("testdata", "single-generated")
	expected := filepath.Join("testdata", "single-expected")

	t.Run("NoTargetDirectory", func(t *testing.T) {
		assert.NilError(t, os.RemoveAll(target))

		assert.NilError(t, Convert(source))

		assert.Assert(t, fs.Equal(target, fs.ManifestFromDir(t, expected)))
	})

	t.Run("ExtraFiles", func(t *testing.T) {
		assert.NilError(t, os.MkdirAll(target, 0o755))
		assert.NilError(t, os.WriteFile(filepath.Join(target, "99-extra.yaml"), nil, 0o644))

		assert.NilError(t, Convert(source))

		assert.Assert(t, fs.Equal(target, fs.ManifestFromDir(t, expected)))
	})
}
