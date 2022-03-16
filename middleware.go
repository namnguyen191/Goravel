package goravel

import (
	"net/http"
	"strconv"

	"github.com/justinas/nosurf"
)

func (grv *Goravel) SessionLoad(next http.Handler) http.Handler {
	return grv.Session.LoadAndSave(next)
}

func (grv *Goravel) NoSurf(next http.Handler) http.Handler {
	csrfHandler := nosurf.New(next)
	secure, _ := strconv.ParseBool(grv.config.cookie.secure)

	csrfHandler.ExemptGlob("/api/*")

	csrfHandler.SetBaseCookie(http.Cookie{
		HttpOnly: true,
		Path:     "/",
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		Domain:   grv.config.cookie.domain,
	})

	return csrfHandler
}
