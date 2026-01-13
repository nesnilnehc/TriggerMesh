package jenkins

import "context"

// DoRequest exports doRequest for testing purposes
// This allows tests (both in-package and external if linked correctly, but mostly for white-box testing) to access doRequest.
func (c *Client) DoRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	return c.doRequest(ctx, method, path, body)
}
