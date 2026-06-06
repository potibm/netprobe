package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHttpProbeConfigUrl(t *testing.T) {
	nourl := HTTPProbeConfig{
		ExpectUp: true,
	}
	assert.Equal(t, "http://example.com/", nourl.GetURL("example.com"))

	withurl := HTTPProbeConfig{
		URL:      strPtr("http://custom-url.com/health"),
		ExpectUp: true,
	}
	assert.Equal(t, "http://custom-url.com/health", withurl.GetURL("example.com"))
}

func TestHttpProbeConfigExpectRedirectToHttps(t *testing.T) {
	unset := HTTPProbeConfig{ExpectUp: true}
	assert.False(t, unset.DoesExpectRedirectToHTTP())

	trueVal := true
	setTrue := HTTPProbeConfig{ExpectUp: true, ExpectRedirectToHTTPS: &trueVal}
	assert.True(t, setTrue.DoesExpectRedirectToHTTP())

	falseVal := false
	setFalse := HTTPProbeConfig{ExpectUp: true, ExpectRedirectToHTTPS: &falseVal}
	assert.False(t, setFalse.DoesExpectRedirectToHTTP())
}

func TestYamlDeserializationSetsExpectRedirectToHttps(t *testing.T) {
	yamlData := []byte(`
name: test
probes:
  - id: portal
    hostname: www.example.com
    http:
      expect_up: true
      expect_redirect_to_https: true
  - id: legacy
    hostname: legacy.example.com
    http:
      expect_up: false
      expect_redirect_to_https: false
`)
	cfg, err := LoadConfigFromBytes(yamlData)
	assert.NoError(t, err)
	assert.Len(t, cfg.Probes, 2)

	assert.NotNil(t, cfg.Probes[0].HTTP)
	assert.True(t, cfg.Probes[0].HTTP.DoesExpectRedirectToHTTP())

	assert.NotNil(t, cfg.Probes[1].HTTP)
	assert.False(t, cfg.Probes[1].HTTP.DoesExpectRedirectToHTTP())
}

func TestHttpsProbeConfigUrl(t *testing.T) {
	nourl := HTTPSProbeConfig{
		ExpectUp: true,
	}
	assert.Equal(t, "https://example.com/", nourl.GetURL("example.com"))

	withurl := HTTPSProbeConfig{
		URL:      strPtr("https://custom-url.com/health"),
		ExpectUp: true,
	}
	assert.Equal(t, "https://custom-url.com/health", withurl.GetURL("example.com"))
}

func TestHttpsProbeConfigExpectValidCert(t *testing.T) {
	unset := HTTPSProbeConfig{ExpectUp: true}
	assert.False(t, unset.DoesExpectValidCert())

	trueVal := true
	setTrue := HTTPSProbeConfig{ExpectUp: true, ExpectValidCertificate: &trueVal}
	assert.True(t, setTrue.DoesExpectValidCert())

	falseVal := false
	setFalse := HTTPSProbeConfig{ExpectUp: true, ExpectValidCertificate: &falseVal}
	assert.False(t, setFalse.DoesExpectValidCert())
}

func TestWebsocketProbeConfigUrl(t *testing.T) {
	nourl := WebsocketProbeConfig{}
	assert.Equal(t, "wss://example.com/ws", nourl.GetURL("example.com"))

	withurl := WebsocketProbeConfig{
		URL: strPtr("ws://custom-url.com/socket"),
	}
	assert.Equal(t, "ws://custom-url.com/socket", withurl.GetURL("example.com"))
}

func TestLoadConfigFromBytes(t *testing.T) {
	yamlData := []byte(`
name: demo
defaults:
  timeout_seconds: 10
  retries: 3
  interval_seconds: 30
probes:
  - id: web
    hostname: www.example.com
    http:
      expect_up: true
    https:
      expect_up: true
      expect_valid_cert: true
    websocket:
      expect_up: true
`)
	cfg, err := LoadConfigFromBytes(yamlData)
	assert.NoError(t, err)
	assert.Equal(t, "demo", cfg.Name)
	assert.Equal(t, 10, cfg.Defaults.TimeoutSeconds)
	assert.Equal(t, 3, cfg.Defaults.Retries)
	assert.Equal(t, 30, cfg.Defaults.IntervalSeconds)
	assert.Len(t, cfg.Probes, 1)
	assert.Equal(t, "web", cfg.Probes[0].ID)
	assert.Equal(t, "www.example.com", cfg.Probes[0].Hostname)
	assert.NotNil(t, cfg.Probes[0].HTTP)
	assert.NotNil(t, cfg.Probes[0].HTTPS)
	assert.NotNil(t, cfg.Probes[0].Websocket)
}

func TestBuildTargets(t *testing.T) {
	trueVal := true
	falseVal := false
	cfg := &Config{
		Name: "test",
		Probes: ProbeConfigList{
			{
				ID:       "p1",
				Hostname: "host1.com",
				HTTP: &HTTPProbeConfig{
					ExpectUp:              true,
					ExpectRedirectToHTTPS: &trueVal,
				},
				HTTPS: &HTTPSProbeConfig{
					ExpectUp:               true,
					ExpectValidCertificate: &falseVal,
				},
				Websocket: &WebsocketProbeConfig{},
			},
			{
				ID:       "p2",
				Hostname: "host2.com",
			},
		},
	}

	targets := cfg.BuildTargets()
	assert.Len(t, targets, 2)

	assert.Equal(t, "p1", targets[0].ID)
	assert.Equal(t, "host1.com", targets[0].Hostname)
	assert.Len(t, targets[0].Checks, 3)

	assert.Equal(t, "p2", targets[1].ID)
	assert.Equal(t, "host2.com", targets[1].Hostname)
	assert.Len(t, targets[1].Checks, 0)
}

func strPtr(s string) *string {
	return &s
}
