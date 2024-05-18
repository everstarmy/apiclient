package aipclient

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

// ClientInterface 定义了HTTP客户端接口
type ClientInterface interface {
	Get(endpoint string, params map[string]string, v interface{}) (int, error)
	Post(endpoint string, jsonStr []byte, v interface{}) (int, error)
	Put(endpoint string, jsonStr []byte, v interface{}) (int, error)
	Delete(endpoint string, params map[string]string, v interface{}) (int, error)
}

// HTTPClient 自定义的HTTP客户端
type HTTPClient struct {
	client  *http.Client
	baseURL string
	token   string
}

var tr = &http.Transport{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
}

// NewHTTPClient 创建一个新的HTTP客户端实例，并通过用户名和密码进行认证获取Bearer Token
func NewHTTPClient(baseURL, authURL, username, password string) (ClientInterface, error) {
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: tr,
	}

	// 进行认证获取Bearer Token
	token, err := authenticate(client, authURL, username, password)
	if err != nil {
		return nil, err
	}

	return &HTTPClient{
		client:  client,
		baseURL: baseURL,
		token:   token,
	}, nil
}

// authenticate 用于通过用户名和密码进行认证，并获取Bearer Token
func authenticate(client *http.Client, authURL, username, password string) (string, error) {
	authData := map[string]string{
		"username": username,
		"password": password,
	}
	jsonData, err := json.Marshal(authData)
	if err != nil {
		return "", fmt.Errorf("创建认证请求数据失败: %w", err)
	}

	req, err := http.NewRequest("POST", authURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建认证请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("认证请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("认证失败，状态码: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取认证响应失败: %w", err)
	}

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析认证响应失败: %w", err)
	}

	token, ok := result["token"]
	if !ok {
		return "", fmt.Errorf("认证响应中未找到token")
	}

	return token, nil
}

// doRequest 发送HTTP请求并将响应解析为指定的数据结构
func (c *HTTPClient) doRequest(req *http.Request, v interface{}) (int, error) {
	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, fmt.Errorf("请求失败，状态码: %d，响应: %s", resp.StatusCode, body)
	}

	if err := json.Unmarshal(body, v); err != nil {
		return resp.StatusCode, fmt.Errorf("解析响应失败: %w", err)
	}

	return resp.StatusCode, nil
}

// Get 发送HTTP GET请求并将响应解析为指定的数据结构
func (c *HTTPClient) Get(endpoint string, params map[string]string, v interface{}) (int, error) {
	u, err := url.Parse(c.baseURL + endpoint)
	if err != nil {
		return 0, fmt.Errorf("解析URL失败: %w", err)
	}

	query := u.Query()
	for key, value := range params {
		query.Set(key, value)
	}
	u.RawQuery = query.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return 0, fmt.Errorf("创建请求失败: %w", err)
	}
	return c.doRequest(req, v)
}

// Post 发送HTTP POST请求并将响应解析为指定的数据结构
func (c *HTTPClient) Post(endpoint string, jsonStr []byte, v interface{}) (int, error) {
	u := c.baseURL + endpoint
	req, err := http.NewRequest("POST", u, bytes.NewBuffer(jsonStr))
	if err != nil {
		return 0, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return c.doRequest(req, v)
}

// Put 发送HTTP PUT请求并将响应解析为指定的数据结构
func (c *HTTPClient) Put(endpoint string, jsonStr []byte, v interface{}) (int, error) {
	u := c.baseURL + endpoint
	req, err := http.NewRequest("PUT", u, bytes.NewBuffer(jsonStr))
	if err != nil {
		return 0, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return c.doRequest(req, v)
}

// Delete 发送HTTP DELETE请求并将响应解析为指定的数据结构
func (c *HTTPClient) Delete(endpoint string, params map[string]string, v interface{}) (int, error) {
	parse, err := url.Parse(c.baseURL + endpoint)
	if err != nil {
		return 0, fmt.Errorf("解析URL失败: %w", err)
	}

	query := parse.Query()
	for key, value := range params {
		query.Set(key, value)
	}
	parse.RawQuery = query.Encode()

	req, err := http.NewRequest("DELETE", parse.String(), nil)
	if err != nil {
		return 0, fmt.Errorf("创建请求失败: %w", err)
	}
	return c.doRequest(req, v)
}
