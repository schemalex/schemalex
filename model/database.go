package model

// NewDatabase creates a new database mode with th given name
func NewDatabase(n string) Database {
	return &database{
		name: n,
	}
}

func (d *database) isDatabase() bool {
	return true
}

func (d *database) ID() string {
	return "database#" + d.name
}

func (d *database) Name() string {
	return d.name
}

func (d *database) IsIfNotExists() bool {
	return d.ifnotexists
}

func (d *database) SetIfNotExists(v bool) Database {
	d.ifnotexists = v
	return d
}
