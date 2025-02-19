# Go를 사용한 분산서비스 개발
소프트웨어 분야에서 Go가 가장 큰 영향을 끼친 분야는 분산시스템이 아닐까? 도커, 쿠버네티스, etcd, 프로메테우스와 같은 프로젝트들은 고로 개발되었다. 
이 프로젝트를 통해서 Go를 사용한 분산시스템을 만들어보고, 이 과정에서 얻은 아이디어를 바탕으로 나만의 소프트웨어를 개발해보고자 한다.

## PART 1
#### 1. 프로젝트 시작

#### 2. 프로토콜 버퍼와 구조체
분산 서비스에서 서비스들은 네트워크로 통신한다. 구조체와 같은 데이터를 네트워크로 보내려면 우선 데이터를 전송할 수 있는 형태로 인코딩해야 한다. 대표적인 형태가 JSON이다. public API 또는 클라이언트를 통제할 수 없는 프로젝트를 만든다면 JSON을 선택한다. JSON은 사람이 읽기에도 컴퓨터가 파싱하기에도 좋다. 하지만 private API 또는 클라이언트를 통제할 수 있는 상황이라면 JSON 보다 더 나은 데이터 구조화와 전송 방식을 택할 수 있다. 생산성이 더 높으면서 빠르고 기능도 많으며 버그도 적은 서비스를 만들 수 있는 인코딩 방식말이다.
그게 바로 프로토콜 버퍼(protocol buffer, 이하 protobuf)이다. 구글에서 만든 protobuf는 언어와 플랫폼에 상관없이 쓸 수 있고 확장할 수 있는, 데이터를 구조화하고 직렬화하는 메커니즘이다. protobuf의 장점은 다음과 같다.
- 자료형 안정성 보장
- 스키마 위반 방지
- 빠른 직렬화
- 하위 호환성 제공

protobuf를 사용하면 데이터 구조를 정의할 수 있고, protobuf가 지원하는 많은 언어의 코드로 컴파일할 수 잇으며, 이후 구조화된 데이터를 다른 데이터 스트림에 쓰거나 읽을 수 있다. 프로토콜 버퍼는 마이크로서비스와 같은 두 시스템 사이에서 통신하기에 좋다. 구글이 고성능 RPC 프레임워크를 개발하고자 gRPC를 만들 때 protobuf를 사용한 이유이다.

##### 2.1 프로토콜 버퍼를 쓰는 이유
- 일관된 스키마 : protobuf로 의미하는 바(symentics)를 인코딩하고 서비스 전반에 걸쳐 사용하면 전체 시스템의 일관성이 보장된다. 
- 버전 관리 제거 : 구글이 protobuf를 만든 이유 중 하나는 버전을 확인할 필요성을 없애고 다음과 같은 지저분한 코드작성을 피하려는 것이다. 
```
if (version ==3) {
    ...
} else if (version > 4) {
    if (version == 5){
        ...
    }
    ...
}

```
- 줄어드는 보일러플레이트 코드
- 확장성
- 언어 불가지론 : 특정 언어에 국한되지 않는다. 서로 다른 언어로 만들어진 서비스 간에 통신할 때 별도의 작업이 필요 없다.
- 성능 : 고성능이며 페이로드가 적고, 직렬화할 때 JSON 보다 6배나 빠르다.

##### 2.2 프로토콜 버퍼 컴파일러 설치하기
```
// 내 시스템에서는 아래와 같은 명령으로 protoc 을 설치했다.
$ sudo apt install protobuf-compiler
$ sudo apt install protoc-gen-go
```
- protoc --version 명령으로 protobuf 컴파일러가 제대로 설치됐는지 확인할 수 있다.
- *.proto 파일 Go 코드로 컴파일하기. 
```
$ protoc [proto 파일 경로] --go_out=. --go_opt=paths=source_relative --proto_path=.
```

