package auth

import (
	"fmt"
	"net/http"
)

type RegistryTransport struct {
	Base  http.RoundTripper
	Token *Token
}

func (t *RegistryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clonedReq := cloneRequest(req)
	if t.Token != nil {
		clonedReq.Header.Set("Authorization", fmt.Sprintf("%s %s", t.Token.Type, t.Token.Value))
	}

	baseResp, err := t.Base.RoundTrip(clonedReq)
	if err != nil {
		return nil, err
	}

	return baseResp, err
}

// cloneRequest returns a clone of the provided *http.Request.
// The clone is a shallow copy of the struct and its Header map.
func cloneRequest(r *http.Request) *http.Request {
	// shallow copy of the struct
	r2 := new(http.Request)
	*r2 = *r
	// deep copy of the Header
	r2.Header = make(http.Header, len(r.Header))
	for k, s := range r.Header {
		r2.Header[k] = append([]string(nil), s...)
	}

	return r2
}
