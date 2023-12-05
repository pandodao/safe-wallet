package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pandodao/safe-wallet/handler/api/reversetwirp"
)

type Server struct {
	rt *reversetwirp.ReverseTwirp
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()

	r.Route("transfers", func(r chi.Router) {
		r.Get("/{trace_id}", s.rt.Handle("FindTransfer", nil))
		r.Post("/", s.rt.Handle("CreateTransfer", nil))
	})

	return r
}
