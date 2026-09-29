package jenkins

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"

	"triggermesh/internal/engine"
)

// JobEngine can check for a job before triggering a build.
type JobEngine interface {
	engine.CIEngine
	JobExists(context.Context, string) (bool, error)
}

// RouteError is returned when a job cannot be resolved to one Jenkins instance.
type RouteError struct {
	status  int
	message string
}

func (e *RouteError) Error() string   { return e.message }
func (e *RouteError) StatusCode() int { return e.status }

// RoutingTrigger selects a requested instance or discovers a unique job.
type RoutingTrigger struct {
	defaultName string
	instances   map[string]JobEngine
}

func NewRoutingTrigger(defaultName string, instances map[string]JobEngine) *RoutingTrigger {
	return &RoutingTrigger{defaultName: defaultName, instances: instances}
}

func (r *RoutingTrigger) namedInstance(name string) (JobEngine, error) {
	if target, ok := r.instances[name]; ok {
		return target, nil
	}
	return nil, &RouteError{http.StatusBadRequest, fmt.Sprintf("unknown Jenkins instance %q", name)}
}

func (r *RoutingTrigger) resolve(job, requested string) (string, JobEngine, error) {
	if requested != "" {
		target, err := r.namedInstance(requested)
		return requested, target, err
	}
	if len(r.instances) == 1 {
		return r.defaultName, r.instances[r.defaultName], nil
	}

	// A failed lookup prevents automatic selection, because the unavailable
	// instance may also own the job.
	type lookup struct {
		name   string
		exists bool
		err    error
	}
	results := make(chan lookup, len(r.instances))
	var wg sync.WaitGroup
	for name, target := range r.instances {
		wg.Add(1)
		go func(name string, target JobEngine) {
			defer wg.Done()
			exists, err := target.JobExists(context.Background(), job)
			results <- lookup{name, exists, err}
		}(name, target)
	}
	wg.Wait()
	close(results)
	matches := make([]string, 0, 1)
	failed := make([]string, 0)
	for result := range results {
		if result.err != nil {
			failed = append(failed, result.name)
		} else if result.exists {
			matches = append(matches, result.name)
		}
	}
	sort.Strings(matches)
	sort.Strings(failed)
	if len(failed) > 0 {
		return "", nil, &RouteError{http.StatusServiceUnavailable, fmt.Sprintf("cannot resolve job %q: Jenkins lookup failed for %s", job, strings.Join(failed, ", "))}
	}
	if len(matches) == 0 {
		return "", nil, &RouteError{http.StatusNotFound, fmt.Sprintf("job %q was not found on any configured Jenkins instance", job)}
	}
	if len(matches) > 1 {
		return "", nil, &RouteError{http.StatusConflict, fmt.Sprintf("job %q exists on multiple Jenkins instances (%s); specify jenkins", job, strings.Join(matches, ", "))}
	}
	return matches[0], r.instances[matches[0]], nil
}

// TriggerBuild discovers a unique job match when the request omits jenkins.
func (r *RoutingTrigger) TriggerBuild(job string, params map[string]string) (*engine.BuildResult, error) {
	return r.TriggerBuildOn("", job, params)
}

// TriggerBuildOn lets a request select one configured Jenkins instance.
func (r *RoutingTrigger) TriggerBuildOn(requested, job string, params map[string]string) (*engine.BuildResult, error) {
	name, target, err := r.resolve(job, requested)
	if err != nil {
		return &engine.BuildResult{Success: false, Message: err.Error()}, err
	}
	result, err := target.TriggerBuild(job, params)
	if result != nil {
		result.Jenkins = name
		if name != r.defaultName && result.BuildID != "" {
			result.BuildID = name + ":" + result.BuildID
		}
	}
	return result, err
}

// GetBuildStatus uses a qualified build ID, then the default instance.
func (r *RoutingTrigger) GetBuildStatus(buildID string) (*engine.BuildResult, error) {
	if name, unqualified, ok := strings.Cut(buildID, ":"); ok {
		target, err := r.namedInstance(name)
		if err != nil {
			return &engine.BuildResult{Success: false, Message: err.Error()}, err
		}
		return target.GetBuildStatus(unqualified)
	}
	return r.instances[r.defaultName].GetBuildStatus(buildID)
}
