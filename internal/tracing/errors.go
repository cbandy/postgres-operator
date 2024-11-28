// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"errors"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"braces.dev/errtrace"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
)

// An error implementing `TracePC() uintptr` contains stack trace information.
// See: [errtrace.UnwrapFrame]
type errtraceProgramCounter interface{ TracePC() uintptr }

var moduleDirectory string

func init() {
	// Calculate the directory of main module reported by [runtime].
	_, moduleDirectory, _, _ = runtime.Caller(0)
	moduleDirectory = strings.TrimSuffix(moduleDirectory,
		filepath.Join("internal", "tracing", "errors.go"))
}

// getFrame returns a [runtime.Frame] corresponding to program information, if any, attached to err.
func getFrame(err error) runtime.Frame {
	var frame runtime.Frame

	if et, ok := err.(errtraceProgramCounter); ok || errors.As(err, &et) {
		frame, _ = runtime.CallersFrames([]uintptr{et.TracePC()}).Next()
	}

	// Remove the path to the directory of the main module; it is just noise.
	frame.File = strings.TrimPrefix(frame.File, moduleDirectory)

	// Remove the path leading up to the package and function name.
	_, frame.Function = path.Split(frame.Function)

	return frame
}

// Check returns true when err is nil. Otherwise, it adds err as an exception
// event on s and returns false. If you intend to return err, consider using
// [Escape] instead.
//
// See: https://opentelemetry.io/docs/specs/semconv/exceptions/exceptions-spans
//
//go:noinline
func Check(s Span, err error) bool {
	if err == nil {
		return true
	}
	if s.IsRecording() {
		if _, ok := err.(errtraceProgramCounter); !ok {
			err = errtrace.GetCaller().Wrap(err)
		}
		addError(s, err)
	}
	return false
}

// Escape adds non-nil err as an escaped exception event on s and returns err.
//
// See: https://opentelemetry.io/docs/specs/semconv/exceptions/exceptions-spans
//
//go:noinline
func Escape(s Span, err error) error {
	if err != nil && s.IsRecording() {
		if _, ok := err.(errtraceProgramCounter); !ok {
			err = errtrace.GetCaller().Wrap(err)
		}
		addError(s, err, semconv.ExceptionEscaped(true))
	}
	return err
}

// addError adds err to s as an exception event with attrs.
//
// When the error includes a file, line, or function name, those are added to the event as code
// attributes. This is similar to [Span.RecordError] but does not bother with the error type,
// which is often [fmt.wrapError] or [errors.errorString].
func addError(s Span, err error, attrs ...attribute.KeyValue) {
	frame := getFrame(err)

	if frame.File != "" {
		attrs = append(attrs,
			semconv.CodeFilepath(frame.File),
			semconv.CodeLineNumber(frame.Line),
		)
	}
	if frame.Function != "" {
		attrs = append(attrs,
			semconv.CodeFunction(frame.Function),
		)
	}

	s.AddEvent(semconv.ExceptionEventName, trace.WithAttributes(append(attrs,
		semconv.ExceptionMessage(err.Error()),
	)...))
}

// Frame adds information about the caller to err. This information will appear
// in logs and in spans passed to [Check] or [Escape].
//
//go:noinline
func Frame(err error) error {
	if err != nil {
		err = errtrace.GetCaller().Wrap(err)
	}
	return err
}

// Frame2 is the same as [Frame] for a function that returns two results.
//
//go:noinline
func Frame2[T any](v T, err error) (T, error) {
	if err != nil {
		err = errtrace.GetCaller().Wrap(err)
	}
	return v, err
}

// Frame3 is the same as [Frame] for a function that returns three results.
//
//go:noinline
func Frame3[T1, T2 any](v1 T1, v2 T2, err error) (T1, T2, error) {
	if err != nil {
		err = errtrace.GetCaller().Wrap(err)
	}
	return v1, v2, err
}
