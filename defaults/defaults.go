// Package defaults holds Jockey's default agents and prompts: the files
// ~/.jockey/agents.toml and ~/.jockey/prompts.toml start from, in the same
// format (a diff between the two folders shows what you changed).
package defaults

import _ "embed"

// Agents is agents.toml: the default pipeline.
//
//go:embed agents.toml
var Agents []byte

// Prompts is prompts.toml: every text Jockey adds to what agents get.
//
//go:embed prompts.toml
var Prompts []byte
