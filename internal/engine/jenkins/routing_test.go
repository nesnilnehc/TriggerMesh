package jenkins

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"triggermesh/internal/engine"
)

type recordingEngine struct {
	jobs       []string
	buildIDs   []string
	exists     bool
	lookupErr  error
	triggerErr error
}

func (e *recordingEngine) JobExists(_ context.Context, _ string) (bool, error) {
	return e.exists, e.lookupErr
}

func (e *recordingEngine) TriggerBuild(job string, _ map[string]string) (*engine.BuildResult, error) {
	e.jobs = append(e.jobs, job)
	return &engine.BuildResult{Success: e.triggerErr == nil, BuildID: job + "/26"}, e.triggerErr
}

func (e *recordingEngine) GetBuildStatus(buildID string) (*engine.BuildResult, error) {
	e.buildIDs = append(e.buildIDs, buildID)
	return &engine.BuildResult{Success: e.triggerErr == nil}, e.triggerErr
}

func TestRoutingTriggerExplicit(t *testing.T) {
	defaultEngine := &recordingEngine{}
	secondaryEngine := &recordingEngine{triggerErr: errors.New("secondary Jenkins unavailable")}
	router := NewRoutingTrigger("primary", map[string]JobEngine{"primary": defaultEngine, "secondary": secondaryEngine})
	if _, err := router.TriggerBuildOn("primary", "example-build-job", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := router.TriggerBuildOn("secondary", "example-build-job", nil); err == nil {
		t.Fatal("selected instance failure must not fall back to default")
	}
	if _, err := router.GetBuildStatus("secondary:example-build-job/26"); err == nil {
		t.Fatal("qualified build ID must use secondary instance")
	}
	if len(defaultEngine.jobs) != 1 || len(secondaryEngine.jobs) != 1 || len(secondaryEngine.buildIDs) != 1 {
		t.Fatalf("unexpected calls: default=%v secondary=%v status=%v", defaultEngine.jobs, secondaryEngine.jobs, secondaryEngine.buildIDs)
	}
}

func TestRoutingTriggerDiscoversUniqueJob(t *testing.T) {
	defaultEngine := &recordingEngine{}
	secondaryEngine := &recordingEngine{exists: true}
	router := NewRoutingTrigger("primary", map[string]JobEngine{"primary": defaultEngine, "secondary": secondaryEngine})
	result, err := router.TriggerBuild("example-build-job", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Jenkins != "secondary" || result.BuildID != "secondary:example-build-job/26" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(defaultEngine.jobs) != 0 || len(secondaryEngine.jobs) != 1 {
		t.Fatalf("unexpected calls: default=%v secondary=%v", defaultEngine.jobs, secondaryEngine.jobs)
	}
}

func TestRoutingTriggerRejectsAmbiguousOrFailedLookup(t *testing.T) {
	defaultEngine := &recordingEngine{exists: true}
	secondaryEngine := &recordingEngine{exists: true}
	router := NewRoutingTrigger("primary", map[string]JobEngine{"primary": defaultEngine, "secondary": secondaryEngine})
	_, err := router.TriggerBuild("same-job", nil)
	if routeErr, ok := err.(*RouteError); !ok || routeErr.StatusCode() != http.StatusConflict {
		t.Fatalf("expected conflict, got %v", err)
	}
	secondaryEngine.lookupErr = errors.New("unavailable")
	_, err = router.TriggerBuild("same-job", nil)
	if routeErr, ok := err.(*RouteError); !ok || routeErr.StatusCode() != http.StatusServiceUnavailable {
		t.Fatalf("expected unavailable, got %v", err)
	}
	if len(defaultEngine.jobs)+len(secondaryEngine.jobs) != 0 {
		t.Fatal("lookup failure or ambiguity must not trigger any build")
	}
}

func TestRoutingTriggerRejectsMissingAndUnknownTarget(t *testing.T) {
	router := NewRoutingTrigger("primary", map[string]JobEngine{"primary": &recordingEngine{}, "secondary": &recordingEngine{}})
	_, err := router.TriggerBuild("missing", nil)
	if routeErr, ok := err.(*RouteError); !ok || routeErr.StatusCode() != http.StatusNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
	_, err = router.TriggerBuildOn("unknown", "missing", nil)
	if routeErr, ok := err.(*RouteError); !ok || routeErr.StatusCode() != http.StatusBadRequest {
		t.Fatalf("expected bad request, got %v", err)
	}
}
