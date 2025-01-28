# # protocolbuf 컴파일 명령
# compile:
# 	protoc api/v1/*.proto \
# 	--go_out=. \
# 	--go_opt=paths=source_relative \
# 	--proto_path=.
# test:
# 	go test -race ./...

# # gRPC 서비스 컴파일
# compile :
# 	protoc api/v1/*.proto \
# 		--go_out=. \
# 		--go-grpc_out=. \
# 		--go_opt=paths=source_relative \
# 		--go-grpc_opt=paths=source_relative \
# 		--proto_path=.


# cfssl, cfssljson을 사용해서 인증서 생성
CONFIG_PATH=${HOME}/.proglog/

.PHONY:init
init:
	mkdir -p ${CONFIG_PATH}

.PHONY:gencert
gencert:
	cfssl gencert -initca test/ca-csr.json | cfssljson -bare ca

	cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=test/ca-config.json -profile=server\
		test/server-csr.json | cfssljson -bare server

# client 인증서 생성
	cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=test/ca-config.json -profile=client\
		test/client-csr.json | cfssljson -bare client

	mv *.pem *.csr ${CONFIG_PATH}

.PHONY: test
test:
	go test -race ./...

# protobuf 컴파일
.PHONY: compile
compile:
	protoc api/v1/*.proto \
		--go_out=. \
		--go-grpc_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		--proto_path=.