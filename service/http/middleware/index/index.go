package index

import (
	"bytes"
	_ "embed"
	"errors"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/eve-online-tools/eve-resfile-proxy/common/mimetype"
	"github.com/eve-online-tools/eve-resfile-proxy/service/buildnumber"
	"github.com/eve-online-tools/eve-resfile-proxy/service/http/urlpath"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs"
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

func Middleware(enabled bool, fsys fs.FS, build *buildnumber.BuildNumber) func(http.Handler) http.Handler {
	if !enabled {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			urlPath := urlpath.CleanURL(r.URL.Path)
			if !strings.HasSuffix(urlPath, "/") {
				next.ServeHTTP(w, r)
				return
			}

			dirPath, err := urlpath.DirFromURL(urlPath)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			rdFS, ok := fsys.(fs.ReadDirFS)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			entries, err := rdFS.ReadDir(dirPath)
			if errors.Is(err, fs.ErrNotExist) {
				next.ServeHTTP(w, r)
				return
			}
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			buildNumber := ""
			if build != nil {
				buildNumber = build.Get()
			}
			data := buildListingData(r.URL, buildNumber, entries)
			var buf bytes.Buffer
			if err := listingTemplate.Execute(&buf, data); err != nil {
				http.Error(w, "Failed to render directory listing.", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(buf.Bytes())
		})
	}
}

func buildListingData(u *url.URL, buildNumber string, entries []fs.DirEntry) listingData {
	path := urlpath.CleanURL(u.Path)

	data := listingData{
		Path:        path,
		BuildNumber: buildNumber,
		Breadcrumbs: buildBreadcrumbs(path),
		Entries:     make([]listingLink, 0, len(entries)),
	}

	if path != "/" {
		data.Parent = parentPath(path)
		data.ParentIcon = ParentIcon()
	}

	for _, entry := range entries {
		name := entry.Name()
		var href string
		if entry.IsDir() {
			href = urlpath.JoinURL(path, name) + "/"
		} else {
			href = urlpath.JoinURL(path, name)
		}

		label := name
		if entry.IsDir() {
			label += "/"
		}

		link := listingLink{
			Href:  template.URL(href),
			Label: label,
			Icon:  IconFor(name, entry.IsDir()),
		}
		if entry.IsDir() {
			link.FileType = "folder"
			link.FileSize = "-"
			link.Compression = "-"
		} else {
			link.FileType = mimetype.ForFilename(name)
			if info, err := entry.Info(); err == nil {
				link.FileSize = FormatFileSize(info.Size())
				if li, ok := info.(vfs.CompressedSizeInfo); ok {
					link.Compression = FormatCompressionPercent(info.Size(), li.CompressedSize())
				}
			}
			if link.FileSize == "" {
				link.FileSize = "-"
			}
			if link.Compression == "" {
				link.Compression = "-"
			}
		}

		data.Entries = append(data.Entries, link)
	}

	return data
}

func buildBreadcrumbs(urlPath string) []listingCrumb {
	path := urlpath.CleanURL(urlPath)
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}

	if path == "/" {
		return []listingCrumb{{Label: "res:", Current: true}}
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	crumbs := make([]listingCrumb, 0, len(parts)+1)
	crumbs = append(crumbs, listingCrumb{
		Href:  template.URL("/"),
		Label: "res:",
	})

	accumulated := ""
	for i, part := range parts {
		accumulated = urlpath.JoinURL(accumulated, part) + "/"
		crumb := listingCrumb{Label: part}
		if i == len(parts)-1 {
			crumb.Current = true
		} else {
			crumb.Href = template.URL(accumulated)
		}
		crumbs = append(crumbs, crumb)
	}

	return crumbs
}

func parentPath(urlPath string) string {
	path := urlpath.CleanURL(urlPath)
	trimmed := strings.TrimSuffix(path, "/")
	if trimmed == "" || trimmed == "/" {
		return "/"
	}
	if idx := strings.LastIndex(trimmed, "/"); idx >= 0 {
		return trimmed[:idx+1]
	}
	return "/"
}

func FormatFileSize(size int64) string {
	return formatFileSize(size)
}

func FormatCompressionPercent(size, compressed int64) string {
	return formatCompressionPercent(size, compressed)
}
