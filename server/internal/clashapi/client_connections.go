package clashapi

func (c *Client) GetConnections() (*ConnectionsResponse, error) {
	var resp ConnectionsResponse
	if err := c.do("GET", "/connections", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) CloseAllConnections() error {
	return c.do("DELETE", "/connections", nil, nil)
}

func (c *Client) CloseConnection(id string) error {
	return c.do("DELETE", "/connections/"+id, nil, nil)
}
