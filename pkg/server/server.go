package server

import (
	"context"
	"crypto/tls"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/kinvolk/karydia/pkg/webhook"
)

type Server struct {
	config *Config

	logger *logrus.Logger

	httpServer *http.Server

	webhook *webhook.Webhook
}

type Config struct {
	Addr string

	Logger *logrus.Logger

	TLSConfig *tls.Config
}

func New(config *Config) (*Server, error) {
	webHook, err := webhook.New(&webhook.Config{})
	if err != nil {
		return nil, err
	}

	server := &Server{
		config:  config,
		webhook: webHook,
	}

	if config.Logger == nil {
		server.logger = logrus.New()
		server.logger.Level = logrus.InfoLevel
	} else {
		// convenience
		server.logger = config.Logger
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", server.handlerHealthz)
	mux.HandleFunc("/webhook", server.webhook.Serve)

	httpServer := &http.Server{
		Addr:      config.Addr,
		Handler:   mux,
		TLSConfig: config.TLSConfig,
	}

	server.httpServer = httpServer

	return server, nil
}

func (s *Server) ListenAndServe() error {
	s.logger.Infof("Listening on %s", s.config.Addr)
	return s.httpServer.ListenAndServeTLS("", "")
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Infof("Shutting down ...")
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) handlerHealthz(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}
