# 프로젝트 구조
## PART I
#### 1. 프로젝트 기본 세팅
1. 커밋 로그 프로토타입 만들기[/LetsGo/internal/server/log.go]
2. HTTP의 JSON 만들기[/LetsGo/internal/server/http.go]
#### 2. 프로토콜 버퍼와 구조체
3. 프로토콜 버퍼로 도메인 자료형 정의하기[/StructureDataWithProtobuf/api/v1/log.proto]
- [/LetsGo/internal/server/log.go] 의 Record 자료형을 protobuf 메시지로 문법에 맞게 바꿔준다.
4. protobuf가 바뀔 때마다 컴파일해야 하므로 Makefile 파일에 compile 이라는 타깃을 만들어두면 편리하다. [/StructureDataWithProtobuf]에 Makefile을 만든다.
#### 3. 로그 패키지 작성
5. 스토어 만들기 : 로그 패키지를 위한 [/internal/log/store.go] 코드를 작성한다. store 구조체는 활성 세그먼트 파일에 접근하는 포인터 필드를 포함한 해당 저장파일에 대한 정보를 가진다. store 구조체를 통해서 레코드를 파일에 기록한다.(로그를 기록할 때 해당 로그의 위치(pos)를 반환한다.) - [data = pos+8(로그 데이터의 길이) / pos+8 ~ data, 실제 데이터]로 구성된다.
6. 인덱스 만들기 : [internal/log/index.go]  - 인덱스는 0부터 숱서대로 로그의 순서를 붙이고(인덱스 오프셋), 해당 인덱스의 위치(pos)를 저장한다. pos값을 알면 실제 로그 기록을 찾을 수 있다.
7. 세그먼트 만들기 : [internal/log/segment.go] 세그먼트는 스토어와 인덱스를 감싸고 둘 사이의 작업을 조율한다. 예를 들어 로그가 활성 세그먼트에 레코드를 추가할 때, 세그먼트는 데이터를 스토어에 쓰고 새로운 인덱스 항목을 인덱스에 추가한다. 읽을 때도 마찬가지이다. 세그먼트는 인덱스에서 인덱스 항목을 찾고 스토어에서 데이터를 가져온다.
8. 로그의 구현: [internal/log/log.go]