package main

import (
	"fmt"
	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/site"
)

func main() {
	cfg, _, err := config.Load(config.Options{SiteRoot: "/home/melashri/projects/nida/MohamedElashri.github.io"})
	if err != nil {
		fmt.Println("Config Error:", err)
		return
	}
	state, err := site.Load("/home/melashri/projects/nida/MohamedElashri.github.io", cfg)
	if err != nil {
		fmt.Println("Site Load Error:", err)
		return
	}
	for _, p := range state.Index.AllPages {
		if p.SectionPath == "publications" || p.SectionPath == "primary-publications" || p.SectionPath == "talks" || p.SectionPath == "teaching" {
			fmt.Printf("Page: %s | Template: %s | Section: %s | Resources: %v\n", p.Slug, p.Template, p.SectionPath, p.Resources)
		}
	}
}
