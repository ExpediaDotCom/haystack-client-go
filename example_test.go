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
	"testing"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

func TestIntegrationWithAgent(t *testing.T) {
	tracer, closer := NewTracer("haystack-agent-test-app", NewFileDispatcher("spans"), TracerOptionsFactory.Tag("lang", "go"), TracerOptionsFactory.Tag("appVer", "v1.1"))
	defer func() {
		err := closer.Close()
		if err != nil {
			panic(err)
		}
	}()

	span1 := tracer.StartSpan("operation1", opentracing.Tag{Key: "my-tag", Value: "something"})
	ext.SpanKind.Set(span1, ext.SpanKindRPCServerEnum)
	ext.Error.Set(span1, false)
	ext.HTTPStatusCode.Set(span1, 200)
	ext.HTTPMethod.Set(span1, "POST")
	span1.LogEventWithPayload("code", 1001)

	span2 := tracer.StartSpan("operation2", opentracing.ChildOf(span1.Context()))
	ext.Error.Set(span2, true)
	ext.HTTPStatusCode.Set(span1, 404)
	span2.Finish()
	span1.Finish()
}
