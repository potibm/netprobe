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

func strPtr(s string) *string {
	return &s
}
