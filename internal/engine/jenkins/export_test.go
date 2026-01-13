package jenkins

// DoRequest exports doRequest for testing purposes
// This allows tests (both in-package and external if linked correctly, but mostly for white-box testing) to access doRequest.
func (c *Client) DoRequest(method, path string, body interface{}) ([]byte, error) {
	return c.doRequest(method, path, body)
}
