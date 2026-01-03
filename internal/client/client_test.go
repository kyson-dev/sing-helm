package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestClient(handler http.Handler) *Client {
	return &Client{
		baseURL: "http://test",
		httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				recorder := httptest.NewRecorder()
				handler.ServeHTTP(recorder, req)
				resp := recorder.Result()
				resp.Request = req
				return resp, nil
			}),
		},
	}
}

func TestGetProxies(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/proxies", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		resp := proxiesResponse{
			Proxies: map[string]ProxyData{
				"Proxy": {
					Type: "Selector",
					All:  []string{"Direct", "Node A"},
					Now:  "Node A",
				},
				"Node A": {Type: "Vmess"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	c := newTestClient(handler)

	proxies, err := c.GetProxies()

	assert.NoError(t, err)
	assert.Contains(t, proxies, "Proxy")
	assert.Equal(t, "Selector", proxies["Proxy"].Type)
	assert.Equal(t, "Node A", proxies["Proxy"].Now)
}

func TestSelectProxy(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/proxies/MyGroup", r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)

		var payload map[string]string
		json.NewDecoder(r.Body).Decode(&payload)
		assert.Equal(t, "Node B", payload["name"])

		w.WriteHeader(http.StatusNoContent) // 204
	})

	c := newTestClient(handler)

	err := c.SelectProxy("MyGroup", "Node B")
	assert.NoError(t, err)
}
