package lib_test

import (
	"testing"
	"time"

	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config/configload"
)

func TestUnixtime(t *testing.T) {
	tests := []struct {
		name string
		hcl  string
		want int64
	}{
		{
			"unixtime",
			`
			server "test" {
			}
			`,
			time.Now().Unix(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf, err := configload.LoadBytes([]byte(tt.hcl), "couper.hcl")
			now, err := cf.Context.HTTPContext(0, nil).Functions["unixtime"].Call([]cty.Value{})
			if err != nil {
				t.Fatal(err)
			}
			if !cty.Number.Equals(now.Type()) {
				t.Errorf("Wrong return type; expected %s, got: %s", cty.Number.FriendlyName(), now.Type().FriendlyName())
			}
			bfnow := now.AsBigFloat()
			inow, _ := bfnow.Int64()
			if inow != tt.want {
				t.Errorf("Wrong return value; expected %d, got: %d", tt.want, inow)
			}
		})
	}
}
