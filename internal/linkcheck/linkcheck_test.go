package linkcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCollectAndCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.WriteHeader(http.StatusOK)
		case "/no-head":
			if r.Method == http.MethodHead {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	links := Collect([]Source{
		{Path: "a.md", HTML: `<a href="` + server.URL + `/ok">x</a> <a href="` + server.URL + `/missing#frag">y</a>`},
		{Path: "b.md", HTML: `<a href="` + server.URL + `/missing">z</a> <a href="` + server.URL + `/no-head">w</a> <a href="/local/">l</a>`},
	})
	if len(links) != 3 || len(links[server.URL+"/missing"]) != 2 {
		t.Fatalf("unexpected links %v", links)
	}

	problems := Check(context.Background(), links, server.Client())
	if len(problems) != 1 || problems[0].URL != server.URL+"/missing" || problems[0].Message != "HTTP 404" {
		t.Fatalf("unexpected problems %+v", problems)
	}
}
