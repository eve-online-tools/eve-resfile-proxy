package service

import (
	"github.com/eve-online-tools/eve-resfile-proxy/common/platform"
	svchttp "github.com/eve-online-tools/eve-resfile-proxy/service/http"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs/rewrite"
	vfstransform "github.com/eve-online-tools/eve-resfile-proxy/vfs/transform"
)

type Config struct {
	ServerName      string
	BuildNumber     string
	Platforms       []platform.Platform
	CacheDir        string
	FullTree        bool
	Rewrites        []rewrite.Rule
	Transforms      []vfstransform.Transform
	TransformLimits vfstransform.Limits
	ConfigDir       string

	ServerConfig svchttp.ServerConfig
}
