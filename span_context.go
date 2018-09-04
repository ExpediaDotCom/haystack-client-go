/*
 *  Copyright 2018 Expedia Group.
 *
 *     Licensed under the Apache License, Version 2.0 (the "License");
 *     you may not use this file except in compliance with the License.
 *     You may obtain a copy of the License at
 *
 *         http://www.apache.org/licenses/LICENSE-2.0
 *
 *     Unless required by applicable law or agreed to in writing, software
 *     distributed under the License is distributed on an "AS IS" BASIS,
 *     WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *     See the License for the specific language governing permissions and
 *     limitations under the License.
 *
 */

package haystack

import (
	"fmt"
)

/*SpanContext implements opentracing.spanContext*/
type SpanContext struct {
	// traceID represents globally unique ID of the trace.
	// Usually generated as a random number.
	traceID string

	// spanID represents span ID that must be unique within its trace,
	// but does not have to be globally unique.
	spanID string

	// parentID refers to the ID of the parent span.
	// Should be 0 if the current span is a root span.
	parentID string

	// Distributed Context baggage. The is a snapshot in time.
	baggage map[string]string
}

// IsValid indicates whether this context actually represents a valid trace.
func (context SpanContext) IsValid() bool {
	return context.traceID != "" && context.spanID != ""
}

/*ForeachBaggageItem implements opentracing.spancontext*/
func (context SpanContext) ForeachBaggageItem(handler func(k, v string) bool) {
	for k, v := range context.baggage {
		if !handler(k, v) {
			break
		}
	}
}

// TraceID returns the trace ID of this span context
func (context SpanContext) TraceID() string {
	return context.traceID
}

// SpanID returns the span ID of this span context
func (context SpanContext) SpanID() string {
	return context.spanID
}

// ParentID returns the parent span ID of this span context
func (context SpanContext) ParentID() string {
	return context.parentID
}

// WithBaggageItem creates a new context with an extra baggage item.
func (context SpanContext) WithBaggageItem(key, value string) *SpanContext {
	var newBaggage map[string]string
	if context.baggage == nil {
		newBaggage = map[string]string{key: value}
	} else {
		newBaggage = make(map[string]string, len(context.baggage)+1)
		for k, v := range context.baggage {
			newBaggage[k] = v
		}
		newBaggage[key] = value
	}
	return &SpanContext{
		traceID:  context.traceID,
		spanID:   context.spanID,
		parentID: context.parentID,
		baggage:  newBaggage,
	}
}

/*ToString represents the string*/
func (context SpanContext) ToString() string {
	return fmt.Sprintf("%+v", context)
}
