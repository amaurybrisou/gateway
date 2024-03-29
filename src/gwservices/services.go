package gwservices

import (
	"github.com/amaurybrisou/ablib/jwtlib"
	"github.com/amaurybrisou/ablib/mailcli"
	"github.com/amaurybrisou/gateway/src/database"
	"github.com/amaurybrisou/gateway/src/gwservices/gwservice"
	"github.com/amaurybrisou/gateway/src/gwservices/payment"
	"github.com/amaurybrisou/gateway/src/gwservices/proxy"
)

type Services struct {
	jwt     *jwtlib.JWT
	svc     gwservice.Service
	proxy   proxy.Proxy
	payment payment.Service
}

func (s Services) Jwt() *jwtlib.JWT {
	return s.jwt
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
	JwtConfig     jwtlib.Config
	ProxyConfig   proxy.Config
}

func NewServices(db *database.Database, mail *mailcli.MailClient, cfg ServiceConfig) Services {
	jwt := jwtlib.New(cfg.JwtConfig)

	return Services{
		jwt:     jwt,
		svc:     gwservice.New(db, jwt),
		proxy:   proxy.New(db, cfg.ProxyConfig),
		payment: payment.NewService(db, jwt, mail, cfg.PaymentConfig),
	}
}
