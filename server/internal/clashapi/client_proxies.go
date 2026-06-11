package clashapi

import (
	"fmt"
	"net/url"
)

func (c *Client) GetProxies() (*ProxiesResponse, error) {
	var resp ProxiesResponse
	if err := c.do("GET", "/proxies", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetProxy(name string) (*ProxyDetail, error) {
	var resp ProxyDetail
	if err := c.do("GET", "/proxies/"+url.PathEscape(name), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) SwitchProxy(groupName, proxyName string) error {
	body := map[string]string{"name": proxyName}
	return c.do("PUT", "/proxies/"+url.PathEscape(groupName), body, nil)
}

func (c *Client) GetProxyDelay(name, testURL string, timeout int) (int, error) {
	path := fmt.Sprintf("/proxies/%s/delay?url=%s&timeout=%d",
		url.PathEscape(name), url.QueryEscape(testURL), timeout)
	var resp DelayResponse
	if err := c.do("GET", path, nil, &resp); err != nil {
		return 0, err
	}
	return resp.Delay, nil
}
