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
	"log"
	"strconv"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/magiconair/properties/assert"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

func TestIntegrationWithHaystackAgent(t *testing.T) {
	tracer, closer := NewTracer("dummy-service", NewAgentDispatcher("haystack_agent", 35000, 3*time.Second, 1000), TracerOptionsFactory.Tag("appVer", "v1.1"))
	defer func() {
		err := closer.Close()
		if err != nil {
			panic(err)
		}
	}()

	serverTime := time.Now()
	serverSpan := tracer.StartSpan("serverOp", opentracing.Tag{Key: "server-tag-key", Value: "something"}, opentracing.StartTime(serverTime))
	ext.SpanKind.Set(serverSpan, ext.SpanKindRPCServerEnum)
	ext.Error.Set(serverSpan, false)
	ext.HTTPStatusCode.Set(serverSpan, 200)
	ext.HTTPMethod.Set(serverSpan, "POST")
	serverSpan.LogEventWithPayload("code", 1001)

	clientTime := time.Now()
	clientSpan := tracer.StartSpan("clientOp", opentracing.ChildOf(serverSpan.Context()), opentracing.StartTime(clientTime))
	ext.SpanKind.Set(clientSpan, ext.SpanKindRPCClientEnum)
	ext.Error.Set(clientSpan, true)
	ext.HTTPStatusCode.Set(clientSpan, 404)

	// finish the two spans
	clientSpan.Finish()
	serverSpan.Finish()

	consumer, err := sarama.NewConsumer([]string{"kafkasvc:9092"}, nil)

	if err != nil {
		panic(err)
	}

	defer func() {
		if err := consumer.Close(); err != nil {
			log.Fatalln(err)
		}
	}()

	partitionConsumer, err := consumer.ConsumePartition("proto-spans", 0, sarama.OffsetOldest)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := partitionConsumer.Close(); err != nil {
			log.Fatalln(err)
		}
	}()

	clientSpanReceived := 0
	serverSpanReceived := 0
	clientParentSpanID := ""
	clientTraceID := ""

	serverSpanID := ""
	serverTraceID := ""

ConsumerLoop:
	for {
		select {
		case msg := <-partitionConsumer.Messages():
			span := &Span{}
			unmarshalErr := span.XXX_Unmarshal(msg.Value)
			if unmarshalErr != nil {
				panic(err)
			}

			verifyCommonAttr(t, span)
			for _, tag := range span.GetTags() {
				if tag.GetKey() == "span.kind" {
					switch tag.GetVStr() {
					case "client":
						clientSpanReceived = clientSpanReceived + 1
						assert.Equal(t, span.GetOperationName(), "clientOp")
						assert.Equal(t, span.StartTime, int64(clientTime.UnixNano()/int64(time.Microsecond)))
						assert.Equal(t, tagVal(span, "server-tag-key"), "")
						assert.Equal(t, tagVal(span, "error"), "true")
						assert.Equal(t, tagVal(span, string(ext.HTTPStatusCode)), "404")
						clientParentSpanID = span.GetParentSpanId()
						clientTraceID = span.GetTraceId()
					case "server":
						serverSpanReceived = serverSpanReceived + 1
						assert.Equal(t, span.GetOperationName(), "serverOp")
						assert.Equal(t, span.StartTime, int64(serverTime.UnixNano()/int64(time.Microsecond)))
						assert.Equal(t, tagVal(span, "server-tag-key"), "something")
						assert.Equal(t, tagVal(span, "error"), "false")
						assert.Equal(t, tagVal(span, string(ext.HTTPStatusCode)), "200")
						serverSpanID = span.GetSpanId()
						serverTraceID = span.GetTraceId()
					}
				}
			}

			// expect only two spans and compare the ID relationship
			if msg.Offset == 1 {
				assert.Equal(t, clientSpanReceived, 1)
				assert.Equal(t, serverSpanReceived, 1)
				assert.Equal(t, serverSpanID, clientParentSpanID)
				assert.Equal(t, serverTraceID, clientTraceID)
				break ConsumerLoop
			}
		}
	}
}

func verifyCommonAttr(t *testing.T, span *Span) {
	assert.Equal(t, span.GetServiceName(), "dummy-service")
	assert.Equal(t, tagVal(span, "appVer"), "v1.1")
}

func tagVal(span *Span, tagKey string) string {
	for _, tag := range span.GetTags() {
		if tag.GetKey() == tagKey {
			switch tag.GetType() {
			case Tag_STRING:
				return tag.GetVStr()
			case Tag_BOOL:
				return strconv.FormatBool(tag.GetVBool())
			case Tag_LONG:
				return fmt.Sprintf("%d", tag.GetVLong())
			case Tag_DOUBLE:
				return fmt.Sprintf("%f", tag.GetVDouble())
			}
		}
	}
	return ""
}
