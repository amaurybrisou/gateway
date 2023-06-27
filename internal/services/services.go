package services

import (
	"github.com/amaurybrisou/gateway/internal/db"
	"github.com/amaurybrisou/gateway/internal/services/gwservice"
	"github.com/amaurybrisou/gateway/internal/services/oauth"
	"github.com/amaurybrisou/gateway/internal/services/payment"
	"github.com/amaurybrisou/gateway/internal/services/proxy"
	"github.com/amaurybrisou/gateway/internal/services/public"
)

type Services struct {
	oauth   oauth.Service
	public  public.Service
	svc     gwservice.Service
	proxy   proxy.Proxy
	payment payment.Service
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

func (s Services) Payment() payment.Service {
	return s.payment
}

type ServiceConfig struct {
	PaymentConfig payment.Config
	GoogleConfig  oauth.Config
}

func NewServices(db *db.Database, cfg ServiceConfig) Services {
	return Services{
		oauth:   oauth.New(db, cfg.GoogleConfig),
		public:  public.New(db),
		svc:     gwservice.New(db),
		proxy:   proxy.New(db),
		payment: *payment.NewService(db, cfg.PaymentConfig),
	}
}
