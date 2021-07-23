package v1

import (
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"net/http"
)

func (h RegistryAPI) Authenticate(middleware http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.TLS == nil || len(req.TLS.PeerCertificates) == 0 {
			_ = utils.RespondError(w, http.StatusUnauthorized, "is not https or there is no peer certificate")
			return
		}
		middleware.ServeHTTP(w, req)
	})
}
