package config

import "net/http"

type Application struct {
	Name     string     `hcl:"name,label"`
	Backend  []*Backend `hcl:"backend,block"`
	BasePath string     `hcl:"base_path,attr"`
	Files    Files      `hcl:"files,block"`
	Path     []*Path    `hcl:"path,block"`

	instance http.Handler
}

func (f *Application) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if f.instance != nil {
		f.instance.ServeHTTP(rw, req)
	}
}

func (f *Application) String() string {
	if f.instance != nil {
		return "File"
	}
	return ""
}
