package config

import (
	"github.com/avenga/couper/config/saml"
)

// Definitions represents the <Definitions> object.
type Definitions struct {
	BasicAuth []*BasicAuth `hcl:"basic_auth,block"`
	JWT       []*JWT       `hcl:"jwt,block"`
	SAML      []*saml.SAML `hcl:"saml,block"`
}
