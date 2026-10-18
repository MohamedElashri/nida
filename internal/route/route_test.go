package route

import "testing"

func TestPage(t *testing.T) {
	cases := map[string]string{
		"/posts/hello/":            "/posts/hello/",
		"/posts/hello":             "/posts/hello/",
		"posts/hello":              "/posts/hello/",
		"/":                        "/",
		"/2024/05/01/hello.html":   "/2024/05/01/hello.html",
		"2024/05/01/hello.html":    "/2024/05/01/hello.html",
		"/about.htm":               "/about.htm",
		"/v1.2/":                   "/v1.2/",
		"/posts//hello.html":       "",
		"/posts/../../etc/passwd":  "",
		"/posts/hello.html?x=1":    "",
		"   /trimmed/   ":          "/trimmed/",
		"/files/report.final.html": "/files/report.final.html",
		"/dotted.dir/inner":        "/dotted.dir/inner/",
		"/posts/hello.html/":       "/posts/hello.html/",
		"/posts/hello\\world.html": "",
	}
	for in, want := range cases {
		got, err := Page("path", in)
		if want == "" {
			if err == nil {
				t.Errorf("Page(%q) = %q, want an error", in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("Page(%q): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("Page(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDir(t *testing.T) {
	cases := map[string]string{
		"/posts/hello/":          "/posts/hello/",
		"/2024/05/01/hello.html": "/2024/05/01/",
		"/hello.html":            "/",
		"/":                      "/",
	}
	for in, want := range cases {
		if got := Dir(in); got != want {
			t.Errorf("Dir(%q) = %q, want %q", in, got, want)
		}
	}
}
