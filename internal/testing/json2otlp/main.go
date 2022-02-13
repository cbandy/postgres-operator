package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	ctx := context.Background()

	// https://opentelemetry.io/docs/reference/specification/sdk-environment-variables
	if os.Getenv("OTEL_SERVICE_NAME") == "" {
		os.Setenv("OTEL_SERVICE_NAME", "go test")
	}

	// https://github.com/gotestyourself/gotestsum#post-run-command
	file, err := os.Open(os.Getenv("GOTESTSUM_JSONFILE"))
	if err != nil {
		log.Fatalf("[GOTESTSUM_JSONFILE=%q] %v", os.Getenv("GOTESTSUM_JSONFILE"), err)
	}
	defer file.Close()

	exporter, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer exporter.Shutdown(ctx)

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.Default()),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	defer provider.Shutdown(ctx)

	tracer := provider.Tracer("json2otlp")

	var (
		packages = make(map[string]trace.Span)
		tests    = make(map[string]map[string]trace.Span)

		last  time.Time
		total trace.Span
	)

	for decoder, line := json.NewDecoder(file), 1; decoder.More(); line++ {
		// https://pkg.go.dev/cmd/test2json
		var event struct {
			Action, Output string
			Package, Test  string
			Time           time.Time
		}

		if err := decoder.Decode(&event); err != nil {
			log.Printf("unable to decode line %v: %v", line, err)
			continue
		}

		// Keep track of the last event timestamp.
		last = event.Time

		// Start a span for the entire suite from the very first event.
		if total == nil {
			_, total = tracer.Start(ctx, "json2otlp", trace.WithTimestamp(event.Time))
		}

		// Start a span for this package.
		if packages[event.Package] == nil {
			_, packages[event.Package] = tracer.Start(
				trace.ContextWithSpan(ctx, total), event.Package,
				trace.WithTimestamp(event.Time))

			tests[event.Package] = make(map[string]trace.Span)
		}

		switch event.Action {
		case "run":
			if event.Test == "" {
				log.Printf("missing Test on line %v", line)
				continue
			}

			// Start a span for this test.
			_, tests[event.Package][event.Test] = tracer.Start(
				trace.ContextWithSpan(ctx, packages[event.Package]), event.Test,
				trace.WithTimestamp(event.Time),
				trace.WithAttributes(
					attribute.String("code.namespace", event.Package),
					attribute.String("code.function", event.Test),
				))

		case "pause", "cont":
			if event.Test == "" {
				log.Printf("missing Test on line %v", line)
				continue
			}

			span := tests[event.Package][event.Test]
			span.AddEvent(event.Action, trace.WithTimestamp(event.Time))

		case "pass", "skip":
			span := packages[event.Package]
			if event.Test != "" {
				span = tests[event.Package][event.Test]
			}
			if event.Action == "pass" {
				span.SetStatus(codes.Ok, "")
			}
			span.End(trace.WithTimestamp(event.Time))

		case "fail":
			span := packages[event.Package]
			if event.Test != "" {
				span = tests[event.Package][event.Test]
			}
			span.SetStatus(codes.Error, event.Output)
			span.End(trace.WithTimestamp(event.Time))
		}
	}

	// Finish the span for the entire suite from the very last event.
	if total != nil {
		total.End(trace.WithTimestamp(last))
	}

	log.Print("TraceID: ", total.SpanContext().TraceID())
}
