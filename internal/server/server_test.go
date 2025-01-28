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
	"google.golang.org/grpc/status"

	// ../config/tls.go 의 테스트를 위한 패키지 임포트
	"github.com/sodami-hub/proglog/internal/config"
	"google.golang.org/grpc/credentials"
)

func TestServ(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
		//client api.LogClient,
		// 멀티 클라이언트의 권한 테스트를 위한 클라이언트 매개변수 변경
		rootClient api.LogClient,
		nobodyClient api.LogClient,
		config *Config,
	){
		"produce/consume a message to/from the log succeeds": testProduceConsume,
		"produce/consume stream succeeds":                    testProduceConsumeStream,
		"consume past log boundary fails":                    testConsumePastBoundary,
	} {
		t.Run(scenario, func(t *testing.T) {
			/*client,*/ rootClient, nobodyClient, config, teardown := setupTest(t, nil)
			defer teardown()
			fn(t, rootClient, nobodyClient, config)
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
func setupTest(t *testing.T, fn func(*Config)) (rootClient api.LogClient, nobodyClient api.LogClient, cfg *Config, teardown func()) {

	// =============================== 일반적인 테스트(TLS 없이 insecure 모드로 연결) ========================================================
	// t.Helper()

	// l, err := net.Listen("tcp", ":0")
	// require.NoError(t, err)

	// cc, err := grpc.NewClient(l.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	// require.NoError(t, err)

	// dir, err := os.MkdirTemp("", "server-test")
	// require.NoError(t, err)

	// clog, err := log.NewLog(dir, log.Config{})
	// require.NoError(t, err)

	// cfg = &Config{
	// 	CommitLog: clog,
	// }
	// if fn != nil {
	// 	fn(cfg)
	// }
	// server, err := NewGRPCServer(cfg)
	// require.NoError(t, err)

	// go func() {
	// 	server.Serve(l)
	// }()

	// client = api.NewLogClient(cc)
	// return client, cfg, func() {
	// 	server.Stop()
	// 	cc.Close()
	// 	l.Close()
	// 	clog.Remove()
	// }

	// ========================= tls를 사용한 설정으로 코드를 변경한다.(서버 인증서로 인증!!) ======================================================================
	// t.Helper()

	// // 클라이언트의 TLS 인증서가 서버에서 만든 CA를 클라이언트의 Root Ca로, 다시 말해 서버의 인증서를 검증할 때 사용하도록 설정했다.
	// // 그리고 클라이언트는 이 인증서를 사용해서 연결한다.
	// l, err := net.Listen("tcp", "127.0.0.1:0")
	// require.NoError(t, err)
	// clientTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
	// 	CAFile: config.CAFile,
	// })
	// require.NoError(t, err)

	// clientCreds := credentials.NewTLS(clientTLSConfig)
	// cc, err := grpc.NewClient(
	// 	l.Addr().String(), grpc.WithTransportCredentials(clientCreds),
	// )
	// require.NoError(t, err)

	// client = api.NewLogClient(cc)

	// // 서버에 인증서를 넣어서 TLS 연결을 처리하겠다.
	// /*
	// 	서버의 인증서와 키를 파싱했다. 그리고 서버의 TLS 인증서를 설정할 때 사용했다.
	// 	이렇게 만든 인증서를 NewGRPCServer() 함수의 gRPC 서버 옵션으로 전달해서 gRPC 서버를 만들었다.
	// 	gRPC 서버 옵션으로는 gRPC 서버의 여러 기능을 활성화할 수 있다. 여기서는 서버 연결을 위한 인증서 설정을 사용했다.
	// 	그 외에도 연결의 타임아웃이나 keep alive 정책 등 다양한 서버 옵션을 설정할 수 있다.
	// */
	// serverTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
	// 	CertFile:      config.ServerCertFile,
	// 	KeyFile:       config.ServerKeyFile,
	// 	CAFile:        config.CAFile,
	// 	ServerAddress: l.Addr().String(),
	// })

	// require.NoError(t, err)
	// serverCreds := credentials.NewTLS(serverTLSConfig)

	// dir, err := os.MkdirTemp("", "server-test")
	// require.NoError(t, err)

	// clog, err := log.NewLog(dir, log.Config{})
	// require.NoError(t, err)

	// cfg = &Config{
	// 	CommitLog: clog,
	// }
	// if fn != nil {
	// 	fn(cfg)
	// }

	// // server.go의 NewGRPCServer() 함수의 매개변수를 수정해야 된다.
	// server, err := NewGRPCServer(cfg, grpc.Creds(serverCreds))
	// require.NoError(t, err)

	// go func() {
	// 	server.Serve(l)
	// }()

	// return client, cfg, func() {
	// 	server.Stop()
	// 	cc.Close()
	// 	l.Close()
	// 	os.RemoveAll(dir)
	// }

	// ========================================================= TLS 상호 인증 테스트 =======================================================================
	// t.Helper()

	// // Client의 인증서를 서버의 CA로 만들었기 때문에 통과할 것이다. 서버와 클라이언트는 서로의 인증서에 대해 CA로 TLS 상호 인증을 했다.
	// // 서버는 중간자의 도청 걱정 없이 실제 클라이언트와 안심하고 통신한다.
	// l, err := net.Listen("tcp", "127.0.0.1:0")
	// require.NoError(t, err)
	// clientTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
	// 	CertFile: config.ClientCertFile,
	// 	KeyFile:  config.ClientKeyFile,
	// 	CAFile:   config.CAFile,
	// })
	// require.NoError(t, err)

	// clientCreds := credentials.NewTLS(clientTLSConfig)
	// cc, err := grpc.NewClient(
	// 	l.Addr().String(), grpc.WithTransportCredentials(clientCreds),
	// )
	// require.NoError(t, err)

	// client = api.NewLogClient(cc)

	// serverTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
	// 	CertFile:      config.ServerCertFile,
	// 	KeyFile:       config.ServerKeyFile,
	// 	CAFile:        config.CAFile,
	// 	ServerAddress: l.Addr().String(),
	// 	Server:        true,
	// })

	// require.NoError(t, err)
	// serverCreds := credentials.NewTLS(serverTLSConfig)

	// dir, err := os.MkdirTemp("", "server-test")
	// require.NoError(t, err)

	// clog, err := log.NewLog(dir, log.Config{})
	// require.NoError(t, err)

	// cfg = &Config{
	// 	CommitLog: clog,
	// }
	// if fn != nil {
	// 	fn(cfg)
	// }

	// // server.go의 NewGRPCServer() 함수의 매개변수를 수정해야 된다.
	// server, err := NewGRPCServer(cfg, grpc.Creds(serverCreds))
	// require.NoError(t, err)

	// go func() {
	// 	server.Serve(l)
	// }()

	// return client, cfg, func() {
	// 	server.Stop()
	// 	cc.Close()
	// 	l.Close()
	// 	os.RemoveAll(dir)
	// }

	// ================================================== ACL(권한) 테스트를 위한 멀티 클라이언트 ==============================

	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	/*
		두 개의 클라이언트를 만들었다. root는 생산과 소비를 할 수 있다. nobody 클라이언트는 아무런 권한이 없다.
		두 클라이언트를 생성하는 코드는 인증서와 키를 제외하고는 다르지 않다.
		서버는 Authorizer 인스턴스를 받는데 서버의 권한 로직을 맡는다.
	*/

	newClient := func(crtPath, keyPath string) (*grpc.ClientConn, api.LogClient, []grpc.DialOption) {
		tlsConfig, err := config.SetupTLSConfig(config.TLSConfig{
			CertFile: crtPath,
			KeyFile:  keyPath,
			CAFile:   config.CAFile,
			Server:   false,
		})
		require.NoError(t, err)
		tlsCreds := credentials.NewTLS(tlsConfig)
		opts := []grpc.DialOption{grpc.WithTransportCredentials(tlsCreds)}
		conn, err := grpc.NewClient(l.Addr().String(), opts...)
		require.NoError(t, err)
		client := api.NewLogClient(conn)
		return conn, client, opts
	}

	var rootConn *grpc.ClientConn
	rootConn, rootClient, _ = newClient(
		config.RootClientCertFile,
		config.RootClientKeyFile,
	)

	var nobodyConn *grpc.ClientConn
	nobodyConn, nobodyClient, _ = newClient(
		config.NobodyClientCertFile,
		config.NobodyClientKeyFile,
	)

	serverTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile:      config.ServerCertFile,
		KeyFile:       config.ServerKeyFile,
		CAFile:        config.CAFile,
		ServerAddress: l.Addr().String(),
		Server:        true,
	})

	require.NoError(t, err)
	serverCreds := credentials.NewTLS(serverTLSConfig)

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

	// server.go의 NewGRPCServer() 함수의 매개변수를 수정해야 된다.
	server, err := NewGRPCServer(cfg, grpc.Creds(serverCreds))
	require.NoError(t, err)

	go func() {
		server.Serve(l)
	}()

	return rootClient, nobodyClient, cfg, func() {
		server.Stop()
		rootConn.Close()
		nobodyConn.Close()
		l.Close()
	}
}

/*
testProduceConsume 테스트는 클라이언트와 서버를 이용해 생산과 소비가 이루어지는지,
로그에 레코드를 추가하고 다시 소비하는지 확인한다.
그리고 보낸 레코드의 오프셋으로 다시 받았을 때 같은 데이터인지 확인한다.
*/
func testProduceConsume(t *testing.T, client, _ api.LogClient, config *Config) {
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
func testConsumePastBoundary(t *testing.T, client, _ api.LogClient, config *Config) {
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

func testProduceConsumeStream(t *testing.T, client, _ api.LogClient, config *Config) {
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
