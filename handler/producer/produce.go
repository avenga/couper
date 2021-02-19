package producer

import (
	"context"
	"net/http"

	"github.com/avenga/couper/eval"
)

var (
	_ Roundtrips = Proxies{}
	_ Roundtrips = Requests{}
)

type Roundtrips interface {
	Produce(ctx context.Context, req *http.Request, evalCtx *eval.HTTP, results chan<- *Result)
}
