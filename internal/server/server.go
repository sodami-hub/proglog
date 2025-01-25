package server

import (
	"context"

	api "github.com/sodami-hub/proglog/api/v1"
	"google.golang.org/grpc"
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
	CommitLog CommitLog
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
log_grpc.pb.go 파일의 API를 구현하려면 Consume()과 Prologduce() 핸들러를 구현해야 된다. gRPC 계층은 구현이 복잡하지 않다.
로그 라이브러리를 호출하고 에러를 처리하는 것이 전부이다.

아래 서버의 두 메서드는 서버의 로그를 생성하고 불러오라는 클라이언트의 요청을 처리한다.
*/
func (s *grpcServer) Produce(ctx context.Context, req *api.ProduceRequest) (*api.ProduceResponse, error) {
	offset, err := s.CommitLog.Append(req.Record)
	if err != nil {
		return nil, err
	}
	return &api.ProduceResponse{Offset: offset}, nil
}

func (s *grpcServer) Consume(ctx context.Context, req *api.ConsumeRequest) (*api.ConsumeResponse, error) {
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

func NewGRPCServer(config *Config) (*grpc.Server, error) {
	gsrv := grpc.NewServer()
	srv, err := newgrpcServer(config)
	if err != nil {
		return nil, err
	}
	api.RegisterLogServer(gsrv, srv)
	return gsrv, nil
}
