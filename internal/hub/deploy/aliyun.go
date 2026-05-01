package deploy

import (
	"context"
	"fmt"
)

// aliyunProvider is a stub — see sealos.go for rationale. Real adapter
// will require an Aliyun ECS API key, a region/zone, an instance type
// pick, plus SSH-based bootstrap of newhub via docker-compose. None of
// that should be merged un-tested.
type aliyunProvider struct{}

func (aliyunProvider) Kind() Kind { return KindAliyun }

func (aliyunProvider) Provision(_ context.Context, _ Inputs) (*Result, error) {
	return nil, fmt.Errorf("阿里云 ECS 自动部署：%w（请先在阿里云控制台开 ECS 实例并部署 lurus-newhub docker 镜像，再用「手动接入」选项）", ErrNotImplemented)
}
