package trust

import (
	"fmt"
	"net/http"
)

type RegistryTransport struct {
	Base  http.RoundTripper
	Token string
}

func (t *RegistryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := cloneRequest(req)
	req2.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.Token))

	baseResp, err := t.Base.RoundTrip(req2)
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
