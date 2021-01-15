package command

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/rs/xid"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/config/env"
	"github.com/avenga/couper/internal/test"
)

func TestNewRun(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	log, hook := logrustest.NewNullLogger()
	//log.Out = os.Stdout

	tests := []struct {
		name     string
		file     string
		args     Args
		envs     []string
		settings *config.Settings
	}{
		{"defaults from file", "01_defaults.hcl", nil, nil, &config.DefaultSettings},
		{"overrides from file", "02_changed_defaults.hcl", nil, nil, &config.Settings{
			DefaultPort:     9090,
			HealthPath:      "/status/health",
			NoProxyFromEnv:  true,
			LogFormat:       config.DefaultSettings.LogFormat,
			RequestIDFormat: "uuid4",
			XForwardedHost:  true,
		}},
		{"defaults with flag port", "01_defaults.hcl", Args{"-p", "9876"}, nil, &config.Settings{
			DefaultPort:     9876,
			HealthPath:      config.DefaultSettings.HealthPath,
			LogFormat:       config.DefaultSettings.LogFormat,
			RequestIDFormat: config.DefaultSettings.LogFormat,
		}},
		{"defaults with flag and env port", "01_defaults.hcl", Args{"-p", "9876"}, []string{"COUPER_DEFAULT_PORT=4561"}, &config.Settings{
			DefaultPort:     4561,
			HealthPath:      config.DefaultSettings.HealthPath,
			LogFormat:       config.DefaultSettings.LogFormat,
			RequestIDFormat: config.DefaultSettings.LogFormat,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			helper := test.New(t)
			ctx, shutdown := context.WithCancel(test.NewContext(context.Background()))
			defer shutdown()
			defer helper.Must(os.Chdir(wd))

			runCmd := NewRun(ctx)
			if runCmd == nil {
				t.Error("create run cmd failed")
				return
			}

			couperFile, err := configload.LoadFile(filepath.Join(wd, "testdata/settings", tt.file))
			helper.Must(err)

			if len(tt.envs) > 0 {
				env.OsEnviron = func() []string {
					return tt.envs
				}
				defer func() { env.OsEnviron = os.Environ }()
			}

			go func() {
				helper.Must(runCmd.Execute(tt.args, couperFile, log.WithContext(ctx)))
			}()
			time.Sleep(time.Second / 4) // sync <> execute, TODO: lock or config chan update
			if !reflect.DeepEqual(couperFile.Settings, tt.settings) {
				t.Errorf("Settings differ:\nwant:\t%#v\ngot:\t%#v\n", tt.settings, couperFile.Settings)
			}

			hook.Reset()

			res, err := test.NewHTTPClient().Get("http://localhost:" + strconv.Itoa(couperFile.Settings.DefaultPort) + couperFile.Settings.HealthPath)
			helper.Must(err)

			if res.StatusCode != http.StatusOK {
				subT.Errorf("expected OK, got: %d", res.StatusCode)
			}

			uid := hook.LastEntry().Data["uid"].(string)
			xidLen := len(xid.New().String())
			if couperFile.Settings.RequestIDFormat == "uuid4" {
				if len(uid) <= xidLen {
					t.Errorf("expected uuid4 format, got: %s", uid)
				}
			} else if len(uid) > xidLen {
				t.Errorf("expected common id format, got: %s", uid)
			}
		})
		time.Sleep(time.Second / 2) // shutdown
	}
}
