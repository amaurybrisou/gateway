package services

import (
	"github.com/amaurybrisou/gateway/internal/db"
	"github.com/amaurybrisou/gateway/internal/services/gwservice"
	"github.com/amaurybrisou/gateway/internal/services/oauth"
	"github.com/amaurybrisou/gateway/internal/services/proxy"
	"github.com/amaurybrisou/gateway/internal/services/public"
	coremiddleware "github.com/amaurybrisou/gateway/pkg/http/middleware"
)

type Services struct {
	oauth          oauth.Service
	public         public.Service
	svc            gwservice.Service
	proxy          proxy.Proxy
	authMiddleware coremiddleware.AuthMiddlewareService
}

func (s Services) Oauth() oauth.Service {
	return s.oauth
}

func (s Services) Public() public.Service {
	return s.public
}

func (s Services) Service() gwservice.Service {
	return s.svc
}

func (s Services) Proxy() proxy.Proxy {
	return s.proxy
}

// func (s Services) AuthMiddleware() coremiddleware.Service {
// 	return s.authMiddleware
// }

func NewServices(db *db.Database) Services {
	return Services{
		oauth:  oauth.New(db),
		public: public.New(db),
		svc:    gwservice.New(db),
		proxy:  proxy.New(db),
	}
}
