// Package seed embeds the committed master-data CSVs so the seed binary
// needs no files at runtime.
package seed

import "embed"

//go:embed *.csv
var FS embed.FS
