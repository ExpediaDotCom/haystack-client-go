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
	"fmt"
	"net/http"
	"time"

	haystack "github.com/ExpediaDotCom/haystack-client-go"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

type consoleLogger struct{}

/*Error prints the error message*/
func (logger *consoleLogger) Error(format string, v ...interface{}) {
	fmt.Printf(format, v...)
	fmt.Print("\n")
}

/*Info prints the info message*/
func (logger *consoleLogger) Info(format string, v ...interface{}) {
	fmt.Printf(format, v...)
	fmt.Print("\n")
}

/*Debug prints the info message*/
func (logger *consoleLogger) Debug(format string, v ...interface{}) {
	fmt.Printf(format, v...)
	fmt.Print("\n")
}

func main() {
	/*Use haystack.NewDefaultAgentDispatcher() for non-dev environment*/
	tracer, closer := haystack.NewTracer("dummy-service", haystack.NewDefaultAgentDispatcher(), haystack.TracerOptionsFactory.Tag("appVer", "v1.1"), haystack.TracerOptionsFactory.Logger(&consoleLogger{}))
	defer func() {
		err := closer.Close()
		if err != nil {
			panic(err)
		}
	}()

	carrier := opentracing.HTTPHeadersCarrier(http.Header(map[string][]string{
		"Trace-Id":  {"409dec0e-473e-11e9-81cc-186590cf29af"},
		"Span-Id":   {"509df8d3-473e-11e9-81cc-186590cf29af"},
		"Parent-Id": {"609df88c-473e-11e9-81cc-186590cf29af"},
	}))
	clientContext, err := tracer.Extract(opentracing.HTTPHeaders, carrier)
	if err != nil {
		panic(err)
	}
	span1 := tracer.StartSpan("operation1", opentracing.ChildOf(clientContext), opentracing.Tag{Key: "my-tag", Value: "something"})
	span1.SetTag(string(ext.SpanKind), ext.SpanKindRPCServerEnum)
	span1.SetTag(string(ext.Error), false)
	span1.SetTag(string(ext.HTTPStatusCode), 200)
	span1.SetTag(string(ext.HTTPMethod), "POST")
	span1.LogEventWithPayload("code", 1001)

	span2 := tracer.StartSpan("operation2", opentracing.ChildOf(span1.Context()))
	// a slightly different way to set the tags on a span
	ext.Error.Set(span2, true)
	ext.HTTPStatusCode.Set(span1, 404)

	// inject the span context in http header using tracer.Inject(...)
	req, _ := http.NewRequest(http.MethodGet, "https://expediadotcom.github.io/haystack/", nil)
	tracer.Inject(span2.Context(), opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(req.Header))
	//
	//log.Println(req.Header)

	span2.Finish()
	span1.Finish()

	time.Sleep(5 * time.Second)
}
