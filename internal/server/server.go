package server

import (
	"context"

	api "github.com/sodami-hub/proglog/api/v1"
	"google.golang.org/grpc"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
)

/*
현재 gRPC서버는 log 패키지와 연결된 상태는 아니다. 하지만 CommitLog 인터페이스가 구현된 로그 패키지는 무엇이든 사용할 수 있다.
예를 들어 이렇게 연결할 수 있다.

newgrpcServer 함수를 보면 Config를 전달하는데 여기에 log 패키지의 인스턴스를 전달하면된다.

config := &NewLog("/temp/log", log.Config{})

srv, err := newgrpcServer(config)
*/
type CommitLog interface {
	Append(*api.Record) (uint64, error)
	Read(uint64) (*api.Record, error)
}

type Config struct {
	CommitLog  CommitLog
	Authorizer Authorizer // 권한에 사용할 필드
}

// 권한에 사용할 상수들. 이 상수들은 ACL 정책 테이블의 값과 매칭된다. 여러번 참조하기 때문에 상수로 정의했다.
const (
	objextWildcard = "*"
	produceAction  = "produce"
	consumeAction  = "consume"
)

// Config의 Authorizer필드는 인터페이스다.
type Authorizer interface {
	Authorize(subject, object, action string) error
}

var _ api.LogServer = (*grpcServer)(nil)

type grpcServer struct {
	api.UnimplementedLogServer
	*Config
}

func newgrpcServer(config *Config) (srv *grpcServer, err error) {
	srv = &grpcServer{
		Config: config,
	}
	return srv, nil
}

/*
이제 서버는 클라이언트를 인증서의 주체로 인증하여, 생산과 소비의 권한이 있는지 확인한다. 만약 권한이 없다면 허가가 거부되었다는 에러를 회신한다.
생산및 소비를 요청하는 클라이언트가 권한이 있다면 메서드는 레코드를 로그에 추가 및 전달한다.
클라이언트 인증서에서 주체를 얻어내려면 두 개의 도우미 함수가 필요하다.

authenticate(ctx context.Context) 함수는 클라이언트 인증서에서 주체를 읽어서 RPC의 콘텍스트에 쓴다.
이러한 함수를 인터셉터라고 하는데, 각각의 RPC 호출을 가로채고 변경해서 요청 처리를 작고 재사요여 가능한 단위로 나눈다.
다른 프레임워크에서는 이러한 개념을 미들웨어라고 부르기도 한다.
subject() 함수는 클라이언트 인증서의 주체를 리턴하여 클라이언트를 인식하고 접근 가능 여부를 확인한다.
*/
func authenticate(ctx context.Context) (context.Context, error) {
	peer, ok := peer.FromContext(ctx)
	if !ok {
		return ctx, status.New(
			codes.Unknown,
			"couldn't find peer info",
		).Err()
	}

	if peer.AuthInfo == nil {
		return context.WithValue(ctx, subjectContextKey{}, ""), nil
	}
	tlsInfo := peer.AuthInfo.(credentials.TLSInfo)
	subject := tlsInfo.State.VerifiedChains[0][0].Subject.CommonName
	ctx = context.WithValue(ctx, subjectContextKey{}, subject)

	return ctx, nil
}

func subject(ctx context.Context) string {
	return ctx.Value(subjectContextKey{}).(string)
}

type subjectContextKey struct{}

/*
log_grpc.pb.go 파일의 API를 구현하려면 Consume()과 Prologduce() 핸들러를 구현해야 된다. gRPC 계층은 구현이 복잡하지 않다.
로그 라이브러리를 호출하고 에러를 처리하는 것이 전부이다.

아래 서버의 두 메서드는 서버의 로그를 생성하고 불러오라는 클라이언트의 요청을 처리한다.
*/
func (s *grpcServer) Produce(ctx context.Context, req *api.ProduceRequest) (*api.ProduceResponse, error) {
	// 권한 확인
	if err := s.Authorizer.Authorize(subject(ctx), objextWildcard, produceAction); err != nil {
		return nil, err
	}

	offset, err := s.CommitLog.Append(req.Record)
	if err != nil {
		return nil, err
	}
	return &api.ProduceResponse{Offset: offset}, nil
}

func (s *grpcServer) Consume(ctx context.Context, req *api.ConsumeRequest) (*api.ConsumeResponse, error) {

	// 권한확인
	if err := s.Authorizer.Authorize(subject(ctx), objextWildcard, consumeAction); err != nil {
		return nil, err
	}

	record, err := s.CommitLog.Read(req.Offset)
	if err != nil {
		return nil, err
	}

	return &api.ConsumeResponse{Record: record}, nil
}

// 스트리밍 API

// ProduceStream 메서드는 양방향 스트리밍 RPC이다. 클라이언트는 서버의 로그로 데이터를 스트리밍할 수 있고,
// 서버는 각 요청의 성공 여부를 회신할 수 있다.
func (s *grpcServer) ProduceStream(stream api.Log_ProduceStreamServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}

		res, err := s.Produce(stream.Context(), req)
		if err != nil {
			return err
		}

		if err = stream.Send(res); err != nil {
			return err
		}
	}
}

// 서버측 스트리밍 RPC이다. 클라이언트가 로그의 어느 위치의 레코드를 읽고 싶은지 밝히면, 서버는 그 위치부터 이어지는 모든 레코드를 스트리밍한다.
// 나아가 서버가 로그 끝까지 스트리밍하면 레코드의 변화가 생길 때마다 클라이언트에 스트리밍한다.
func (s *grpcServer) ConsumeStream(req *api.ConsumeRequest, stream api.Log_ConsumeStreamServer) error {
	for {
		select {
		case <-stream.Context().Done():
			return nil
		default:
			res, err := s.Consume(stream.Context(), req)
			switch err.(type) {
			case nil:
			case api.ErrOffsetOutOfRange:
				continue
			default:
				return err
			}
			if err = stream.Send(res); err != nil {
				return err
			}
			req.Offset++
		}
	}
}

func NewGRPCServer(config *Config, opts ...grpc.ServerOption) (*grpc.Server, error) {

	// 미들웨어를 통한 권한 확인 : authenticate 함수를 gRPC 서버에 연결해서 서버가 각각의 RPC의 주체를 확인하고 권한을 확인한다.
	opts = append(opts,
		grpc.StreamInterceptor(
			grpc_middleware.ChainStreamServer(
				grpc_auth.StreamServerInterceptor(authenticate))),
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(
				grpc_auth.UnaryServerInterceptor(authenticate))),
	)

	gsrv := grpc.NewServer(opts...)
	srv, err := newgrpcServer(config)
	if err != nil {
		return nil, err
	}
	api.RegisterLogServer(gsrv, srv)
	return gsrv, nil
}
