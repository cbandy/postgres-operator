/*
 Copyright 2022 Crunchy Data Solutions, Inc.
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

package events

import (
	"fmt"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/duration"
)

// Format returns event in a format similar to `kubectl describe`.
func Format(event corev1.Event) string {
	source := event.Source.Component
	if source == "" {
		source = event.ReportingController
	}

	timestamp := event.EventTime.Time
	if timestamp.IsZero() {
		timestamp = event.FirstTimestamp.Time
	}

	interval := duration.HumanDuration(time.Since(timestamp))
	if event.Series != nil {
		interval = fmt.Sprintf("%s (x%d over %s)",
			duration.HumanDuration(time.Since(event.Series.LastObservedTime.Time)),
			event.Series.Count, interval)
	} else if event.Count > 1 {
		interval = fmt.Sprintf("%s (x%d over %s)",
			duration.HumanDuration(time.Since(event.LastTimestamp.Time)),
			event.Count, interval)
	}

	return fmt.Sprintf("%s\t%-8s\t%s\t%-8s\t%s",
		event.Type, event.Reason, interval, source, event.Message)
}

// FormatList returns events formatted and separated by newlines.
func FormatList(events []corev1.Event) string {
	var buffer strings.Builder

	if len(events) > 0 {
		_, _ = buffer.WriteString(Format(events[0]))

		for _, event := range events[1:] {
			_, _ = buffer.WriteString("\n" + Format(event))
		}
	}

	return buffer.String()
}

// Since returns events that occurred after t.
func Since(events []corev1.Event, t time.Time) []corev1.Event {
	var result []corev1.Event

	for _, event := range events {
		if event.EventTime.After(t) || event.FirstTimestamp.After(t) {
			result = append(result, event)
		} else if event.Series != nil && event.Series.LastObservedTime.After(t) {
			result = append(result, event)
		} else if event.Count > 1 && event.LastTimestamp.After(t) {
			result = append(result, event)
		}
	}

	return result
}

// SortByTimestamp sorts events by LastTimestamp, oldest first.
func SortByTimestamp(events []corev1.Event) {
	sort.Slice(events, func(i, j int) bool {
		return events[i].LastTimestamp.Time.Before(events[j].LastTimestamp.Time)
	})
}
