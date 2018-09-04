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
	"testing"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/suite"
)

type TracerTestSuite struct {
	suite.Suite
	tracer     opentracing.Tracer
	closer     io.Closer
	dispatcher Dispatcher
}

func (suite *TracerTestSuite) SetupTest() {
	suite.dispatcher = NewInMemoryDispatcher()
	tracer, closer := NewTracer("my-service", suite.dispatcher, TracerOptionsFactory.Tag("t1", "v1"))
	suite.tracer = tracer
	suite.closer = closer
}

func (suite *TracerTestSuite) TearDownTest() {
	if suite.tracer != nil {
		err := suite.closer.Close()
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

	suite.Equal(parentSpanContext.TraceID(), childSpanContext.TraceID(), "trace id should match for both parent and child span")
	suite.Equal(parentSpanContext.SpanID(), childSpanContext.ParentID(), "parent id should match for parent and child span")
	suite.NotEqual(parentSpanContext.SpanID(), childSpanContext.SpanID(), "span id should be unique for parent and child span")

	for _, span := range dispatcher.spans {
		suite.Len(span.Tags(), 2, "1 tag should be present on the span")
		suite.Equal(span.Tags()[0], opentracing.Tag{Key: "t1", Value: "v1"}, "tag key should be t1")
		suite.Equal(span.Tags()[1], tag, "tag key should be user_agent")
	}
}

func (suite *TracerTestSuite) TestTracerInject() {
	carrier := make(map[string]string)
	span1 := suite.tracer.StartSpan("op1")
	err := suite.tracer.Inject(span1.Context(), opentracing.HTTPHeaders, carrier)
	if err != nil {
		panic(err)
	}
	suite.Len(carrier, 3, "trace-id, span-id, parent-id should be injected in the http headers")

	spanContext, err := suite.tracer.Extract(opentracing.HTTPHeaders, carrier)
	if err != nil {
		panic(err)
	}

	suite.Equal(carrier["Trace-ID"], spanContext.(*SpanContext).TraceID())
	suite.Equal(carrier["Span-ID"], spanContext.(*SpanContext).SpanID())
	suite.Equal(carrier["Parent-ID"], spanContext.(*SpanContext).ParentID())
}

func TestTracerTestSuite(t *testing.T) {
	suite.Run(t, new(TracerTestSuite))
}