#### 3. 로그 패키지 작성
로그는 분산 서비스를 만드는 가장 중요한 도구이다. 미리 쓰기 로그(로그 선행 기입, write-ahead log, WAL), 트랜잭션 로그, 커밋 로그 등으로 부르는 로그는 스토리지 엔진, 메시지 큐, 버전 컨트롤, 복제(replication)와 합의 알고리즘(consensus algorithm)의 핵심이다. 분산 시스템을 만들며 마주치는 많은 무제는 로그로 해결할 수 있다. 직접 로그를 만들어보면 다음과 같은 내용을 배울 수 있다.
- 로그를 이요해 문제를 해결하거나, 어려운 문제를 좀 더 쉽게 만드는 방법
- 기존의 로그 기반 시스템을 변경하거나, 새로운 로그 기반 시스템을 만드는 방법
- 데이터를 효율적으로 읽고 쓰는 스토리지 엔진을 만드는 방법
- 시스템 오류에도 데이터 손실을 막는 방법
- 데이터를 디스크에 저장하기 위해 인코딩하거나 자신만의 와이어 프로토콜을 만들고 애플리케이션 간에 전송하는 방법

##### 3.1 로그의 강력함
파일 시스템이나 데이터베이스의 스토리지 엔진 개발자들은 시스템의 무결성을 높이고자 로그를 사용한다. 예를 들어, ext 파일 시스템의 저널, PostgreSQL과 같은 데이터베이스에서 사용하는 WAL, 리액트와 함께 사용하는 리덕스, 자바스크립트 라이브러리는 변경사항을 객체로 로그에 저장하고, 이러한 변경 사항을 순수 함수로 처리하여 애플리케이션의 상태를 업데이트한다. 이러한 예시로 보듯이 로그는 순서가 있는 데이터를 저장, 공유, 처리할 때 사용한다. 하나의 도구로 데이터베이스를 복제하고, 분산 서비스를 조율하며, 프런트엔드 애플리케이션의 상태를 관리할 수 있다. 시스템의 변경사항을 분리할 수 없는 단위의 연산까지 쪼개고 나눈 다음에 로그로 저장, 공유, 처리할 수 있다면 분산 서비스에서의 문제를 비롯한 많은 문제를 해결할 수 있다.
완벽한 로그는 마지막 상태를 포함한, 존재했던 모든 상태를 가지며, 이러한 로그만 있으면 복잡해보이던 기능도 얼마든지 만들 수 있다. 

##### 3.2 로그의 작동 원리
- 로그는 추가만 할 수 있는 레코드의 연속이다. 레코드는 로그의 끝에 추가하며, 보통은 최근의 로그들을 오래된 레코드부터 읽는다. 어떠한 데이터라도 로그로 저장할 수 있다. 우리는 로그라는 표현을 사람이 읽을 문자열이라는 의미로 써왔다. 하지만 로그 시스템 사용자가 많아지면서 다른 프로그램이 읽을 수 있는, 바이너리로 인코딩된 메시지라는 의미로 바뀌었다. 
우리가 사용할 로그 또는 레코드는 특정 자료형의 데이터를 의미하지 않는다. 로그에 레코드를 추가하면, 로그는 레코드에 고유하면서 순차적인 오프셋 숫자를 할당하는데, 이는 레코드의 ID와 같다. 로그는 레코드의 오프셋과 생성 시간으로 정렬된 데이터베이스 테이블과 같다.
- 로그를 구현하면서 마주치는 첫 번째 문제는 무한한 용량의 디스크는 없다는 점이다. 파일 하나에 끝없이 추가할 수 없으므로 로그를 여러 개의 세그먼트로 나눈다. 로그가 커지면, 이미 처리를 마쳤거나 다른 공간에 별도로 보관한 오래된 세그먼트부터 지우면서 디스크 공간을 확보한다. 서비스는 계속 새로운 세그먼트를 만들기도 하고 세그먼트로부터 데이터를 소비하기도 한다. 이때 고루틴들이 같은 데이터에 접근하더라도 충돌이 거의 발생하지 않는다.
- 세그먼트 목록에는 항상 하나의 활성 세그먼트(active segment)가 있다. 유일하게 레코드를 쓸 수 있는 세그먼트이다. 
- 세그먼트는 저장 파일과 인덱스 파일로 구성된다. 인덱스 파일은 레코드의 오프셋을 저장 파일의 실제 위치로 매핑해서 빠르게 읽을 수 있도록 한다. 특정 오프셋의 레코드를 읽으려면 먼저 인덱스 파일에서 원하는 레코드의 저장 파일에서 위치를 알아내고, 저장 파일에서 해당 위치의 레코드를 읽는다. 인덱스 파일은 오프셋과 저장 파일에서의 위치라는 두 필드만을 가지므로 저장소 파일보다 훨씬 작다. 따라서 메모리 맵 파일로 만들어서 파일 연산이 아닌 메모리 데이터를 다루듯 빠르게 만들 수 있다.

