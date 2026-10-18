package migrate

import "errors"

// hugoSource migrates Hugo sites.
type hugoSource struct{}

func (hugoSource) Name() string  { return "hugo" }
func (hugoSource) Title() string { return "Hugo" }

func (hugoSource) Detect(root string) int { return 0 }

func (hugoSource) Plan(p *Plan) error {
	return errors.New("migration from Hugo is not implemented yet")
}
