package server

import (
	"context"
	"net"
	"os"
	"testing"

	api "github.com/sodami-hub/proglog/api/v1"
	"github.com/sodami-hub/proglog/internal/log"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func TestServ(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
		client api.LogClient,
		config *Config,
	){
		"produce/consume a message to/from the log succeeds": testProduceConsume,
		"produce/consume stream succeeds":                    testProduceConsumeStream,
		"consume past log boundary fails":                    testConsumePastBoundary,
	} {
		t.Run(scenario, func(t *testing.T) {
			client, config, teardown := setupTest(t, nil)
			defer teardown()
			fn(t, client, config)
		})
	}
}

/*
setupTest 함수는 각각의 테스트 케이스를 위한 준비를 해주는 도우미 함수이다.
테스트는 서버를 실행할 컴퓨터의 로컬 네트워크 주소를 가진 리스너부터 만든다.
0번 포트로 설정하면 자동으로 사용하지 않는 포트를 할당한다.
다음으로 리스너에 보안을 고려하지 않은 연결을 수행한다. 클라이언트는 이 연결을 사용한다.

이어서 서버를 생성하고 고루틴에서 요청을 처리한다. Serve() 메서드가 블로킹 호출이므로
고루틴에서 호출하지 않으면 이어지는 테스트가 실행되지 않는다.
*/
func setupTest(t *testing.T, fn func(*Config)) (client api.LogClient, cfg *Config, teardown func()) {
	t.Helper()

	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	cc, err := grpc.NewClient(l.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	dir, err := os.MkdirTemp("", "server-test")
	require.NoError(t, err)

	clog, err := log.NewLog(dir, log.Config{})
	require.NoError(t, err)

	cfg = &Config{
		CommitLog: clog,
	}
	if fn != nil {
		fn(cfg)
	}
	server, err := NewGRPCServer(cfg)
	require.NoError(t, err)

	go func() {
		server.Serve(l)
	}()

	client = api.NewLogClient(cc)
	return client, cfg, func() {
		server.Stop()
		cc.Close()
		l.Close()
		clog.Remove()
	}
}

/*
testProduceConsume 테스트는 클라이언트와 서버를 이용해 생산과 소비가 이루어지는지,
로그에 레코드를 추가하고 다시 소비하는지 확인한다.
그리고 보낸 레코드의 오프셋으로 다시 받았을 때 같은 데이터인지 확인한다.
*/
func testProduceConsume(t *testing.T, client api.LogClient, config *Config) {
	ctx := context.Background()

	want := &api.Record{
		Value: []byte("hello world"),
	}

	produce, err := client.Produce(
		ctx,
		&api.ProduceRequest{
			Record: want,
		},
	)
	require.NoError(t, err)

	consume, err := client.Consume(ctx, &api.ConsumeRequest{
		Offset: produce.Offset,
	})
	require.NoError(t, err)
	require.Equal(t, want.Value, consume.Record.Value)
	require.Equal(t, want.Offset, consume.Record.Offset)
}

/*
testConsumePastBoundary 테스트는 클라이언트가 로그의 범위를 벗어난 소비를 시도할 때,
서버가 api.ErrOffsetOutOfRange() 에러를 회신하는지 확인
*/
func testConsumePastBoundary(t *testing.T, client api.LogClient, config *Config) {
	ctx := context.Background()

	produce, err := client.Produce(ctx, &api.ProduceRequest{
		Record: &api.Record{
			Value: []byte("hello world"),
		},
	})
	require.NoError(t, err)

	consume, err := client.Consume(ctx, &api.ConsumeRequest{
		Offset: produce.Offset + 1,
	})
	if consume != nil {
		t.Fatal("consume not nil")
	}

	// /api/v1/error.go 에서 정의한 error에 대해서 클라이언트쪽에서 error 코드를 확인하는 방법 들
	got := status.Code(err)
	want := status.Code(api.ErrOffsetOutOfRange{}.GRPCStatus().Err())
	if got != want {
		t.Fatalf("got err : %v, want: %v", got, want)
	}
}

func testProduceConsumeStream(t *testing.T, client api.LogClient, config *Config) {
	ctx := context.Background()

	records := []*api.Record{{
		Value:  []byte("first message"),
		Offset: 0,
	}, {
		Value:  []byte("second message"),
		Offset: 1,
	}}

	{
		stream, err := client.ProduceStream(ctx)
		require.NoError(t, err)

		for offset, record := range records {
			err = stream.Send(&api.ProduceRequest{
				Record: record,
			})
			require.NoError(t, err)
			res, err := stream.Recv()
			require.NoError(t, err)
			if res.Offset != uint64(offset) {
				t.Fatalf("got offset : %d, want : %d", res.Offset, offset)
			}
		}
	}
	{
		stream, err := client.ConsumeStream(ctx, &api.ConsumeRequest{Offset: 0})
		require.NoError(t, err)

		for i, record := range records {
			res, err := stream.Recv()
			require.NoError(t, err)
			require.Equal(t, res.Record, &api.Record{
				Value:  record.Value,
				Offset: uint64(i),
			})
		}
	}
}