##### 3.3 로그 만들기
저장 파일과 인덱스 파일부터 하나씩 차근차근 만들고 세그먼트를 만든 다음 마지막으로 로그를 만든다. 만들어나가는 단계별로 테스트도 작성한다. 로그(log)는 레코드, 레코드 저장 파일, 세그먼트라는 추상적인 데이터 자료형을 아우르는 표현이기에 다음과 같이 용어들을 정리한다.
- 레코드 : 로그에 저장한 데이터
- 저장 파일 : 레코드를 저장하는 파일
- 인덱스 파일 : 인덱스를 저장하는 파일
- 세그먼트 : 저장 파일과 인데스 파일을 묶어서 말하는 추상적 개념
- 로그 : 모든 세그먼트를 묶어서 말하는 추상적 개념


## PART 2 - 네트워크
#### 4. gRPC 요청 처리
이번 장에서는 로그 라이브러리를 기반으로 여러 사람이 같은 데이터로 소통하는 서비스를 만든다. 서비스는 여러 컴퓨터에 걸쳐서 작동한다. 클러스터의 구현은 뒤에서 알아보고. 현재 분산 서비스에 대한 요청을 처리하는 최고의 도구는 구글의 gRPC이다.

##### 4.1 gRPC
과거 분산 서비스를 만들 때 가장 어려웠던 두 가지 문제는 호환성의 유지와 서버-클라이언트 사이의 성능을 관리하는 문제였다.
서버와 클라이언트의 호환성을 유지함으로써 클라이언트가 보내는 요청을 서버가 이해하고, 서버의 응답 역시 클라이언트가 이해할 수 있다는 것을 보장하고자 했다.
좋은 성능을 유지하기 위해 가장 중요한 것은 데이터베이스 쿼리의 최적화와, 비즈니스 로직의 구현에 사용하는 알고리즘의 최적화이다.
gRPC는 분산 시스템을 만두는 과정에서 문제 해결에 큰 도움이 되었다. gRPC의 장점은 무엇일까?

##### 4.2 서비스를 만들 때의 목표
네트워크로 제공하는 서비스를 만들 때 지향하는 목표가 무엇인지, 그리고 이러한 목표에 다가설 때 gRPC가 어떤 도움이 되는지를 정리해보겠다.
1. 단순화: 네트워크 통신은 기술적이며 복잡하다. 서비스를 만든다면 요청-응답의 직렬화 같은 세부적인 기술보다는 서비스로 풀어보려는 문제 자체에 집중하고, 기술적인 세부사항은 추상화되어 쉽게 가져다 쓸 수 있는 API를 사용하고 싶을 것이다. 다양한 추상화 수준의 프레임워크 중에서 gRPC의 추상화는 중.상급 수준이라 할 수 있다. 익스프레스보다는 높은 추상화이다. 
gRPC는 어떻게 직렬화할지, 엔드포인트를 어떻게 구성할지 정행져 있으며 양방향 스트리밍을 제공한다. 하지만 루비 온 레일즈 보다는 낮다고 할 수 있는데, 레일즈는 요청 처리에서부터 데이터 저장, 애패르리케이션의 구조에 이르기까지 모두 처리하기 때문이다. gRPC는 미들웨어를 이용해 확장할 수 있다. 서비스를 만들다 보면 로깅, 인증, 속도 제한, 트레이싱과 같은 많은 문제를 만나는데, gRPC 커뮤니티는 이러한 문제를 해결할 수 있는 많은 미들웨어를 만들어왔다.
2. 유지보수 : gRPC에서는 작은 변겨일 때는 protobuf의 필드 버저닝으로 충분하며, 메이저 변경이 있을 때는 서비스의 여러 버전을 쉽게 작성하고 실행할 수 있다.
3. 보안 : gRPC는 보안 소켓 계층(Secure Socket Layer,SSL)과 전송 계층 보안(Transprot Layer Security, TLS)를 지원하여 클라이언트와 서버 사이를 오가는 모든 데이터를 암호화한다. 
4. 사용성 : 
5. 성능 : gRPC는 protobuf와 HTTP/2 기반으로 만들어졌다. protobuf는 직렬화에 유리하고 HTTP/2는 연결을 오래 유지할 수 있는 이점이 있따. 덕분에 gRPC를 사용하는 서비스는 효율적으로 구동하며 서버 비용을 아낄 수 있다.
6. 확장성 : gRPC를 사용하면 필요에 따라 다양한 로드 밸런싱을 쓸 수 있다. 클라이언트 로드 밸런싱, 프록시 로드 밸런싱, 색인 로드 밸런싱, 그리고 서비스 메시 등이 있다. 또한 gRPC를 사용하면 gRPC가 지원하는 다양한 언어로 서비슬르 클라이언트 및 서버로 컴파일할 수 있다.

