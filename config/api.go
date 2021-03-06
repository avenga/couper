package config

// API represents the <API> object.
type API struct {
	AccessControl        []string  `hcl:"access_control,optional"`
	CORS                 *CORS     `hcl:"cors,block"`
	BasePath             string    `hcl:"base_path,optional"`
	DisableAccessControl []string  `hcl:"disable_access_control,optional"`
	Endpoints            Endpoints `hcl:"endpoint,block"`
	ErrorFile            string    `hcl:"error_file,optional"`
}

// APIs represents a list of <API> objects.
type APIs []*API
