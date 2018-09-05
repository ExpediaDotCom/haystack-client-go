[![Build Status](https://travis-ci.org/ExpediaDotCom/haystack-client-go.svg?branch=master)](https://travis-ci.org/ExpediaDotCom/haystack-client-go)
[![License](https://img.shields.io/badge/license-Apache%20License%202.0-blue.svg)](https://github.com/ExpediaDotCom/haystack/blob/master/LICENSE)

# Haystack bindings for Go OpenTracing API.

This is Haystack's client library for Golang that implements [OpenTracing API 1.0](https://github.com/opentracing/opentracing-go/).


## How to use the library?

Check our detailed [example](examples/example.go) on how to initialize tracer, start a span and send it to one of the dispatchers. This example is actually an integration test uses haystack-agent container 


## How to build this library?
`git clone --recursive https://github.com/ExpediaDotCom/haystack-client-go` - clone the repo 

`make glide` - if you are running for the very first time

`make test validate` - go test and validate the code 
