// Package defaults holds Herd's default agents and prompts: the files
// ~/.herd/agents.toml and ~/.herd/prompts.toml start from, in the same
// format (a diff between the two folders shows what you changed).
package defaults

import _ "embed"

// Agents is agents.toml: the default pipeline.
//
//go:embed agents.toml
var Agents []byte

// Prompts is prompts.toml: every text Herd adds to what agents get.
//
//go:embed prompts.toml
var Prompts []byte
