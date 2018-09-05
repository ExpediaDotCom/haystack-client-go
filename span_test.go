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

type SpanTestSuite struct {
	suite.Suite
	tracer     opentracing.Tracer
	closer     io.Closer
	dispatcher Dispatcher
}

func (suite *SpanTestSuite) SetupTest() {
	suite.dispatcher = NewInMemoryDispatcher()
	tracer, closer := NewTracer("my-service", suite.dispatcher, TracerOptionsFactory.Tag("t1", "v1"))
	suite.tracer = tracer
	suite.closer = closer
}

func (suite *SpanTestSuite) TearDownTest() {
	if suite.tracer != nil {
		err := suite.closer.Close()
		if err != nil {
			panic(err)
		}
	}
}
func (suite *SpanTestSuite) TestSpanProperties() {
	sp1 := suite.tracer.StartSpan("op1").(*_Span)
	suite.Equal(sp1.ServiceName(), "my-service")
	suite.Len(sp1.Tags(), 1)
	suite.Equal(sp1.Tags()[0].Key, "t1")
	suite.Equal(sp1.Tags()[0].Value.(string), "v1")
	suite.Equal("op1", sp1.OperationName())
	suite.Equal(suite.tracer, sp1.Tracer())
	suite.NotNil(sp1.Context())
}
func (suite *SpanTestSuite) TestBaggageIterator() {
	span1 := suite.tracer.StartSpan("sp1").(*_Span)

	span1.SetBaggageItem("bk1", "something")
	span1.SetBaggageItem("bk2", "100")
	expectedBaggage := map[string]string{"bk1": "something", "bk2": "100"}
	suite.Equal(expectedBaggage, extractBaggage(span1))
	assertBaggageRecords(suite, span1, expectedBaggage)

	span2 := suite.tracer.StartSpan("sp2", opentracing.ChildOf(span1.Context())).(*_Span)
	suite.Equal(expectedBaggage, extractBaggage(span2))
}

func extractBaggage(sp opentracing.Span) map[string]string {
	b := make(map[string]string)
	sp.Context().ForeachBaggageItem(func(k, v string) bool {
		b[k] = v
		return true
	})
	return b
}

func assertBaggageRecords(suite *SpanTestSuite, sp *_Span, expected map[string]string) {
	suite.Len(sp.logs, len(expected))
	for _, logRecord := range sp.logs {
		suite.Len(logRecord.Fields, 3)
		suite.Equal("event:baggage", logRecord.Fields[0].String())
		key := logRecord.Fields[1].Value().(string)
		value := logRecord.Fields[2].Value().(string)

		suite.Contains(expected, key)
		suite.Equal(expected[key], value)
	}
}

func TestUnitSpanSuite(t *testing.T) {
	suite.Run(t, new(SpanTestSuite))
}
