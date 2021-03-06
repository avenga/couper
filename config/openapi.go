package config

// OpenAPI represents the <OpenAPI> object.
type OpenAPI struct {
	File                     string `hcl:"file"`
	IgnoreRequestViolations  bool   `hcl:"ignore_request_violations,optional"`
	IgnoreResponseViolations bool   `hcl:"ignore_response_violations,optional"`
}
