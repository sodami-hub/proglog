# protocolbuf 컴파일 명령
compile:
	protoc api/v1/*.proto \
	--go_out=. \
	--go_opt=paths=source_relative \
	--proto_path=.
test:
	go test -race ./...

# gRPC 서비스 컴파일
compile :
	protoc api/v1/*.proto \
		--go_out=/ \
		--go-grpc_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		--proto_path=.