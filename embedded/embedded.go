package embedded

import _ "embed"

// Database migrations.

//go:embed sql/1x0.sql
// DBMigration1x0 is the initial database setup from first version.
var DBMigration1x0 string
