.PHONY: codegen
codegen: idl-submodule
	go get -u github.com/golang/protobuf/protoc-gen-go	
	cp haystack-idl/proto/agent/spanAgent.proto haystack-idl/proto/.
	protoc -I haystack-idl/proto/  --go_out=plugins=grpc:. haystack-idl/proto/span.proto	
	protoc -I haystack-idl/proto/  --go_out=plugins=grpc:. haystack-idl/proto/spanAgent.proto
	rm 	haystack-idl/proto/spanAgent.proto

idl-submodule:
	git submodule init
	git submodule update

.PHONY: test
test: codegen
	go test -run TestUnit*

.PHONY: integration_test
integration_test:
	docker-compose -f docker-compose.yaml -p sandbox up -d
	sleep 30
	go test -run TestIntegration*
	docker-compose -f docker-compose.yaml -p sandbox stop

.PHONY: glide
glide:
	glide --version || go get github.com/Masterminds/glide
	glide update

.PHONY: validate
validate:
	./scripts/validate-go
