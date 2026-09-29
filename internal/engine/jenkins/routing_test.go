package jenkins

import (
	"errors"
	"testing"

	"triggermesh/internal/engine"
)

type recordingEngine struct {
	jobs     []string
	buildIDs []string
	err      error
}

func (e *recordingEngine) TriggerBuild(job string, _ map[string]string) (*engine.BuildResult, error) {
	e.jobs = append(e.jobs, job)
	return &engine.BuildResult{Success: e.err == nil}, e.err
}

func (e *recordingEngine) GetBuildStatus(buildID string) (*engine.BuildResult, error) {
	e.buildIDs = append(e.buildIDs, buildID)
	return &engine.BuildResult{Success: e.err == nil}, e.err
}

func TestRoutingTrigger(t *testing.T) {
	defaultEngine := &recordingEngine{}
	ddiEngine := &recordingEngine{err: errors.New("DDI Jenkins unavailable")}
	router := NewRoutingTrigger(defaultEngine, map[string]engine.CIEngine{
		"publish-recloud-ddi-artifacts": ddiEngine,
	})

	if _, err := router.TriggerBuild("recloud-jcy-v2-design-publish", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := router.TriggerBuild("publish-recloud-ddi-artifacts", map[string]string{"branch": "3.45"}); err == nil {
		t.Fatal("routed job must not fall back to the default Jenkins when its target fails")
	}
	if _, err := router.GetBuildStatus("publish-recloud-ddi-artifacts/26"); err == nil {
		t.Fatal("build status must use the routed Jenkins instance")
	}
	if len(defaultEngine.jobs) != 1 || defaultEngine.jobs[0] != "recloud-jcy-v2-design-publish" {
		t.Fatalf("unexpected default jobs: %v", defaultEngine.jobs)
	}
	if len(ddiEngine.jobs) != 1 || ddiEngine.jobs[0] != "publish-recloud-ddi-artifacts" {
		t.Fatalf("unexpected DDI jobs: %v", ddiEngine.jobs)
	}
	if len(ddiEngine.buildIDs) != 1 || ddiEngine.buildIDs[0] != "publish-recloud-ddi-artifacts/26" {
		t.Fatalf("unexpected DDI build IDs: %v", ddiEngine.buildIDs)
	}
}
