package deploy

import (
	"context"
	"fmt"
)

// sealosProvider is a stub. Real implementation is deferred to a session
// where Sealos cluster credentials are available — building 600 lines of
// unverified Sealos client code violates Goal-Driven Execution (no test
// to drive correctness).
//
// Day 1 plan: surface ErrNotImplemented so the wizard can route the user
// to the manual path with a clear message.
type sealosProvider struct{}

func (sealosProvider) Kind() Kind { return KindSealos }

func (sealosProvider) Provision(_ context.Context, _ Inputs) (*Result, error) {
	return nil, fmt.Errorf("Sealos 自动部署：%w（请先在 Sealos 控制台手动部署 lurus-newhub，再用「手动接入」选项）", ErrNotImplemented)
}
