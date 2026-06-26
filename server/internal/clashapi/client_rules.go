package clashapi

func (c *Client) GetRules() (*RulesResponse, error) {
	var resp RulesResponse
	if err := c.do("GET", "/rules", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
