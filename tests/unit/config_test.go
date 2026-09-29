package unit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"triggermesh/internal/config"
)

const validConfig = `
server:
  port: 8080
database:
  path: ./data/test.db
jenkins:
  default: primary
  instances:
    primary:
      url: https://primary.example.com
      token: primary-token
    secondary:
      url: https://secondary.example.com
      username: secondary-user
      token: secondary-token
      timeout: 45
api:
  keys:
    - test-api-key
`

func loadConfigText(t *testing.T, content string) (*config.Config, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return config.Load(path)
}

func TestLoadConfigWithNamedJenkinsInstances(t *testing.T) {
	cfg, err := loadConfigText(t, validConfig)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Jenkins.Default != "primary" || len(cfg.Jenkins.Instances) != 2 {
		t.Fatalf("unexpected instances: %+v", cfg.Jenkins)
	}
	primary := cfg.Jenkins.Instances["primary"]
	if primary.Timeout != 30 || primary.Username != primary.Token {
		t.Fatalf("defaults not applied: %+v", primary)
	}
	secondary := cfg.Jenkins.Instances["secondary"]
	if secondary.Timeout != 45 || secondary.Username != "secondary-user" {
		t.Fatalf("unexpected secondary instance: %+v", secondary)
	}
	if cfg.Server.Host != "0.0.0.0" || cfg.Server.MaxBodySize != 1<<20 {
		t.Fatalf("server defaults not applied: %+v", cfg.Server)
	}
}

func TestJenkinsConfigurationValidation(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{"missing default", strings.Replace(validConfig, "default: primary", "default: missing", 1), "jenkins.default"},
		{"missing token", strings.Replace(validConfig, "token: secondary-token", "token: ''", 1), "token is required"},
		{"invalid URL", strings.Replace(validConfig, "https://secondary.example.com", "ftp://secondary.example.com", 1), "HTTP(S) URL"},
		{"invalid name", strings.Replace(validConfig, "    secondary:", "    'bad:name':", 1), "must contain only"},
		{"legacy config", strings.Replace(validConfig, "  default: primary", "  url: https://legacy.example.com\n  token: legacy-token\n  default: primary", 1), "jenkins.instances"},
		{"unknown instances key", strings.Replace(validConfig, "  instances:", "  old_instances:", 1), "old_instances"},
		{"retired routing key", validConfig + "job_targets:\n  example-build-job: secondary\n", "job_targets"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := loadConfigText(t, tc.content)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
}

func TestOtherConfigurationValidation(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{"bad port", strings.Replace(validConfig, "port: 8080", "port: 70000", 1), "invalid server.port"},
		{"bad body limit", strings.Replace(validConfig, "port: 8080", "port: 8080\n  max_body_size: 104857601", 1), "max_body_size"},
		{"missing API keys", strings.Replace(validConfig, "    - test-api-key", "    - ''", 1), "api.keys"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := loadConfigText(t, tc.content)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
}

func TestServerEnvironmentOverrides(t *testing.T) {
	t.Setenv("TRIGGERMESH_SERVER_PORT", "9090")
	t.Setenv("TRIGGERMESH_DATABASE_PATH", "/tmp/custom.db")
	cfg, err := loadConfigText(t, validConfig)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != 9090 || cfg.Database.Path != "/tmp/custom.db" {
		t.Fatalf("environment overrides not applied: %+v", cfg)
	}
}

func TestConfigReadErrors(t *testing.T) {
	if _, err := config.Load(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Fatal("missing file should fail")
	}
	if _, err := loadConfigText(t, "jenkins: ["); err == nil {
		t.Fatal("invalid YAML should fail")
	}
}

func TestGetLogLevel(t *testing.T) {
	for _, level := range []string{"debug", "info", "warn", "error"} {
		t.Setenv("TRIGGERMESH_LOG_LEVEL", level)
		if got := config.GetLogLevel(); got != level {
			t.Fatalf("expected %q, got %q", level, got)
		}
	}
	t.Setenv("TRIGGERMESH_LOG_LEVEL", "invalid")
	if got := config.GetLogLevel(); got != "info" {
		t.Fatalf("expected info, got %q", got)
	}
}
