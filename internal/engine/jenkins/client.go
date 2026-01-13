package jenkins

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"triggermesh/internal/config"
	"triggermesh/internal/logger"
)

// Client represents a Jenkins API client
type Client struct {
	url      string
	username string
	token    string
	client   *http.Client
}

// NewClient creates a new Jenkins client instance
func NewClient(cfg config.JenkinsConfig) *Client {
	// Create HTTP client with timeout
	timeout := time.Duration(cfg.Timeout) * time.Second
	client := &http.Client{
		Timeout: timeout,
	}

	// Normalize URL: remove trailing slash to avoid double slashes in paths
	url := strings.TrimSuffix(cfg.URL, "/")

	return &Client{
		url:      url,
		username: cfg.Username,
		token:    cfg.Token,
		client:   client,
	}
}

// doRequest sends an HTTP request to the Jenkins API
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	// Build the full URL
	url := c.url + path

	var reqBody io.Reader
	if body != nil {
		// Marshal body to JSON
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	// Create the request with context
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Jenkins API uses Basic Authentication
	// Format: username:token
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", c.username, c.token)))
	req.Header.Set("Authorization", "Basic "+auth)

	// Send the request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Check if the response status is successful
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Error("Jenkins API request failed", "status", resp.Status, "body", string(respBody), "url", url)
		return nil, formatJenkinsError(resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// doBuildRequest sends a POST request to trigger a Jenkins build without parameters
// Returns build ID and build URL extracted from the Location header
func (c *Client) doBuildRequest(ctx context.Context, buildPath string) (string, string, error) {
	fullURL := c.url + buildPath

	// Get CSRF crumb first - some Jenkins versions require it in the form data
	crumbField, crumbValue, err := c.getCrumb(ctx)
	if err != nil {
		logger.Warn("Failed to get CSRF crumb, proceeding without it", "error", err)
	}

	// Create form data - Jenkins Stapler servlet requires specific form fields
	// For non-parameterized builds, Jenkins expects a "json" field with build configuration
	formData := url.Values{}
	// Add json field with empty build configuration (required by Jenkins Stapler)
	formData.Set("json", "{}")

	// Include the crumb in the form data if available
	if crumbField != "" && crumbValue != "" {
		formData.Set(crumbField, crumbValue)
	}

	reqBody := strings.NewReader(formData.Encode())

	// Create the request with context
	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, reqBody)
	if err != nil {
		return "", "", err
	}

	// Set Content-Type for form-encoded data
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Set authentication
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", c.username, c.token)))
	req.Header.Set("Authorization", "Basic "+auth)

	// Also set crumb in header if available (some Jenkins versions require both)
	if crumbField != "" && crumbValue != "" {
		req.Header.Set(crumbField, crumbValue)
	}

	// Send the request
	resp, err := c.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	// Read response body for error messages
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response body: %v", err)
	}

	// Check if the response status is successful
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Error("Jenkins build request failed", "status", resp.Status, "body", string(respBody), "url", fullURL)
		return "", "", formatJenkinsError(resp.StatusCode, string(respBody))
	}

	// Extract build ID and URL from Location header
	location := resp.Header.Get("Location")
	buildID, buildURL := c.extractBuildInfo(location, buildPath)

	return buildID, buildURL, nil
}