#### 5. 서비스 보안
서비스의 보안은 프로젝트에서 해결하려는 문제만큼 중요하다. 그 이유는 다음과 같다.
1. 해킹을 막아준다.
2. 보안이 우수해야 팔린다.
3. 보안 기능을 나중에 넣기는 어렵다.

##### 5.1 서비스 보안의 세 단계
1. 주고받은 데이터는 암호화하여 중간자 공격에 대비한다.
2. 클라이언트를 인증한다.
3. 인증한 클라이언트의 권한을 결정한다.

###### 5.1.1 주고받은 데이터의 암호화
데이터를 암호화하여 주고받으면 중간자 공격을 막을 수 있다. 주고받느 데이터를 중간자 공격으로부터 막아주는, 가장 널리 쓰이는 암호화 방법은 TLS 이다. SSL을 계승한 TLS는 한때 온라인 은행처럼 보안이 매주 중요한 웹사이트에만 필요하다고 여겨졌으나, 이제는 모든 사이트가 TLS를 사용해야 한다는 인식이 자리 잡았다.

클라이언트와 서버의 통신은 TLS 핸드셰이크부터 시작한다. 
1. 사용할 TLS 버전을 명시한다.
2. 사용할 암호화 스위트(cipher suite, 암호화 알고리즘들의 모음)을 결정한다.
3. 서버의 개인 키와 인증 기관의 서명으로 서버를 인증한다.
4. 핸드셰이크가 끝나면 대칭 키 암호화를 위해 세션 키를 생성한다.

TLS 핸드셰이크는 TLS가 알아서 처리한다. 우리는 클라이언트와 서버의 인증서만 준비하면 된다. 이 인증서를 이용해서 gRPC over TLS를 할 수 있다.

TLS 지원을 구현하여 서비스가 주고받는 데이터를 암호화하고 서버를 인증하겠다.

###### 5.1.2 클라이언트 인증
TLS로 클라이언트와 서버 통신의 보안을 강화했으니, 다음은 인증이다. 여기서 인증이란 클라이언트가 누구인지 확인하는 것이다(참고로, 서버는 TLS가 인증했다). 예를 들어 트위터는 클라이언트가 포스팅할 때마다 트윗을 포스팅하는 사람이 본인인지 확인한다.

대부분의 웹 서버시는 TLS를 단방향 인증으로 서버만 인증한다. 클라이언트 인증은 애플리케이션에서 구현할 몫이며, 보통은 사용자명-비밀번호와 토큰의 조합으로 구현한다. TLS 상호 인증(mutual authentication, 양방향 인증 two-way authentication 이라고도 한다.)은 기계 간 통신에 많이 사용한다. 분산 시스템이 대표적이다. 이 경우에는 서버와 클라이언트 모두 인증서를 이용해 자신을 인증해야 한다. TLS 상호 인증은 효과적이고 간단하며 이미 많이 적용되었다. TLS 상호 인증의 사용자가 많은 만큼, 새로운 서비스를 만든다면 이를 지원해야 한다. 

TLS 상호 인증을 구현해보겠다.

###### 5.1.3 클라이언트 권한
인증(authentication)과 권한 결정(인가, authorization)은 깊게 연관되며 둘 다 'auth'라 줄여 불이기도 한다. 또한 인증과 권한은 요청의 생명주기에서 거의 동시에 이루어지며 서버 코드에서도 서로 가까운 곳에 있다. 사실 특정 리소스의 소유자가 하나인 대부분의 웹 서비스에서는 인증과 권한이 하나의 프로세스이다. 예를 들어 트위터 계정은 소유자가 하나이다. 그래서 클라이언트 인증이 되면, 그 계정에서 할 수 있는 활동은 다 할 수 있다.

인증과 권한의 구분은 리소스의 접근을 공유하고 소유권의 레벨이 다양할 때 필요하다. 우리가 만든 로그 서비스를 예로 들면, 앨리스는 로그 소유자이면서 읽고 쓰기 접근 권한을 가지고 있지만 밥은 읽을 권한만 가지는 식이다. 이런 경우에 세분화한 접근 제어 권한이 필요하다.

