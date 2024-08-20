package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pandodao/safe-wallet/handler/api/reversetwirp"
	"github.com/pandodao/safe-wallet/handler/rpc"
)

func New(svr *rpc.Server) *Server {
	path, handler := svr.Handler()
	return &Server{
		rt: &reversetwirp.ReverseTwirp{
			Target: handler,
			Path:   path,
		},
	}
}

type Server struct {
	rt *reversetwirp.ReverseTwirp
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()

	r.Route("/transfers", func(r chi.Router) {
		r.Get("/{trace_id}", s.rt.Handle("FindTransfer", nil))
		r.Post("/", s.rt.Handle("CreateTransfer", nil))
	})

	r.Route("/wallets", func(r chi.Router) {
		r.Post("/", s.rt.Handle("CreateWallet", nil))
		r.Get("/{user_id}", s.rt.Handle("FindWallet", nil))
	})

	return r
}
