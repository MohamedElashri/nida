package render

import (
	"testing"
	"time"

	"github.com/MohamedElashri/nida/internal/content"
)

func TestBuildPaginatorUsesConfiguredPath(t *testing.T) {
	paginator := buildPaginator("/posts/", "p", 2, 3, nil)

	if paginator.Previous != "/posts/" {
		t.Fatalf("expected first page to use the section URL, got %q", paginator.Previous)
	}
	if paginator.Next != "/posts/p/3/" {
		t.Fatalf("expected custom paginate path, got %q", paginator.Next)
	}
	if paginator.PageLinks[1].URL != "/posts/p/2/" || !paginator.PageLinks[1].Current {
		t.Fatalf("unexpected page link %+v", paginator.PageLinks[1])
	}
}

func TestLatestItemsFiltersSectionsAndAllowsUnlimited(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 1, d, 0, 0, 0, 0, time.UTC) }
	items := []content.Page{
		{Title: "Note", SectionPath: "notes", Date: day(3)},
		{Title: "Old post", SectionPath: "posts", Date: day(1)},
		{Title: "New post", SectionPath: "posts/2026", Date: day(2)},
	}

	got := latestItems(items, []string{"posts"}, 0)
	if len(got) != 2 || got[0].Title != "New post" || got[1].Title != "Old post" {
		t.Fatalf("unexpected filtered items %+v", got)
	}
	if got := latestItems(items, nil, 1); len(got) != 1 || got[0].Title != "Note" {
		t.Fatalf("expected newest single item, got %+v", got)
	}
}

func TestJoinRobots(t *testing.T) {
	if got := joinRobots("noindex", ""); got != "noindex" {
		t.Fatalf("got %q", got)
	}
	if got := joinRobots("noindex", "noai"); got != "noindex, noai" {
		t.Fatalf("got %q", got)
	}
}
