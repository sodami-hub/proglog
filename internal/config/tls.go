package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

/*
테스트마다 조금씩 다른 *tls.Config 설정을 사용한다. SetupTLSConfig 함수는 테스트마다 요청하는 자료형에 맞는 *tls.Config를 리턴한다.
  - 클라이언트의 *tls.Config는 서버 인증서를 검증하는 설정으로 *tls.Config의 RootCAs를 설정한다.
  - 클라이언트의 *tls.Config는 서버 인증서와 검증과 함께, 서버가 클라이언트 인증서를 검증할 수 있게 RootCAS와 Certificates를 설정한다.
  - 서버의 *tls.Config는 클라이언트 인증서의 검증과 함께, 클라이언트가 서버 인증서를 검증할 수 있게 ClientCAs, Certificates를 설정하고
    ClientAuth 모드를 tls.RequireAndVerifyClientCert로 설정한다.
*/
func SetupTLSConfig(cfg TLSConfig) (*tls.Config, error) {
	var err error
	tlsConfig := &tls.Config{}
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		tlsConfig.Certificates = make([]tls.Certificate, 1)
		tlsConfig.Certificates[0], err = tls.LoadX509KeyPair(
			cfg.CertFile,
			cfg.KeyFile,
		)
		if err != nil {
			return nil, err
		}
	}
	if cfg.CAFile != "" {
		b, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, err
		}
		ca := x509.NewCertPool()
		ok := ca.AppendCertsFromPEM([]byte(b))
		if !ok {
			return nil, fmt.Errorf("failed to parse root certificate: %q", cfg.CAFile)
		}
		if cfg.Server {
			tlsConfig.ClientCAs = ca
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		} else {
			tlsConfig.RootCAs = ca
		}
		tlsConfig.ServerName = cfg.ServerAddress
	}
	return tlsConfig, nil
}

// 구조체 추가
// TLSConfig는 SetupTLSConfig 함수가 사용할 매개변수이며, 이 설정에 따라 그에 맞는 *tls.Config를 리턴한다.
type TLSConfig struct {
	CertFile      string
	KeyFile       string
	CAFile        string
	ServerAddress string
	Server        bool
}
