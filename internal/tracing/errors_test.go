// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"
	"errors"
	"runtime"
	"testing"

	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"gotest.tools/v3/assert"
)

func TestFrame(t *testing.T) {
	assert.NilError(t, Frame(nil))

	_, _, baseline, _ := runtime.Caller(0)
	err := Frame(errors.New("bang"))

	frame := getFrame(err)
	assert.Equal(t, frame.File, "internal/tracing/errors_test.go")
	assert.Equal(t, frame.Line, baseline+1)
	assert.Equal(t, frame.Function, "tracing.TestFrame")
}

func TestFrame2(t *testing.T) {
	v, err := Frame2('x', nil)
	assert.Equal(t, v, 'x')
	assert.NilError(t, err)

	{
		_, _, baseline, _ := runtime.Caller(0)
		v, err := Frame2(22, errors.New("bang"))

		assert.Equal(t, v, 22)

		frame := getFrame(err)
		assert.Equal(t, frame.File, "internal/tracing/errors_test.go")
		assert.Equal(t, frame.Line, baseline+1)
		assert.Equal(t, frame.Function, "tracing.TestFrame2")
	}
}

func TestFrame3(t *testing.T) {
	a, b, err := Frame3([]byte(`gg`), true, nil)
	assert.DeepEqual(t, a, []byte(`gg`))
	assert.Equal(t, b, true)
	assert.NilError(t, err)

	{
		_, _, baseline, _ := runtime.Caller(0)
		a, b, err := Frame3(false, "asdf", errors.New("bang"))

		assert.Equal(t, a, false)
		assert.Equal(t, b, "asdf")

		frame := getFrame(err)
		assert.Equal(t, frame.File, "internal/tracing/errors_test.go")
		assert.Equal(t, frame.Line, baseline+1)
		assert.Equal(t, frame.Function, "tracing.TestFrame3")
	}
}

func TestCheck(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tracer := trace.NewTracerProvider(
		trace.WithSpanProcessor(recorder),
	).Tracer("")

	{
		_, span := tracer.Start(context.Background(), "")
		assert.Assert(t, Check(span, nil))
		span.End()

		spans := recorder.Ended()
		assert.Equal(t, len(spans), 1)
		assert.Equal(t, len(spans[0].Events()), 0, "expected no events")
	}

	{
		_, span := tracer.Start(context.Background(), "")
		_, _, baseline, _ := runtime.Caller(0)
		assert.Assert(t, !Check(span, errors.New("msg")))
		span.End()

		spans := recorder.Ended()
		assert.Equal(t, len(spans), 2)
		assert.Equal(t, len(spans[1].Events()), 1, "expected one event")

		event := spans[1].Events()[0]
		assert.Equal(t, event.Name, semconv.ExceptionEventName)

		attrs := event.Attributes
		assert.Equal(t, len(attrs), 4)
		assert.Equal(t, string(attrs[0].Key), "code.filepath")
		assert.Equal(t, string(attrs[1].Key), "code.lineno")
		assert.Equal(t, string(attrs[2].Key), "code.function")
		assert.Equal(t, string(attrs[3].Key), "exception.message")
		assert.Equal(t, attrs[0].Value.AsInterface(), "internal/tracing/errors_test.go")
		assert.Equal(t, attrs[1].Value.AsInterface(), int64(baseline+1))
		assert.Equal(t, attrs[2].Value.AsInterface(), "tracing.TestCheck")
		assert.Equal(t, attrs[3].Value.AsInterface(), "msg")
	}
}

func TestEscape(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tracer := trace.NewTracerProvider(
		trace.WithSpanProcessor(recorder),
	).Tracer("")

	{
		_, span := tracer.Start(context.Background(), "")
		assert.NilError(t, Escape(span, nil))
		span.End()

		spans := recorder.Ended()
		assert.Equal(t, len(spans), 1)
		assert.Equal(t, len(spans[0].Events()), 0, "expected no events")
	}

	{
		_, span := tracer.Start(context.Background(), "")
		expected := errors.New("somesuch")
		_, _, baseline, _ := runtime.Caller(0)
		assert.Assert(t, errors.Is(Escape(span, expected), expected),
			"expected to unwrap the original error")
		span.End()

		spans := recorder.Ended()
		assert.Equal(t, len(spans), 2)
		assert.Equal(t, len(spans[1].Events()), 1, "expected one event")

		event := spans[1].Events()[0]
		assert.Equal(t, event.Name, semconv.ExceptionEventName)

		attrs := event.Attributes
		assert.Equal(t, len(attrs), 5)
		assert.Equal(t, string(attrs[0].Key), "exception.escaped")
		assert.Equal(t, string(attrs[1].Key), "code.filepath")
		assert.Equal(t, string(attrs[2].Key), "code.lineno")
		assert.Equal(t, string(attrs[3].Key), "code.function")
		assert.Equal(t, string(attrs[4].Key), "exception.message")
		assert.Equal(t, attrs[0].Value.AsInterface(), true)
		assert.Equal(t, attrs[1].Value.AsInterface(), "internal/tracing/errors_test.go")
		assert.Equal(t, attrs[2].Value.AsInterface(), int64(baseline+1))
		assert.Equal(t, attrs[3].Value.AsInterface(), "tracing.TestEscape")
		assert.Equal(t, attrs[4].Value.AsInterface(), "somesuch")
	}
}
