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

package main

import (
	haystack "github.com/ExpediaDotCom/haystack-client-go"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

func main() {
	/*Use haystack.NewDefaultAgentDispatcher() for non-dev environment*/
	tracer, closer := haystack.NewTracer("haystack-agent-test-app", haystack.NewFileDispatcher("spans"), haystack.TracerOptionsFactory.Tag("appVer", "v1.1"))
	defer func() {
		err := closer.Close()
		if err != nil {
			panic(err)
		}
	}()

	span1 := tracer.StartSpan("operation1", opentracing.Tag{Key: "my-tag", Value: "something"})
	span1.SetTag(string(ext.SpanKind), ext.SpanKindRPCServerEnum)
	span1.SetTag(string(ext.Error), false)
	span1.SetTag(string(ext.HTTPStatusCode), 200)
	span1.SetTag(string(ext.HTTPMethod), "POST")
	span1.LogEventWithPayload("code", 1001)

	span2 := tracer.StartSpan("operation2", opentracing.ChildOf(span1.Context()))
	// a slightly different way to set the tags on a span
	ext.Error.Set(span2, true)
	ext.HTTPStatusCode.Set(span1, 404)

	span2.Finish()
	span1.Finish()
}
