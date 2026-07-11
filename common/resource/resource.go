package resource

import "github.com/eve-online-tools/eve-resfile-proxy/common/platform"

type Resource string

const (
	Full       Resource = "full"
	OSSpecific Resource = "osspecific"
	Prefetch   Resource = "prefetch"
)

var (
	pathMap = map[platform.Platform]map[Resource]string{
		platform.Windows: {
			Full:       "app:/resfileindex.txt",
			OSSpecific: "app:/resfileindex_Windows.txt",
			Prefetch:   "app:/resfileindex_prefetch.txt",
		},
		platform.Mac: {
			Full:       "app:/EVE.app/Contents/Resources/build/resfileindex.txt",
			OSSpecific: "app:/EVE.app/Contents/Resources/build/resfileindex_macOS.txt",
			Prefetch:   "app:/EVE.app/Contents/Resources/build/resfileindex_prefetch.txt",
		},
	}
)

func (r Resource) String() string {
	return string(r)
}

func (r Resource) Path(platform platform.Platform) string {
	return pathMap[platform][r]
}
