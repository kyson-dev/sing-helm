package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetProxies(t *testing.T) {
	// 1. 模拟 Clash API 服务器
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/proxies", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		// 返回假数据
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
	}))
	defer ts.Close()

	// 2. 初始化 Client (去除 http:// 前缀，因为 httptest 自动带了)
	// 这里我们需要稍微 hack 一下或者在 New 里面处理，
	// 既然 New 里面强制加了 http://，我们传 URL 时要去掉 scheme
	// 为了简单，我们直接构造 Client 结构体
	c := &Client{
		baseURL:    ts.URL,
		httpClient: ts.Client(),
	}

	// 3. 执行测试
	proxies, err := c.GetProxies()

	// 4. 断言
	assert.NoError(t, err)
	assert.Contains(t, proxies, "Proxy")
	assert.Equal(t, "Selector", proxies["Proxy"].Type)
	assert.Equal(t, "Node A", proxies["Proxy"].Now)
}

func TestSelectProxy(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证 URL 路径是否包含组名
		assert.Equal(t, "/proxies/MyGroup", r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)

		// 验证 Payload
		var payload map[string]string
		json.NewDecoder(r.Body).Decode(&payload)
		assert.Equal(t, "Node B", payload["name"])

		w.WriteHeader(http.StatusNoContent) // 204
	}))
	defer ts.Close()

	c := &Client{baseURL: ts.URL, httpClient: ts.Client()}

	err := c.SelectProxy("MyGroup", "Node B")
	assert.NoError(t, err)
}
