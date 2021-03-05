package lib

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function/stdlib"

	"github.com/avenga/couper/config/configload"
)

func TestJwtSign(t *testing.T) {
	tests := []struct {
		name      string
		hcl       string
		jspLabel  string
		claims    string
		want      string
	}{
		{
			"HS256 / key",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "HS256"
					key = "$3cRe4"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.Hf-ZtIlsxR2bDOdAEMaDHaOBmfVWTQi9U68yV4YHW9w",
		},
		{
			"RS256 / key",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "RS256"
					key = <<EOF
-----BEGIN RSA PRIVATE KEY-----
MIICWwIBAAKBgQDGSd+sSTss2uOuVJKpumpFAamlt1CWLMTAZNAabF71Ur0P6u83
3RhAIjXDSA/QeVitzvqvCZpNtbOJVegaREqLMJqvFOUkFdLNRP3f9XjYFFvubo09
tcjX6oGEREKDqLG2MfZ2Z8LVzuJc6SwZMgVFk/63rdAOci3W9u3zOSGj4QIDAQAB
AoGAMzI1rw0FW1J0wLkTWQFJmOGSBLhs9Sk/75DX7kqWxe6D5A07kIfkUALFMNN1
SdVa4R10uibXkULdxRLKJ6YEPLGAN3UmdbnBGxZ+fHAKY3PxM5lL9d7ET08A0u/8
6vB+GZ8w0eqsp4EFzmXI5LS63cRo9GA5iliGpKWtd2IUA2UCQQDnZHJTHW21vrXv
GqXoPxOoQAflxvnHYDgNQcRJxlEokFmSK405n7G2//NrsSnXYmUsA/wdh9YsAYZ3
4xy6hKE3AkEA22Aw58FnypcRAKBTqEWHv957szAmz9R6mLJqG7283YWXL0VGDOuR
qdC4QjMrix3O8WbJxGNaVCrvYKVtKEfPpwJAGGWw4C6UKLuI90L6BzjPW8gUjRej
sm/kuREcHyM3320I5K6O32qFFGR8R/iQDtOjEzcAWCTAYjdu9CkQGGJvlQJAHpCR
X8jfmCdiFA9CeKBvYHk0DOw5jB1Tk3DQPds6tDaHsOta7jPoEJvnADo25+QYUCP9
GqKpFC8DORjzU3hl4wJACEzmqzAco2M4mVc+PxPX0b3LHaREyXURd+faFXUecxSF
BShcGHZl9nzWDtEZzgdX7cbG5nRUo1+whzBQdYoQmg==
-----END RSA PRIVATE KEY-----
					EOF
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.oSS8rC1KonyZ-JZTZhkqZb5bN0_2Lrbl4J33nLgWroc5vDvmLW0KnX0RQfXy0OjX4uBBYTThActqqqM6vidaXmBfsQ77uB9narWeAptRnKqEPlY-onTHDmTMCz7vQ9wbLT7Aa6MYlhRqKX5adpPPbwBUuhm2I-yMF80nSmFpSk0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf, err := configload.LoadBytes([]byte(tt.hcl), "couper.hcl")
			if err != nil {
				t.Fatal(err)
			}
			claims, err := stdlib.JSONDecode(cty.StringVal(tt.claims))
			if err != nil {
				t.Fatal(err)
			}
			token, err := cf.Context.Functions["jwt_sign"].Call([]cty.Value{cty.StringVal(tt.jspLabel), claims})
			if err != nil {
				t.Fatal(err)
			}
			if token.AsString() != tt.want {
				t.Errorf("Expected %q, got: %#v", tt.want, token.AsString())
			}
		})
	}
}