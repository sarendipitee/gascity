// Package beadsdoltliteinit embeds the minimal DoltLite init support pack.
package beadsdoltliteinit

import "embed"

// PackFS contains the minimal DoltLite init support pack files.
//
//go:embed pack.toml all:assets
var PackFS embed.FS
