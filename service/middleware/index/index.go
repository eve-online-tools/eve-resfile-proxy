package index

import (
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	resindex "github.com/eve-online-tools/eve-resfile-proxy/internal/index"
	"github.com/eve-online-tools/eve-resfile-proxy/internal/mimetype"
)

//go:embed listing.html
var listingHTML string

var listingTemplate = template.Must(template.New("listing").Parse(listingHTML))

type listingData struct {
	Path        string
	BuildNumber string
	Parent      string
	ParentIcon  template.URL
	Breadcrumbs []listingCrumb
	Entries     []listingLink
}

type listingCrumb struct {
	Href    template.URL
	Label   string
	Current bool
}

type listingLink struct {
	Href        template.URL
	Label       string
	Icon        template.URL
	FileType    string
	FileSize    string
	Compression string
}

func Middleware(enabled bool, indexSet *resindex.IndexSet) func(http.Handler) http.Handler {
	if !enabled {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/") {
				next.ServeHTTP(w, r)
				return
			}

			pref, err := resindex.ParsePlatform(r.URL.Query().Get("platform"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			dirPrefix := resindex.DirPrefixFromURL(r.URL.Path)
			entries, ok := indexSet.ListDirectory(dirPrefix, pref)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_ = listingTemplate.Execute(w, buildListingData(r.URL, indexSet.BuildNumber, entries))
		})
	}
}

func buildListingData(u *url.URL, buildNumber string, entries []resindex.ListingEntry) listingData {
	path := u.Path
	if path == "" {
		path = "/"
	}

	data := listingData{
		Path:        path,
		BuildNumber: buildNumber,
		Breadcrumbs: buildBreadcrumbs(u),
		Entries:     make([]listingLink, 0, len(entries)),
	}

	if u.Path != "/" {
		parent := parentPath(u.Path)
		data.Parent = linkHref(parent, u.Query())
		data.ParentIcon = ParentIcon()
	}

	for _, entry := range entries {
		var href string
		if entry.IsDir {
			href = linkHref(pathJoin(u.Path, entry.Name)+"/", u.Query())
		} else {
			href = linkHref(pathJoin(u.Path, entry.Name), u.Query())
		}

		label := entry.Name
		if entry.IsDir {
			label += "/"
		}

		link := listingLink{
			Href:  template.URL(href),
			Label: label,
			Icon:  IconFor(entry.Name, entry.IsDir),
		}
		if entry.IsDir {
			link.FileType = "folder"
			link.FileSize = "-"
			link.Compression = "-"
		} else {
			link.FileType = mimetype.ForFilename(entry.Name)
			link.FileSize = resindex.FormatFileSize(entry.Size)
			link.Compression = resindex.FormatCompressionPercent(entry.Size, entry.CompressedSize)
		}

		data.Entries = append(data.Entries, link)
	}

	return data
}

func buildBreadcrumbs(u *url.URL) []listingCrumb {
	path := u.Path
	if path == "" {
		path = "/"
	}
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}

	if path == "/" {
		return []listingCrumb{{Label: "res:", Current: true}}
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	crumbs := make([]listingCrumb, 0, len(parts)+1)
	crumbs = append(crumbs, listingCrumb{
		Href:  template.URL(linkHref("/", u.Query())),
		Label: "res:",
	})

	accumulated := ""
	for i, part := range parts {
		accumulated += "/" + part + "/"
		crumb := listingCrumb{Label: part}
		if i == len(parts)-1 {
			crumb.Current = true
		} else {
			crumb.Href = template.URL(linkHref(accumulated, u.Query()))
		}
		crumbs = append(crumbs, crumb)
	}

	return crumbs
}

func parentPath(p string) string {
	trimmed := strings.TrimSuffix(p, "/")
	if trimmed == "" {
		return "/"
	}
	if idx := strings.LastIndex(trimmed, "/"); idx >= 0 {
		return trimmed[:idx+1]
	}
	return "/"
}

func pathJoin(base, name string) string {
	if base == "/" {
		return "/" + name
	}
	return strings.TrimSuffix(base, "/") + "/" + name
}

func linkHref(path string, query url.Values) string {
	if len(query) == 0 {
		return path
	}
	return fmt.Sprintf("%s?%s", path, query.Encode())
}
