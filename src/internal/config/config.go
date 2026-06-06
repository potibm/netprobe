package config

type Config struct {
	Name     string          `yaml:"name"`
	Defaults Defaults        `yaml:"defaults"`
	Probes   ProbeConfigList `yaml:"probes"`
}

type Defaults struct {
	TimeoutSeconds  int `yaml:"timeout_seconds"`
	Retries         int `yaml:"retries"`
	IntervalSeconds int `yaml:"interval_seconds"`
}

type ProbeConfigList []ProbeConfig

type ProbeConfig struct {
	ID        string                `yaml:"id"`
	Hostname  string                `yaml:"hostname"`
	HTTP      *HTTPProbeConfig      `yaml:"http,omitempty"`
	HTTPS     *HTTPSProbeConfig     `yaml:"https,omitempty"`
	Websocket *WebsocketProbeConfig `yaml:"websocket,omitempty"`
}

type HTTPProbeConfig struct {
	URL                   *string `yaml:"url,omitempty"`
	ExpectUp              bool    `yaml:"expect_up"`
	ExpectRedirectToHTTPS *bool   `yaml:"expect_redirect_to_https,omitempty"`
}

func (c *HTTPProbeConfig) GetURL(hostname string) string {
	if c.URL != nil {
		return *c.URL
	}

	return "http://" + hostname + "/"
}

func (c *HTTPProbeConfig) DoesExpectRedirectToHTTP() bool {
	if c.ExpectRedirectToHTTPS != nil {
		return *c.ExpectRedirectToHTTPS
	}

	return false
}

type HTTPSProbeConfig struct {
	URL                    *string `yaml:"url,omitempty"`
	ExpectUp               bool    `yaml:"expect_up"`
	ExpectValidCertificate *bool   `yaml:"expect_valid_cert,omitempty"`
}

func (c *HTTPSProbeConfig) GetURL(hostname string) string {
	if c.URL != nil {
		return *c.URL
	}

	return "https://" + hostname + "/"
}

func (c *HTTPSProbeConfig) DoesExpectValidCert() bool {
	if c.ExpectValidCertificate != nil {
		return *c.ExpectValidCertificate
	}

	return false
}

type WebsocketProbeConfig struct {
	URL *string `yaml:"url,omitempty"`
}

func (c *WebsocketProbeConfig) GetURL(hostname string) string {
	if c.URL != nil {
		return *c.URL
	}

	return "wss://" + hostname + "/ws"
}
