package jenkins

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"triggermesh/internal/engine"
	"triggermesh/internal/logger"
)

// jenkinsBuildResult represents the result of a Jenkins build
type jenkinsBuildResult struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
}

// jenkinsJobInfo represents Jenkins job information
type jenkinsJobInfo struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Trigger implements the CIEngine interface for Jenkins
type Trigger struct {
	client *Client
}

// NewTrigger creates a new Jenkins trigger instance
func NewTrigger(client *Client) *Trigger {
	return &Trigger{
		client: client,
	}
}

// TriggerBuild triggers a Jenkins build for the given job with the provided parameters
func (t *Trigger) TriggerBuild(jobName string, params map[string]string) (*engine.BuildResult, error) {
	// Validate job name
	if jobName == "" {
		return &engine.BuildResult{
			Success: false,
			Message: "Job name cannot be empty",
		}, fmt.Errorf("job name cannot be empty")
	}

	// Validate job name format (no special characters that could cause path issues)
	if strings.Contains(jobName, "..") || strings.Contains(jobName, "/") {
		return &engine.BuildResult{
			Success: false,
			Message: "Invalid job name format",
		}, fmt.Errorf("invalid job name format: %s", jobName)
	}

	// Build the path for the build trigger API
	buildPath := fmt.Sprintf("/job/%s/build", url.PathEscape(jobName))

	// If there are parameters, use the buildWithParameters endpoint
	if len(params) > 0 {
		buildPath = fmt.Sprintf("/job/%s/buildWithParameters", url.PathEscape(jobName))
	}

	// Jenkins API for buildWithParameters expects form-encoded data, not JSON
	// We'll use a custom method for parameterized builds
	var buildID string
	var buildURL string
	var err error

	if len(params) > 0 {
		buildID, buildURL, err = t.client.doParameterizedRequest(buildPath, params)
	} else {
		buildID, buildURL, err = t.client.doBuildRequest(buildPath)
	}

	if err != nil {
		return &engine.BuildResult{
			Success: false,
			Message: fmt.Sprintf("Failed to trigger Jenkins build: %v", err),
		}, err
	}

	return &engine.BuildResult{
		Success:  true,
		Message:  fmt.Sprintf("Successfully triggered Jenkins build for job %s", jobName),
		BuildID:  buildID,
		BuildURL: buildURL,
	}, nil
}

// GetBuildStatus returns the status of a Jenkins build by its ID
func (t *Trigger) GetBuildStatus(buildID string) (*engine.BuildResult, error) {
	// Validate buildID
	if buildID == "" {
		return &engine.BuildResult{
			Success: false,
			Message: "Build ID cannot be empty",
		}, fmt.Errorf("build ID cannot be empty")
	}

	// Parse buildID to extract job name and build number
	// Expected format: jobName/buildNumber
	parts := strings.Split(buildID, "/")
	if len(parts) != 2 {
		return &engine.BuildResult{
			Success: false,
			Message: "Invalid build ID format. Expected: jobName/buildNumber",
		}, fmt.Errorf("invalid build ID format: %s", buildID)
	}

	jobName := parts[0]
	buildNumber := parts[1]

	// Validate build number
	if buildNumber == "" {
		return &engine.BuildResult{
			Success: false,
			Message: "Build number cannot be empty",
		}, fmt.Errorf("build number cannot be empty")
	}

	// Build the path for the build info API
	buildPath := fmt.Sprintf("/job/%s/%s/api/json", url.PathEscape(jobName), url.PathEscape(buildNumber))

	// Send the request to Jenkins
	respBody, err := t.client.doRequest("GET", buildPath, nil)
	if err != nil {
		return &engine.BuildResult{
			Success: false,
			Message: fmt.Sprintf("Failed to get Jenkins build status: %v", err),
		}, err
	}

	// Parse the response to get build status
	var buildInfo jenkinsBuildResult
	if err := json.Unmarshal(respBody, &buildInfo); err != nil {
		logger.Warn("Failed to parse build info, returning basic info", "error", err)
		// If parsing fails, return basic info
		return &engine.BuildResult{
			Success:  true,
			Message:  fmt.Sprintf("Retrieved build status for %s", buildID),
			BuildID:  buildID,
			BuildURL: fmt.Sprintf("%s/job/%s/%s/", t.client.url, jobName, buildNumber),
		}, nil
	}

	buildURL := buildInfo.URL
	if buildURL == "" {
		buildURL = fmt.Sprintf("%s/job/%s/%s/", t.client.url, jobName, buildNumber)
	}

	return &engine.BuildResult{
		Success:  true,
		Message:  fmt.Sprintf("Retrieved build status for %s", buildID),
		BuildID:  buildID,
		BuildURL: buildURL,
	}, nil
}
