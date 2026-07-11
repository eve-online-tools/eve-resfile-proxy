package domain

import "strings"

type Domain string

const (
	Binaries  Domain = "https://binaries.eveonline.com"
	Resources Domain = "https://resources.eveonline.com"
)

var All = []Domain{Binaries, Resources}

func (d Domain) String() string {
	return string(d)
}

func (d Domain) URL(path string) string {
	return strings.TrimRight(d.String(), "/") + "/" + strings.TrimPrefix(path, "/")
}
