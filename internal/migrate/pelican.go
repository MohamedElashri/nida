package migrate

import "errors"

// pelicanSource migrates Pelican sites.
type pelicanSource struct{}

func (pelicanSource) Name() string  { return "pelican" }
func (pelicanSource) Title() string { return "Pelican" }

func (pelicanSource) Detect(root string) int { return 0 }

func (pelicanSource) Plan(p *Plan) error {
	return errors.New("migration from Pelican is not implemented yet")
}
