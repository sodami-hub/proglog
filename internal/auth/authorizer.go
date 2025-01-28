package auth

import (
	"fmt"

	"github.com/casbin/casbin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func New(model, policy string) *Authorizer {
	enforcer := casbin.NewEnforcer(model, policy)
	return &Authorizer{
		enforcer: enforcer,
	}
}

type Authorizer struct {
	enforcer *casbin.Enforcer
}

/*
Authorizer 구조체와 Authorizer() 메서드를 정의했다. 이 메서드는 Casbin의 Enforce() 함수를 사용한다.
Enforce() 함수는 특정한 주체가 특정 행위를 특정 대상에 할 수 있는지를 Casbin에 설정한 모델과 정책에 기반해서
확인하여 알려준다. New() 함수의 모델과 정책 매개변수는 이들을 정의한 파일의 경로이다.

모델을 정의하는 파일은 Casbin의 권한 메커니즘을 설정하며 여기서 사용할 모델은 ACL이다.
정책의 경우는 ACL 테이블을 담은 CSV 파일이다.
*/
func (a *Authorizer) Authorize(subject, object, action string) error {
	if !a.enforcer.Enforce(subject, object, action) {
		msg := fmt.Sprintf(
			"%s not permitted to %s to %s",
			subject,
			action,
			object,
		)
		st := status.New(codes.PermissionDenied, msg)
		return st.Err()
	}
	return nil
}
