package embedded

import _ "embed"

// Database migrations.

//go:embed sql/1x0.sql
// DBMigration1x0 is the initial database setup from first version.
var DBMigration1x0 string

//go:embed sql/1x1.sql
var DBMigration1x1 string

//go:embed sql/1x2.sql
var DBMigration1x2 string
