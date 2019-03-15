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
	"strings"
	"testing"

	opentracing "github.com/opentracing/opentracing-go"

	"github.com/stretchr/testify/suite"
)

type TracerTestSuite struct {
	suite.Suite
	tracer               opentracing.Tracer
	closer               io.Closer
	dualSpanModeTracer   opentracing.Tracer
	dualSpanTracerCloser io.Closer

	dispatcher Dispatcher
}

func (suite *TracerTestSuite) SetupTest() {
	suite.dispatcher = NewInMemoryDispatcher()
	tracer, closer := NewTracer("my-service", suite.dispatcher, TracerOptionsFactory.Tag("t1", "v1"))
	dualSpanModeTracer, dualSpanTracerCloser := NewTracer("my-service", suite.dispatcher, TracerOptionsFactory.Tag("t1", "v1"), TracerOptionsFactory.UseDualSpanMode())
	suite.tracer = tracer
	suite.closer = closer
	suite.dualSpanModeTracer = dualSpanModeTracer
	suite.dualSpanTracerCloser = dualSpanTracerCloser
}

func (suite *TracerTestSuite) TearDownTest() {
	if suite.tracer != nil {
		err := suite.closer.Close()
		if err != nil {
			panic(err)
		}
	}

	if suite.dualSpanTracerCloser != nil {
		err := suite.dualSpanTracerCloser.Close()
		if err != nil {
			panic(err)
		}
	}
}
func (suite *TracerTestSuite) TestTracerProperties() {
	tag := opentracing.Tag{Key: "user_agent", Value: "ua"}
	span1 := suite.tracer.StartSpan("op1", tag)
	span2 := suite.tracer.StartSpan("op2", opentracing.ChildOf(span1.Context()), tag)

	span2.Finish()
	span1.Finish()

	dispatcher := suite.dispatcher.(*InMemoryDispatcher)
	suite.Len(dispatcher.spans, 2, "2 spans should be dispatched")
	childSpan := dispatcher.spans[0]
	childSpanContext := childSpan.Context().(*SpanContext)

	parentSpan := dispatcher.spans[1]
	parentSpanContext := parentSpan.Context().(*SpanContext)

	suite.Equal(parentSpanContext.TraceID, childSpanContext.TraceID, "trace id should match for both parent and child span")
	suite.Equal(parentSpanContext.SpanID, childSpanContext.ParentID, "parent id should match for parent and child span")
	suite.NotEqual(parentSpanContext.SpanID, childSpanContext.SpanID, "span id should be unique for parent and child span")

	for _, span := range dispatcher.spans {
		suite.Len(span.Tags(), 2, "1 tag should be present on the span")
		suite.Equal(span.Tags()[0], opentracing.Tag{Key: "t1", Value: "v1"}, "tag key should be t1")
		suite.Equal(span.Tags()[1], tag, "tag key should be user_agent")
	}
}

