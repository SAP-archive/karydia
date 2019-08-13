// Copyright (C) 2019 SAP SE or an SAP affiliate company. All rights reserved.
// This file is licensed under the Apache Software License, v. 2 except as
// noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	"crypto/tls"
	"github.com/karydia/karydia/pkg/logger"
	"net/http"

	"github.com/karydia/karydia"
	"github.com/karydia/karydia/pkg/webhook"
)

type Server struct {
	config *Config

	logger *logger.Logger

	httpServer *http.Server
}

type Config struct {
	Addr string

	Logger *logger.Logger

	TLSConfig *tls.Config
}

func New(config *Config, webhook *webhook.Webhook) (*Server, error) {
	server := &Server{
		config: config,
	}

	if config.Logger == nil {
		server.logger = logger.NewComponentLogger("server")
	} else {
		// convenience
		server.logger = config.Logger
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", server.handlerHealthz)
	if webhook != nil {
		mux.HandleFunc("/webhook/validating", func(w http.ResponseWriter, r *http.Request) {
			webhook.Serve(w, r, false)
		})
		mux.HandleFunc("/webhook/mutating", func(w http.ResponseWriter, r *http.Request) {
			webhook.Serve(w, r, true)
		})
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
	s.logger.Infof("karydia server version: %s", karydia.Version)
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
		s.logger.Infof("remote_addr: %s; method: %s; status: %s; url: %s", r.RemoteAddr, r.Method, rw.status, r.URL)
	})
}
