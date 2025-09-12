// Copyright 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

//go:build generate

//go:generate go run post-process.go

package main

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/itchyny/gojq"
	"sigs.k8s.io/yaml"
)

func main() {
	cwd := need(os.Getwd())
	dir := filepath.Join(cwd, "..", "..", "config", "crd", "bases")
	query := "post-process.jq"
	fileNames := map[string][]string{}

	slog.Info("Reading", "file", query)
	code := need(gojq.Compile(
		need(gojq.Parse(string(need(os.ReadFile(query))))),
		gojq.WithModuleLoader(gojq.NewModuleLoader([]string{"$ORIGIN/../lib"})),
		gojq.WithEnvironLoader(os.Environ),
	))

	slog.Info("Reading", "directory", dir)
	for _, entry := range need(os.ReadDir(dir)) {
		if entry.Type() == 0 {
			ext := filepath.Ext(entry.Name())
			fileNames[ext] = append(fileNames[ext], entry.Name())
		}
	}

	for _, yamlName := range fileNames[".yaml"] {
		slog.Info("Reading", "file", yamlName)
		yamlPath := filepath.Join(dir, yamlName)
		yamlValue := any(nil)

		must(yaml.UnmarshalStrict(need(os.ReadFile(yamlPath)), &yamlValue))

		result := code.Run(yamlValue)
		if v, ok := result.Next(); ok {
			if err, ok := v.(error); ok {
				panic(err)
			}

			slog.Info("Writing", "file", yamlName)
			must(os.WriteFile(yamlPath, append([]byte("---\n"), need(yaml.Marshal(v))...), 0o644))
		}

		if _, ok := result.Next(); ok {
			panic("unexpected second result")
		}
	}
}

func must(err error) { need(0, err) }
func need[V any](v V, err error) V {
	if err != nil {
		panic(err)
	}
	return v
}
