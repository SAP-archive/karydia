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
}

type Config struct {
	Addr string

	Logger *logrus.Logger

	TLSConfig *tls.Config
}

func New(config *Config, webhook *webhook.Webhook) (*Server, error) {
	server := &Server{
		config: config,
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
	if webhook != nil {
		mux.HandleFunc("/webhook", webhook.Serve)
	}

	httpServer := &http.Server{
		Addr:      config.Addr,
		Handler:   server.middlewareLogger(mux),
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

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (s *Server) middlewareLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{w, http.StatusOK}
		next.ServeHTTP(rw, r)
		s.logger.WithFields(logrus.Fields{
			"remote_addr": r.RemoteAddr,
			"method":      r.Method,
			"status":      rw.status,
		}).Infof("%s", r.URL)
	})
}
