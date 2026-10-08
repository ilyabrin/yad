// Command demo serves a fake Yandex.Disk API filled with sample files, so the
// README animation can be recorded without touching a real account.
//
// It answers only what yad asks for during the recording, in the same shapes
// the real API uses. Point yad at it with YANDEX_DISK_API_URL:
//
//	go run ./tools/demo &
//	YANDEX_DISK_TOKEN=demo YANDEX_DISK_API_URL=http://127.0.0.1:8765/v1/disk yad
//
// See tools/demo/demo.tape for the recording itself.
package main

import (
	"cmp"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

// file is one entry of the sample Disk.
type file struct {
	name     string
	size     int64
	modified string
	mime     string
	media    string
}

func dir(name, modified string) file { return file{name: name, modified: modified} }

func (f file) isDir() bool { return f.mime == "" && f.size == 0 }

// tree is the sample Disk, keyed by folder path.
var tree = map[string][]file{
	"/": {
		dir("Documents", "2026-09-28T09:12:00+00:00"),
		dir("Photos", "2026-10-02T18:40:00+00:00"),
		dir("Projects", "2026-10-05T11:03:00+00:00"),
		dir("Music", "2026-06-14T20:15:00+00:00"),
		dir("Отпуск 2026", "2026-08-30T16:22:00+00:00"),
		{"backup-2026.zip", 5_368_709_120, "2026-09-15T03:00:00+00:00", "application/zip", "compressed"},
		{"budget-2026.xlsx", 48_230, "2026-09-30T08:41:00+00:00", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "spreadsheet"},
		{"notes.md", 6_144, "2026-10-06T21:10:00+00:00", "text/markdown", "document"},
		{"presentation.pdf", 2_412_544, "2026-10-01T14:55:00+00:00", "application/pdf", "document"},
	},
	"/Photos": {
		dir("Screenshots", "2026-09-12T10:00:00+00:00"),
		{"iceland-glacier.jpg", 4_812_301, "2026-08-14T12:31:00+00:00", "image/jpeg", "image"},
		{"iceland-waterfall.jpg", 5_209_887, "2026-08-15T09:02:00+00:00", "image/jpeg", "image"},
		{"iceland-aurora.jpg", 3_998_120, "2026-08-16T23:47:00+00:00", "image/jpeg", "image"},
		{"kyoto-temple.jpg", 4_102_554, "2026-04-03T07:15:00+00:00", "image/jpeg", "image"},
		{"lisbon-tram.jpg", 3_671_009, "2026-05-21T17:44:00+00:00", "image/jpeg", "image"},
		{"family-dinner.heic", 2_884_190, "2026-09-20T19:30:00+00:00", "image/heic", "image"},
	},
	"/Documents": {
		{"contract-2026.pdf", 812_441, "2026-03-11T10:20:00+00:00", "application/pdf", "document"},
		{"invoice-2026-09.pdf", 96_512, "2026-09-30T16:05:00+00:00", "application/pdf", "document"},
		{"resume.docx", 41_203, "2026-07-02T13:48:00+00:00", "application/msword", "document"},
	},
}

// trash holds the sample deleted items.
var trash = []struct {
	file
	origin, deleted string
}{
	{file{"old-draft.docx", 38_912, "2026-09-02T11:00:00+00:00", "application/msword", "document"}, "disk:/Documents/old-draft.docx", "2026-10-04T09:15:00+00:00"},
	{file{"blurry-photo.jpg", 3_201_455, "2026-08-16T22:10:00+00:00", "image/jpeg", "image"}, "disk:/Photos/blurry-photo.jpg", "2026-10-05T18:02:00+00:00"},
}

// published records which files have been shared, and starts with one.
var (
	mu        sync.Mutex
	published = map[string]bool{"/presentation.pdf": true}
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8765", "address to listen on")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/disk/{$}", diskInfo)
	mux.HandleFunc("GET /v1/disk/resources", resources)
	mux.HandleFunc("PUT /v1/disk/resources/publish", publish(true))
	mux.HandleFunc("PUT /v1/disk/resources/unpublish", publish(false))
	mux.HandleFunc("PATCH /v1/disk/public/resources/public-settings", publicSettings)
	mux.HandleFunc("GET /v1/disk/trash/resources", trashList)
	mux.HandleFunc("/", notFound)

	log.Printf("fake Yandex.Disk API on http://%s/v1/disk", *addr)
	srv := &http.Server{Addr: *addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(srv.ListenAndServe())
}

func diskInfo(w http.ResponseWriter, _ *http.Request) {
	reply(w, http.StatusOK, map[string]any{
		"total_space":   int64(2_199_023_255_552), // 2 TB, a real Yandex 360 plan
		"used_space":    int64(1_317_351_872_102),
		"trash_size":    7_240_367,
		"max_file_size": int64(53_687_091_200),
		"is_paid":       true,
		"user":          map[string]string{"login": "demo", "display_name": "Demo User"},
	})
}

func resources(w http.ResponseWriter, r *http.Request) {
	p := clean(r.URL.Query().Get("path"))

	if entries, ok := tree[p]; ok {
		items := sorted(entries, r.URL.Query().Get("sort"))
		limit := atoi(r.URL.Query().Get("limit"), 20)
		offset := atoi(r.URL.Query().Get("offset"), 0)
		page := items[min(offset, len(items)):min(offset+limit, len(items))]

		list := make([]map[string]any, 0, len(page))
		for _, f := range page {
			list = append(list, resource(path.Join(p, f.name), f))
		}
		folder := resource(p, dir(path.Base(p), "2026-10-06T00:00:00+00:00"))
		folder["_embedded"] = map[string]any{
			"path": "disk:" + p, "items": list, "total": len(items),
			"limit": limit, "offset": offset, "sort": r.URL.Query().Get("sort"),
		}
		reply(w, http.StatusOK, folder)
		return
	}

	if f, ok := lookup(p); ok {
		reply(w, http.StatusOK, resource(p, f))
		return
	}
	notFound(w, r)
}

func publish(on bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := clean(r.URL.Query().Get("path"))
		if _, ok := lookup(p); !ok {
			notFound(w, r)
			return
		}
		mu.Lock()
		published[p] = on
		mu.Unlock()
		reply(w, http.StatusOK, map[string]any{
			"href":   "http://" + r.Host + "/v1/disk/resources?path=" + "disk:" + p,
			"method": http.MethodGet,
		})
	}
}

