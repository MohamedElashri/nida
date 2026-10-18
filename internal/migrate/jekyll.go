package migrate

import "errors"

// jekyllSource migrates Jekyll sites.
type jekyllSource struct{}

func (jekyllSource) Name() string  { return "jekyll" }
func (jekyllSource) Title() string { return "Jekyll" }

func (jekyllSource) Detect(root string) int { return 0 }

func (jekyllSource) Plan(p *Plan) error {
	return errors.New("migration from Jekyll is not implemented yet")
}
