package v1

import (
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"net/http"
)

const (
	UserHeader   = "X-Remote-User"
	GroupHeader  = "X-Remote-Group"
	ExtrasHeader = "X-Remote-Extra-"
)

func Authenticate(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.TLS == nil || len(req.TLS.PeerCertificates) == 0 {
			_ = utils.RespondError(w, http.StatusUnauthorized, "is not https or there is no peer certificate")
			return
		}
		h.ServeHTTP(w, req)
	})
}
