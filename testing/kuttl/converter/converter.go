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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func main() {
	for _, filename := range os.Args[1:] {
		fmt.Printf("converting %v …", filename)

		if err := Convert(filename); err != nil {
			fmt.Fprintf(os.Stderr, "unable to convert %v: %+v\n", filename, err)
			os.Exit(1)
		}

		fmt.Println("\tok")
	}
}

// Convert reads the YAML filename and writes its documents as a KUTTL test case.
func Convert(filename string) error {
	dir := strings.TrimSuffix(filename, filepath.Ext(filename)) + "-generated"

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	documents := 0
	decoder := yaml.NewYAMLToJSONDecoder(file)
	writer := Writer{Directory: dir}

	if err = os.RemoveAll(dir); err == nil {
		err = os.MkdirAll(dir, 0o755)
	}

	for err == nil {
		var u unstructured.Unstructured
		if err = decoder.Decode(&u); errors.Is(err, io.EOF) {
			return nil
		}
		if err == nil {
			documents++
			err = writer.Write(documents, u)
		}
	}

	return err
}

// TestingStep represents everything about a single KUTTL test step.
type TestingStep struct {
	// Timeout is the maximum amount of time to wait for this step's assertions.
	// It can be an integral number of seconds or a time.Duration string.
	Timeout *intstr.IntOrString `json:"timeout"`

	Resources []unstructured.Unstructured `json:"resources"`
	Asserts   []unstructured.Unstructured `json:"asserts"`
	Errors    []unstructured.Unstructured `json:"errors"`

	Files struct {
		Resources []string `json:"resources"`
		Asserts   []string `json:"asserts"`
		Errors    []string `json:"errors"`
	} `json:"files"`
}

type Writer struct {
	Directory string
}

func (w Writer) Write(index int, object unstructured.Unstructured) error {
	gvk := schema.FromAPIVersionAndKind(object.GetAPIVersion(), object.GetKind())

	switch gvk {
	case schema.GroupVersionKind{Group: "testing", Version: "v1", Kind: "Step"}:
		return w.writeTestingStep(index, object)

	default:
		filename := fmt.Sprintf("%02d-%s.yaml", index, gvk.Kind)
		return w.writeFile(filename, object)
	}
}

// writeFile writes objects as a series of YAML documents to a file name.
func (w Writer) writeFile(name string, objects ...unstructured.Unstructured) error {
	var buffer bytes.Buffer

	for i := range objects {
		// TODO: import another YAML package to marshal.
		b, err := objects[i].MarshalJSON()
		if err == nil {
			_, _ = buffer.WriteString("---\n")
			err = json.Indent(&buffer, b, "", "  ")
		}
		if err != nil {
			return err
		}
	}

	return os.WriteFile(filepath.Join(w.Directory, name), buffer.Bytes(), 0o644)
}

func (w Writer) writeTestingStep(index int, object unstructured.Unstructured) error {
	var step TestingStep
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &step)

	anyFiles := !reflect.ValueOf(step.Files).IsZero()
	asserts := step.Asserts
	resources := step.Resources

	if err == nil && anyFiles {
		// https://github.com/kudobuilder/kuttl/blob/main/keps/0004-test-composability.md
		ts := map[string]interface{}{"apiVersion": "kuttl.dev/v1beta1", "kind": "TestStep"}

		if len(step.Files.Resources) > 0 {
			ts["apply"] = step.Files.Resources
		}
		if len(step.Files.Asserts) > 0 {
			ts["assert"] = step.Files.Asserts
		}
		if len(step.Files.Errors) > 0 {
			ts["error"] = step.Files.Errors
		}

		resources = append(resources, unstructured.Unstructured{Object: ts})
	}

	if err == nil && step.Timeout != nil {
		// https://kuttl.dev/docs/testing/reference.html#testassert
		ta := map[string]interface{}{"apiVersion": "kuttl.dev/v1beta1", "kind": "TestAssert"}

		if step.Timeout.StrVal != "" {
			var d time.Duration
			if d, err = time.ParseDuration(step.Timeout.StrVal); err == nil {
				if d < time.Second {
					d = time.Second
				} else {
					d = d.Round(time.Second)
				}
				ta["timeout"] = d / time.Second
			}
		} else {
			ta["timeout"] = step.Timeout.IntVal
		}

		asserts = append([]unstructured.Unstructured{{Object: ta}}, asserts...)
	}

	if err == nil && len(asserts) > 0 {
		// https://kuttl.dev/docs/testing/asserts-errors.html
		filename := fmt.Sprintf("%02d-assert.yaml", index)
		err = w.writeFile(filename, asserts...)
	}

	if err == nil && len(step.Errors) > 0 {
		// https://kuttl.dev/docs/testing/asserts-errors.html
		filename := fmt.Sprintf("%02d-errors.yaml", index)
		err = w.writeFile(filename, step.Errors...)
	}

	if err == nil && len(resources) > 0 {
		filename := fmt.Sprintf("%02d-Step.yaml", index)
		err = w.writeFile(filename, resources...)
	}

	return err
}
