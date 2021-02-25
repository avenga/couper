package config

// Definitions represents the <Definitions> object.
type Definitions struct {
	BasicAuth []*BasicAuth `hcl:"basic_auth,block"`
	JWT       []*JWT       `hcl:"jwt,block"`
	SAML2     []*SAML2     `hcl:"saml2,block"`
}
