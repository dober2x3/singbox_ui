package clashapi

func (c *Client) GetConfigs() (*ConfigResponse, error) {
	var resp ConfigResponse
	if err := c.do("GET", "/configs", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) PatchConfigs(partial map[string]interface{}) error {
	return c.do("PUT", "/configs", partial, nil)
}

func (c *Client) GetMode() (string, error) {
	cfg, err := c.GetConfigs()
	if err != nil {
		return "", err
	}
	return cfg.Mode, nil
}

func (c *Client) SetMode(mode string) error {
	return c.PatchConfigs(map[string]interface{}{"mode": mode})
}