// doParameterizedRequest sends a POST request to trigger a Jenkins build with parameters
// Jenkins buildWithParameters expects form-encoded data
func (c *Client) doParameterizedRequest(ctx context.Context, buildPath string, params map[string]string) (string, string, error) {
	fullURL := c.url + buildPath

	// Create form data
	formData := url.Values{}
	for k, v := range params {
		formData.Set(k, v)
	}

	// Create the request with form-encoded body and context
	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return "", "", err
	}

	// Set headers for form-encoded data
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Set authentication
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", c.username, c.token)))
	req.Header.Set("Authorization", "Basic "+auth)

	// Jenkins expects a CSRF token for POST requests
	crumbField, crumbValue, err := c.getCrumb(ctx)
	if err != nil {
		logger.Warn("Failed to get CSRF crumb, proceeding without it", "error", err)
	} else if crumbField != "" && crumbValue != "" {
		req.Header.Set(crumbField, crumbValue)
	}

	// Send the request
	resp, err := c.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	// Read response body for error messages
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response body: %v", err)
	}

	// Check if the response status is successful
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Error("Jenkins parameterized build request failed", "status", resp.Status, "body", string(respBody), "url", fullURL)
		return "", "", formatJenkinsError(resp.StatusCode, string(respBody))
	}

	// Extract build ID and URL from Location header
	location := resp.Header.Get("Location")
	buildID, buildURL := c.extractBuildInfo(location, buildPath)

	return buildID, buildURL, nil
}

// getCrumb retrieves the CSRF crumb from Jenkins for POST requests
// Returns the crumb field name and value separately
func (c *Client) getCrumb(ctx context.Context) (string, string, error) {
	crumbURL := c.url + "/crumbIssuer/api/json"

	req, err := http.NewRequestWithContext(ctx, "GET", crumbURL, nil)
	if err != nil {
		return "", "", err
	}

	// Set authentication
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", c.username, c.token)))
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("failed to get crumb: %s", resp.Status)
	}

	var crumbData struct {
		Crumb             string `json:"crumb"`
		CrumbRequestField string `json:"crumbRequestField"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&crumbData); err != nil {
		return "", "", err
	}

	// Return the crumb field name and value separately
	crumbField := crumbData.CrumbRequestField
	if crumbField == "" {
		crumbField = "Jenkins-Crumb" // Default field name
	}

	return crumbField, crumbData.Crumb, nil
}

// extractBuildInfo extracts build ID and URL from Jenkins Location header
// Location format: /job/jobName/buildNumber/ or http://jenkins/job/jobName/buildNumber/
func (c *Client) extractBuildInfo(location, buildPath string) (string, string) {
	if location == "" {
		// If no location header, try to extract from buildPath
		// buildPath format: /job/jobName/build or /job/jobName/buildWithParameters
		parts := strings.Split(strings.TrimPrefix(buildPath, "/job/"), "/")
		if len(parts) > 0 {
			jobName := parts[0]
			return "", fmt.Sprintf("%s/job/%s/", c.url, jobName)
		}
		return "", ""
	}

	// Parse location to extract job name and build number
	// Location can be relative or absolute
	var pathPart string
	if strings.HasPrefix(location, "http://") || strings.HasPrefix(location, "https://") {
		// Absolute URL
		u, err := url.Parse(location)
		if err != nil {
			return "", ""
		}
		pathPart = u.Path
	} else {
		// Relative URL
		pathPart = location
	}

	// Extract job name and build number from path
	// Format: /job/jobName/buildNumber/
	parts := strings.Split(strings.Trim(pathPart, "/"), "/")
	if len(parts) >= 3 && parts[0] == "job" {
		jobName := parts[1]
		buildNumber := parts[2]
		buildID := jobName + "/" + buildNumber
		buildURL := fmt.Sprintf("%s/job/%s/%s/", c.url, jobName, buildNumber)
		return buildID, buildURL
	}

	return "", ""
}

// formatJenkinsError formats Jenkins API errors into user-friendly messages
// without exposing internal implementation details
func formatJenkinsError(statusCode int, responseBody string) error {
	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("authentication failed: invalid credentials")
	case http.StatusForbidden:
		return fmt.Errorf("access denied: insufficient permissions")
	case http.StatusNotFound:
		return fmt.Errorf("resource not found")
	case http.StatusBadRequest:
		return fmt.Errorf("invalid request")
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return fmt.Errorf("jenkins server error: please try again later")
	default:
		// For other errors, return a generic message
		return fmt.Errorf("jenkins api request failed")
	}
}
