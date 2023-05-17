package server

import (
	"time"

	"github.com/poisonbian/hersql/config"
	"github.com/poisonbian/hersql/log"
	"github.com/dolthub/go-mysql-server/auth"
	"github.com/dolthub/vitess/go/mysql"
	"go.uber.org/zap"
)

type Server struct {
	Listener *mysql.Listener
	h        *Handler
	logger   *zap.SugaredLogger
}

func NewServer(conf *config.Conf) (*Server, error) {
	logger := log.GetLogger(conf.Log)
	handler := NewHandler(time.Duration(conf.Server.ConnReadTimeout)*time.Millisecond, conf, logger, NewSessionManager())

	l, err := NewListener(conf.Server.Protocol, conf.Server.Address, handler)
	if err != nil {
		return nil, err
	}

	listenerCfg := mysql.ListenerConfig{
		Listener:           l,
		AuthServer:         auth.NewNativeSingle(conf.Server.UserName, conf.Server.UserPassword, auth.AllPermissions).Mysql(),
		Handler:            handler,
		ConnReadTimeout:    time.Duration(conf.Server.ConnReadTimeout) * time.Millisecond,
		ConnWriteTimeout:   time.Duration(conf.Server.ConnWriteTimeout) * time.Millisecond,
		MaxConns:           conf.Server.MaxConnections,
		ConnReadBufferSize: mysql.DefaultConnBufferSize,
	}
	vtListnr, err := mysql.NewListenerWithConfig(listenerCfg)
	if err != nil {
		return nil, err
	}
	if conf.Server.Version != "" {
		vtListnr.ServerVersion = conf.Server.Version
	}
	vtListnr.TLSConfig = nil
	vtListnr.RequireSecureTransport = false

	return &Server{Listener: vtListnr, h: handler, logger: logger}, nil
}

func (s *Server) Start() error {
	s.logger.Infof("hersql starting server on [%s]", s.Listener.Addr())
	s.Listener.Accept()
	return nil
}

func (s *Server) Close() error {
	s.logger.Infof("hersql server close")
	s.Listener.Close()
	return nil
}
