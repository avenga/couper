package config

// Definitions represents the <Definitions> object.
type Definitions struct {
	BasicAuth []*BasicAuth `hcl:"basic_auth,block"`
	JWT       []*JWT       `hcl:"jwt,block"`
	SAML      []*SAML      `hcl:"saml,block"`
}
