package clashapi

func (c *Client) GetLogLevel() (string, error) {
	var resp struct {
		Level string `json:"level"`
	}
	if err := c.do("GET", "/logs/level", nil, &resp); err != nil {
		return "", err
	}
	return resp.Level, nil
}

func (c *Client) SetLogLevel(level string) error {
	return c.do("PUT", "/logs/level", map[string]string{"level": level}, nil)
}
