/*
/handleProduce 요청의 Body에 일반적인 문자열을 value에 넣으면 아래와 같은 메시지가 응답된다.
illegal base64 data at input byte 0

Go의 encoding/json 패키지는 []byte를 base64-encoding string으로 인코딩한다. 그래서 value 값으로
base64로 인코딩된 값을 넣어야 된다.

$ curl -X POST localhost:8080 -d '{"record" : {"value":"TGV0J3MgR28gIzEk"}}'
{"offset":0}
$ curl -X GET localhost:8080 -d '{"offset":0}'
{"record":{"value":"TGV0J3MgR28gIzEk","offset":0}}
*/

package main

import (
	"fmt"
	"log"

	"github.com/sodami-hub/proglog/internal/server"
)

func main() {
	srv := server.NewHTTPServer(":8080")
	fmt.Println("Listening localhost:8080 ... ")
	log.Fatal(srv.ListenAndServe())
}
