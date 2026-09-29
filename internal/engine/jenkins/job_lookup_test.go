package jenkins

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"triggermesh/internal/config"
)

func TestJobExistsDistinguishesMatchAbsenceAndFailure(t *testing.T) {
	for _, tc := range []struct {
		name      string
		status    int
		wantMatch bool
		wantErr   bool
	}{
		{"found", http.StatusOK, true, false},
		{"missing", http.StatusNotFound, false, false},
		{"denied", http.StatusForbidden, false, true},
		{"redirect", http.StatusFound, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/login" {
					w.WriteHeader(http.StatusOK)
					return
				}
				if r.URL.Path != "/job/team/job/build/api/json" || r.URL.Query().Get("tree") != "name" {
					t.Errorf("unexpected lookup URL %q", r.URL.String())
				}
				if user, token, ok := r.BasicAuth(); !ok || user != "reader" || token != "secret" {
					t.Error("missing Jenkins credentials")
				}
				if tc.status == http.StatusFound {
					http.Redirect(w, r, "/login", http.StatusFound)
					return
				}
				w.WriteHeader(tc.status)
			}))
			defer server.Close()
			client := NewClient(config.JenkinsConfig{URL: server.URL, Username: "reader", Token: "secret", Timeout: 2})
			found, err := client.JobExists(context.Background(), "team/build")
			if found != tc.wantMatch || (err != nil) != tc.wantErr {
				t.Fatalf("found=%v err=%v", found, err)
			}
		})
	}
}

func TestFolderBuildLocation(t *testing.T) {
	client := NewClient(config.JenkinsConfig{URL: "https://jenkins.example.com", Token: "secret", Timeout: 2})
	buildID, buildURL := client.extractBuildInfo("/job/team/job/build/26/", jobPath("team/build")+"/build")
	if buildID != "team/build/26" || buildURL != "https://jenkins.example.com/job/team/job/build/26/" {
		t.Fatalf("unexpected build ID %q or URL %q", buildID, buildURL)
	}
}
