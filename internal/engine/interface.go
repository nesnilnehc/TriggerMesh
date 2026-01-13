package engine

// BuildResult represents the result of a CI build trigger
type BuildResult struct {
	Success  bool   `json:"success"`
	BuildID  string `json:"build_id,omitempty"`
	BuildURL string `json:"build_url,omitempty"`
	Message  string `json:"message"`
}

// CIEngine is an interface for CI engines
type CIEngine interface {
	// TriggerBuild triggers a build for the given job with the provided parameters
	TriggerBuild(jobName string, params map[string]string) (*BuildResult, error)

	// GetBuildStatus returns the status of a build by its ID
	GetBuildStatus(buildID string) (*BuildResult, error)
}
