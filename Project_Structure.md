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