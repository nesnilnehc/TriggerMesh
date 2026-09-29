package jenkins

import (
	"strings"

	"triggermesh/internal/engine"
)

// RoutingTrigger sends selected jobs to another Jenkins instance while keeping
// existing jobs on the default instance.
type RoutingTrigger struct {
	defaultTrigger engine.CIEngine
	routes         map[string]engine.CIEngine
}

func NewRoutingTrigger(defaultTrigger engine.CIEngine, routes map[string]engine.CIEngine) *RoutingTrigger {
	return &RoutingTrigger{defaultTrigger: defaultTrigger, routes: routes}
}

func (r *RoutingTrigger) triggerForJob(jobName string) engine.CIEngine {
	if trigger, ok := r.routes[jobName]; ok {
		return trigger
	}
	return r.defaultTrigger
}

func (r *RoutingTrigger) TriggerBuild(jobName string, params map[string]string) (*engine.BuildResult, error) {
	return r.triggerForJob(jobName).TriggerBuild(jobName, params)
}

func (r *RoutingTrigger) GetBuildStatus(buildID string) (*engine.BuildResult, error) {
	jobName, _, _ := strings.Cut(buildID, "/")
	return r.triggerForJob(jobName).GetBuildStatus(buildID)
}