func (suite *TracerTestSuite) TestTracerWithDualSpanMode_1() {
	serverTag := opentracing.Tag{Key: "span.kind", Value: "server"}
	clientTag := opentracing.Tag{Key: "span.kind", Value: "client"}
	headerMap := map[string]string{
		"trace-id":        "T1",
		"Span-ID":         "S1",
		"Parent-ID":       "P1",
		"Baggage-myKey":   "myVal",
		"baggage-mykey-1": "myval",
	}

	upstreamSpanContext, _ := suite.dualSpanModeTracer.Extract(opentracing.HTTPHeaders, buildHTTPHeaderCarrier(headerMap))
	serverSpan := suite.dualSpanModeTracer.StartSpan("op1", serverTag, opentracing.ChildOf(upstreamSpanContext))
	clientSpan := suite.dualSpanModeTracer.StartSpan("op2", clientTag, opentracing.ChildOf(serverSpan.Context()))

	clientSpan.Finish()
	serverSpan.Finish()

	dispatcher := suite.dispatcher.(*InMemoryDispatcher)
	suite.Len(dispatcher.spans, 2, "2 spans should be dispatched")
	receivedClientSpan := dispatcher.spans[0]
	receivedClientSpanCtx := receivedClientSpan.Context().(*SpanContext)

	receivedServerSpan := dispatcher.spans[1]
	receivedServerSpanCtx := receivedServerSpan.Context().(*SpanContext)
	suite.Equal(receivedServerSpanCtx.TraceID, "T1", "Trace Ids should match")
	suite.Equal(receivedServerSpanCtx.Baggage[strings.ToLower("myKey")], "myVal", "baggage key should match")
	suite.Equal(receivedServerSpanCtx.Baggage[strings.ToLower("mykey-1")], "myval", "baggage lowercase key should match")
	suite.NotEqual(receivedServerSpanCtx.SpanID, "S1", "SpanId should be newly created")
	suite.NotEqual(receivedServerSpanCtx.SpanID, "P1", "SpanId should be newly created")
	suite.NotEqual(receivedServerSpanCtx.SpanID, "T1", "SpanId should be newly created")
	suite.Equal(receivedServerSpanCtx.ParentID, "S1", "Parent Ids should match")
	suite.Equal(receivedServerSpan.Tags()[1], serverTag, "span.kind tag should be present and equal to server")

	suite.Equal(receivedClientSpanCtx.TraceID, "T1", "Parent Ids should match")
	suite.NotEqual(receivedClientSpanCtx.SpanID, receivedServerSpanCtx.SpanID, "SpanId should be newly created")
	suite.NotEqual(receivedClientSpanCtx.SpanID, receivedServerSpanCtx.SpanID, "SpanId should be newly created")
	suite.NotEqual(receivedClientSpanCtx.SpanID, receivedServerSpanCtx.SpanID, "SpanId should be newly created")
	suite.Equal(receivedClientSpanCtx.ParentID, receivedServerSpanCtx.SpanID, "Parent Ids should match")
	suite.Equal(receivedClientSpan.Tags()[1], clientTag, "span.kind tag should be present and equal to client")
}

func (suite *TracerTestSuite) TestTracerWithSingleSpanMode_1() {
	serverTag := opentracing.Tag{Key: "span.kind", Value: "server"}
	clientTag := opentracing.Tag{Key: "span.kind", Value: "client"}
	headerMap := map[string]string{
		"Trace-ID":  "T1",
		"Span-ID":   "S1",
		"Parent-ID": "P1",
	}

	upstreamSpanContext, _ := suite.tracer.Extract(opentracing.HTTPHeaders, buildHTTPHeaderCarrier(headerMap))
	serverSpan := suite.tracer.StartSpan("op1", serverTag, opentracing.ChildOf(upstreamSpanContext))
	clientSpan := suite.tracer.StartSpan("op2", clientTag, opentracing.ChildOf(serverSpan.Context()))

	clientSpan.Finish()
	serverSpan.Finish()

	dispatcher := suite.dispatcher.(*InMemoryDispatcher)
	suite.Len(dispatcher.spans, 2, "2 spans should be dispatched")
	receivedClientSpan := dispatcher.spans[0]
	receivedClientSpanCtx := receivedClientSpan.Context().(*SpanContext)

	receivedServerSpan := dispatcher.spans[1]
	receivedServerSpanCtx := receivedServerSpan.Context().(*SpanContext)
	suite.Equal(receivedServerSpanCtx.TraceID, "T1", "Trace Ids should match")
	suite.Equal(receivedServerSpanCtx.SpanID, "S1", "Span Ids should match")
	suite.Equal(receivedServerSpanCtx.ParentID, "P1", "Parent Ids should match")
	suite.Equal(receivedServerSpan.Tags()[1], serverTag, "span.kind tag should be present and equal to server")

	suite.Equal(receivedClientSpanCtx.TraceID, "T1", "Parent Ids should match")
	suite.NotEqual(receivedClientSpanCtx.SpanID, "S1", "SpanId should be newly created")
	suite.NotEqual(receivedClientSpanCtx.SpanID, "P1", "SpanId should be newly created")
	suite.NotEqual(receivedClientSpanCtx.SpanID, "T1", "SpanId should be newly created")
	suite.Equal(receivedClientSpanCtx.ParentID, "S1", "Parent Ids should match")
	suite.Equal(receivedClientSpan.Tags()[1], clientTag, "span.kind tag should be present and equal to client")
}

