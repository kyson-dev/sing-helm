package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// ProxyData 对应 Clash API 返回的单个代理/节点组结构
type ProxyData struct {
	Type string   `json:"type"`          // 类型: Selector, URLTest, Vmess 等
	All  []string `json:"all,omitempty"` // 包含的所有节点名称 (仅 Selector 有)
	Now  string   `json:"now,omitempty"` // 当前选中的节点 (仅 Selector 有)
}

// ProxiesResponse 对应 GET /proxies 的响应
type proxiesResponse struct {
	Proxies map[string]ProxyData `json:"proxies"`
}

// DelayResult 延迟测试结果
type DelayResult struct {
	Delay int `json:"delay"` // 毫秒
}

// Client 封装对 Sing-box API 的 HTTP 请求
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New 创建客户端
func New(host string) *Client {
	return &Client{
		baseURL: fmt.Sprintf("http://%s", host),
		httpClient: &http.Client{
			Timeout: 2 * time.Second, // 内网请求，超时设短一点
		},
	}
}

// GetProxies 获取所有代理节点信息
// 返回一个 map，Key 是组名(如 "Proxy"), Value 是详细信息
func (c *Client) GetProxies() (map[string]ProxyData, error) {
	url := fmt.Sprintf("%s/proxies", c.baseURL)
	
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("connect api failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned status: %s", resp.Status)
	}

	var result proxiesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode json failed: %w", err)
	}

	return result.Proxies, nil
}

// SelectProxy 切换节点
// group: 选择器组的名字 (如 "Proxy", "GLOBAL")
// node: 目标节点的名字
func (c *Client) SelectProxy(group, node string) error {
	url := fmt.Sprintf("%s/proxies/%s", c.baseURL, group)
	
	// 构造 payload: {"name": "node_name"}
	payload := map[string]string{"name": node}
	data, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Clash API 成功时通常返回 204 No Content
	if resp.StatusCode >= 400 {
		return fmt.Errorf("api error: %s", resp.Status)
	}

	return nil
}

// GetNodeDelay 测试指定节点的延迟
// name: 节点名称
// testURL: 测试链接 (如 http://www.gstatic.com/generate_204)
// timeout: 超时时间 (毫秒)
func (c *Client) GetNodeDelay(name string, testURL string, timeout int) (int, error) {
	// 构造 URL: /proxies/:name/delay?url=...&timeout=...
	// 注意对 params 进行 url encode
	params := url.Values{}
	params.Add("url", testURL)
	params.Add("timeout", strconv.Itoa(timeout))
	
	// 注意：节点名称可能包含特殊字符（空格、emoji），必须 Encode
	encodedName := url.PathEscape(name)
	apiURL := fmt.Sprintf("%s/proxies/%s/delay?%s", c.baseURL, encodedName, params.Encode())

	resp, err := c.httpClient.Get(apiURL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("bad status: %s", resp.Status)
	}

	var res DelayResult
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return 0, err
	}

	return res.Delay, nil
}