package goravel

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (grv *Goravel) routes() http.Handler {
	mux := chi.NewRouter()
	mux.Use(middleware.RequestID)
	mux.Use(middleware.RealIP)
	mux.Use(middleware.Recoverer)
	mux.Use(grv.SessionLoad)
	mux.Use(grv.NoSurf)

	if grv.Debug {
		mux.Use(middleware.Logger)
	}

	return mux
}