func (suite *TracerTestSuite) TestTracerWithSingleSpanMode_2() {
	serverTag := opentracing.Tag{Key: "error", Value: true}
	clientTag := opentracing.Tag{Key: "error", Value: false}
	headerMap := map[string]string{
		"Trace-ID":  "T1",
		"Span-ID":   "S1",
		"Parent-ID": "P1",
	}

	upstreamSpanContext, _ := suite.tracer.Extract(opentracing.HTTPHeaders, buildHTTPHeaderCarrier(headerMap))
	serverSpan := suite.tracer.StartSpan("op1", serverTag, opentracing.ChildOf(upstreamSpanContext))
	clientSpan := suite.tracer.StartSpan("op2", clientTag, opentracing.ChildOf(serverSpan.Context()))

	clientSpan.Finish()
	serverSpan.Finish()

	dispatcher := suite.dispatcher.(*InMemoryDispatcher)
	suite.Len(dispatcher.spans, 2, "2 spans should be dispatched")
	receivedClientSpan := dispatcher.spans[0]
	receivedClientSpanCtx := receivedClientSpan.Context().(*SpanContext)

	receivedServerSpan := dispatcher.spans[1]
	receivedServerSpanCtx := receivedServerSpan.Context().(*SpanContext)

	suite.Equal(receivedServerSpanCtx.TraceID, "T1", "Trace Ids should match")
	suite.Equal(receivedServerSpanCtx.SpanID, "S1", "Span Ids should match")
	suite.Equal(receivedServerSpanCtx.ParentID, "P1", "Parent Ids should match")
	suite.Equal(receivedServerSpan.Tags()[1].Key, "error", "error tag should be present")
	suite.Equal(receivedServerSpan.Tags()[1].Value, true, "error tag should be true")

	suite.Equal(receivedClientSpanCtx.TraceID, "T1", "Parent Ids should match")
	suite.NotEqual(receivedClientSpanCtx.SpanID, "S1", "SpanId should be newly created")
	suite.NotEqual(receivedClientSpanCtx.SpanID, "P1", "SpanId should be newly created")
	suite.NotEqual(receivedClientSpanCtx.SpanID, "T1", "SpanId should be newly created")
	suite.Equal(receivedClientSpanCtx.ParentID, "S1", "Parent Ids should match")
	suite.Equal(receivedClientSpan.Tags()[1].Key, "error", "error tag should be present")
	suite.Equal(receivedClientSpan.Tags()[1].Value, false, "error tag should be false")
}

func (suite *TracerTestSuite) TestTracerInject() {
	carrier := opentracing.HTTPHeadersCarrier(make(map[string][]string))

	span1 := suite.tracer.StartSpan("op1")
	err := suite.tracer.Inject(span1.Context(), opentracing.HTTPHeaders, carrier)
	if err != nil {
		panic(err)
	}
	suite.Len(carrier, 3, "trace-id, span-id, parent-id should be injected in the http headers")

	ctx, err := suite.tracer.Extract(opentracing.HTTPHeaders, carrier)
	if err != nil {
		panic(err)
	}

	suite.NotEqual("", ctx.(*SpanContext).TraceID)
	suite.NotEqual("", ctx.(*SpanContext).SpanID)
	suite.NotEqual("", ctx.(*SpanContext).ParentID)
	suite.Equal(true, ctx.(*SpanContext).IsExtractedContext)
}

func buildHTTPHeaderCarrier(headerMap map[string]string) *opentracing.HTTPHeadersCarrier {
	httpHeaderCarrier := opentracing.HTTPHeadersCarrier(make(map[string][]string))
	for k, v := range headerMap {
		httpHeaderCarrier.Set(k, v)
	}
	return &httpHeaderCarrier
}

func TestUnitTracerSuite(t *testing.T) {
	suite.Run(t, new(TracerTestSuite))
}
