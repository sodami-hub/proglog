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

#   client 인증서 생성(서버와 같은 CA로 클라이언트의 인증서를 생성한다.)
	cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=test/ca-config.json -profile=client\
		test/client-csr.json | cfssljson -bare client

# ACL(권한)에 대한 테스트를 위해서 여러 권한을 가진 클라이언트를 생성한다. - multi client
	cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=test/ca-config.json -profile=client\
		-cn="root" test/client-csr.json | cfssljson -bare root-client

	cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=test/ca-config.json -profile=client\
		-cn="nobody" test/client-csr.json | cfssljson -bare nobody-client

	mv *.pem *.csr ${CONFIG_PATH}

$(CONFIG_PATH)/model.conf:
	cp test/model.conf $(CONFIG_PATH)/model.conf
$(CONFIG_PATH)/policy.csv:
	cp test/policy.csv $(CONFIG_PATH)/policy.csv

.PHONY: test
test: $(CONFIG_PATH)/policy.csv $(CONFIG_PATH)/model.conf
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