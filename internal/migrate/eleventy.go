package migrate

import "errors"

// eleventySource migrates Eleventy sites.
type eleventySource struct{}

func (eleventySource) Name() string  { return "eleventy" }
func (eleventySource) Title() string { return "Eleventy" }

func (eleventySource) Detect(root string) int { return 0 }

func (eleventySource) Plan(p *Plan) error {
	return errors.New("migration from Eleventy is not implemented yet")
}