우리의 서비스에는 목록에 기반한 접근 제어 권한을 구현하겠다. 접근 제어 권한으로 클라이언트에 로그를 읽거나 쓸 권한을 줄지를 제어할 것이다.

##### 5.2 TLS로 서버 인증하기
주고받는 데이터를 암호화하고 서버를 인증할 때 TLS를 사용해보겠다. 인증서를 얻고 사용할 때 더 쉽게 관리하는 법도 다뤄보겠다.

###### 5.2.1 CFSSL로 나만의 CA 작동하기
서버 코드를 바꾸기 전에 인증서부터 준비하겠다. 서드파티 인증 기관(certificate authority, CA)에서 인증서를 받을 수도 있지만 인증 기관에 따라 돈이 들고 꽤 번거롭다. 신뢰할 수 있는 인증서가 필요하다고 반드시 코모도나 레츠인크립트와 같은 회사에서 발급하지 않아도 된다. 직접 만든 CA로 인증서를 발급하면 된다. 적절한 도구만 쓰면 무료로 쉽게 발급할 수 있다.
글로벌 보안업체인 클라우드플레어(CloudFlare)가 만든 CFSSL 툴킷은 TLS 인증서를 서명하고, 증명하며, 묶을 수 있다. 오픈소스라서 누구든 사용할 수 있다.
CFSSL 중에서 두 개의 도구를 사용하겠다.
- cfssl: TLS 인증서를 서명하고, 증명하며, 묶어주고, 그 결과를 JSON으로 내보낸다.
- cfssljson : JSON 출력을 받아서 키, 인증서, CSR, 번들 파일로 나눈다.

다음 명령으로 설치한다. go get -u 명령으로 의존성을 추가해줘야 되는건가? go get, go install 이거 잘 모르겠다.
```
$ go install github.com/cloudflare/cfssl/cmd/cfssl
$ go install gibhub.com/cloudflare/cfssl/cmd/cfssljson
또는
$ sudo apt install golang-cfssl   // 나는 이걸로 설치했다.
Makefile의 gencert를 실행한다.
$ make gencert
```

- 이후의 과정은 Project_Structure를 통해서 설명한다.

##### 5.3 TLS 상호 인증으로 클라이언트 인증하기
TLS를 이용해 연결을 암호화하고 서버를 인증했다. 여기서 더 나아가 TLS 상호 인증(양방향 인증)을 구현해보겠다. 서버 역시 CA를 사용하여 클라이언트를 검증한다. 먼저 클라이언트 인증서가 필요하다. 클라이언트 인증서 또한 cfssl, cfssljson으로 생성할 수 있다.

##### 5.4 ACL로 권한 부여하기
클라이언트 뒤에 누가 있는지를 인증하면, 인증한 누군가의 특정 행위에 대한 권한을 확인하게 된다. 권한이란 누군가가 무엇인가에 접근할 수 있을지를 확인하는 것이다. 권한을 구현하는 가장 간단한 방법은 접근 제어 목록(access control list, ACL)이다. ACL은 규칙 테이블이라 할 수 있는데 각 행은 'A는 B라는 행위를 C라는 대상에 할 수 있다'는 형식의 규칙을 담는다. 예를 들어 이런 규칙이 있다고 하자. 앨리스는 <죄와 벌>책을 읽을 권한이 있다. 여기서 앨리스는 행위의 주체, 읽는 것은 행위, <죄와 벌>책은 대상이다.

ACL은 만들기 쉽다. 맵이나 CSV파일과 같은 테이블일 뿌이며 데이터로 전환할 수 있다. 조금 더 복잡하게 구현한다면 키-값 저장 또는 관계형 데이터베이스에 저장할 수 있다. 여기서는 Casbin이라는 라이브러리를 사용하겠다. ACL을 포함한 다양한 제어 모델에 기반하여 권한을 강제한다. Casbin은 여러 서비스에서 많이 사용하고 테스트했으며 확장할 수도 있다. 

먼저 Casbin 패키지를 추가한다. 작업 디렉터리를 생성한다.
```
$ go get github.com/casbin/casbin
$ mkdir internal/auth
```
이후 코드는 프로젝트 구조를 통해서 설명을 이어간다.