// publicSettings accepts link protection for a published file, as the real
// API does after publishing.
func publicSettings(w http.ResponseWriter, r *http.Request) {
	p := clean(r.URL.Query().Get("path"))
	mu.Lock()
	ok := published[p]
	mu.Unlock()
	if !ok {
		notFound(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// trashList answers in the real shape: the trash root is a resource and its
// contents sit under "_embedded".
func trashList(w http.ResponseWriter, _ *http.Request) {
	items := make([]map[string]any, 0, len(trash))
	for i, t := range trash {
		item := resource("", t.file)
		item["path"] = "trash:/" + t.name + "_" + strconv.Itoa(1728000000+i)
		item["origin_path"] = t.origin
		item["deleted"] = t.deleted
		items = append(items, item)
	}
	reply(w, http.StatusOK, map[string]any{
		"path": "trash:/", "name": "trash", "type": "dir",
		"created": "2024-01-01T00:00:00+00:00", "modified": "2026-10-05T18:02:00+00:00",
		"_embedded": map[string]any{
			"path": "trash:/", "items": items, "total": len(items), "limit": 100, "offset": 0,
		},
	})
}

func notFound(w http.ResponseWriter, _ *http.Request) {
	reply(w, http.StatusNotFound, map[string]string{
		"error": "DiskNotFoundError", "message": "Не удалось найти запрошенный ресурс.",
		"description": "Resource not found.",
	})
}

// resource renders an entry the way the API does.
func resource(p string, f file) map[string]any {
	r := map[string]any{
		"path": "disk:" + p, "name": f.name, "created": f.modified, "modified": f.modified,
	}
	if f.isDir() {
		r["type"] = "dir"
		return r
	}
	r["type"] = "file"
	r["size"] = f.size
	r["mime_type"] = f.mime
	r["media_type"] = f.media
	mu.Lock()
	defer mu.Unlock()
	if published[p] {
		r["public_url"] = "https://yadi.sk/d/" + shortID(p)
		r["public_key"] = shortID(p)
	}
	return r
}

// sorted orders folders before files, as the real API does whatever the sort,
// then by the requested field.
func sorted(entries []file, by string) []file {
	out := slices.Clone(entries)
	desc := strings.HasPrefix(by, "-")
	field := strings.TrimPrefix(by, "-")
	slices.SortStableFunc(out, func(a, b file) int {
		if a.isDir() != b.isDir() {
			if a.isDir() {
				return -1
			}
			return 1
		}
		var c int
		switch field {
		case "size":
			c = cmp.Compare(a.size, b.size)
		case "modified":
			c = cmp.Compare(a.modified, b.modified)
		default:
			c = cmp.Compare(strings.ToLower(a.name), strings.ToLower(b.name))
		}
		if desc {
			c = -c
		}
		return c
	})
	return out
}

func lookup(p string) (file, bool) {
	for _, f := range tree[path.Dir(p)] {
		if f.name == path.Base(p) {
			return f, true
		}
	}
	return file{}, false
}

// clean turns "disk:/a/b", "/a/b/" and "" into "/a/b" or "/".
func clean(p string) string {
	p = strings.TrimPrefix(p, "disk:")
	if p == "" {
		return "/"
	}
	return path.Clean("/" + p)
}

func shortID(p string) string {
	h := 0
	for _, c := range p {
		h = h*31 + int(c)
	}
	return strconv.FormatInt(int64(h&0xffffff), 36) + "demo"
}

func atoi(s string, fallback int) int {
	if n, err := strconv.Atoi(s); err == nil && n >= 0 {
		return n
	}
	return fallback
}

func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
