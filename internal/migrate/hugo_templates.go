package migrate

// planHugoTemplates translates Hugo layouts into candidate Nida templates
// with Plan.Template. dirs are source-relative layout directories in
// lookup order: the site's layouts/ first, then each theme's.
func planHugoTemplates(p *Plan, dirs []string) {}

var _ = planHugoTemplates // called by the Hugo source once it lands
