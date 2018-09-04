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
	"io"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/satori/go.uuid"
)

/*Tracer implements the opentracing.tracer*/
type Tracer struct {
	serviceName string
	logger      Logger
	dispatcher  Dispatcher
	commonTags  []opentracing.Tag
	timeNow     func() time.Time
	idGenerator func() string
	propagtors  map[interface{}]Propagator
}

/*NewTracer creates a new tracer*/
func NewTracer(
	serviceName string,
	dispatcher Dispatcher,
	options ...TracerOption,
) (opentracing.Tracer, io.Closer) {
	tracer := &Tracer{
		serviceName: serviceName,
		dispatcher:  dispatcher,
	}
	tracer.propagtors = make(map[interface{}]Propagator)
	tracer.propagtors[opentracing.TextMap] = NewDefaultTextMapPropagator()
	tracer.propagtors[opentracing.HTTPHeaders] = NewTextMapPropagator(PropagatorOpts{}, URLCodex{})

	for _, option := range options {
		option(tracer)
	}

	if tracer.timeNow == nil {
		tracer.timeNow = time.Now
	}

	if tracer.logger == nil {
		tracer.logger = NullLogger{}
	}

	if tracer.idGenerator == nil {
		tracer.idGenerator = func() string {
			return uuid.NewV4().String()
		}
	}

	dispatcher.SetLogger(tracer.logger)
	return tracer, tracer
}

/*StartSpan starts a new span*/
func (tracer *Tracer) StartSpan(
	operationName string,
	options ...opentracing.StartSpanOption,
) opentracing.Span {
	sso := opentracing.StartSpanOptions{}

	for _, o := range options {
		o.Apply(&sso)
	}

	if sso.StartTime.IsZero() {
		sso.StartTime = tracer.timeNow()
	}

	var followsFromIsParent = false
	var parent *SpanContext

	for _, ref := range sso.References {
		if ref.Type == opentracing.ChildOfRef {
			if parent == nil || followsFromIsParent {
				parent = ref.ReferencedContext.(*SpanContext)
			}
		} else if ref.Type == opentracing.FollowsFromRef {
			if parent == nil {
				parent = ref.ReferencedContext.(*SpanContext)
				followsFromIsParent = true
			}
		}
	}

	spanContext := tracer.createSpanContext(parent)

	span := &_Span{
		tracer:        tracer,
		context:       spanContext,
		operationName: operationName,
		startTime:     sso.StartTime,
		duration:      0,
	}

	for _, tag := range tracer.Tags() {
		span.SetTag(tag.Key, tag.Value)
	}
	for k, v := range sso.Tags {
		span.SetTag(k, v)
	}

	return span
}

func (tracer *Tracer) createSpanContext(parent *SpanContext) *SpanContext {
	if parent == nil || !parent.IsValid() {
		return &SpanContext{
			TraceID: tracer.idGenerator(),
			SpanID:  tracer.idGenerator(),
		}
	}
	return &SpanContext{
		TraceID:  parent.TraceID,
		SpanID:   tracer.idGenerator(),
		ParentID: parent.SpanID,
		Baggage:  parent.Baggage,
	}
}

/*Inject implements Inject() method of opentracing.Tracer*/
func (tracer *Tracer) Inject(ctx opentracing.SpanContext, format interface{}, carrier interface{}) error {
	c, ok := ctx.(*SpanContext)
	if !ok {
		return opentracing.ErrInvalidSpanContext
	}
	if injector, ok := tracer.propagtors[format]; ok {
		return injector.Inject(c, carrier)
	}
	return opentracing.ErrUnsupportedFormat
}

/*Extract implements Extract() method of opentracing.Tracer*/
func (tracer *Tracer) Extract(
	format interface{},
	carrier interface{},
) (opentracing.SpanContext, error) {
	if extractor, ok := tracer.propagtors[format]; ok {
		return extractor.Extract(carrier)
	}
	return nil, opentracing.ErrUnsupportedFormat
}

/*Tags return all common tags */
func (tracer *Tracer) Tags() []opentracing.Tag {
	return tracer.commonTags
}

/*DispatchSpan dispatches the span to a dispatcher*/
func (tracer *Tracer) DispatchSpan(span *_Span) {
	if tracer.dispatcher != nil {
		tracer.dispatcher.Dispatch(span)
	}
}

/*Close closes the tracer*/
func (tracer *Tracer) Close() error {
	if tracer.dispatcher != nil {
		tracer.dispatcher.Close()
	}
	return nil
}
