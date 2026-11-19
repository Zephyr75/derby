// Package defaults holds Derby's default agents and prompts: the files
// ~/.derby/agents.toml and ~/.derby/prompts.toml start from, in the same
// format (a diff between the two folders shows what you changed).
package defaults

import _ "embed"

// Agents is agents.toml: the default pipeline.
//
//go:embed agents.toml
var Agents []byte

// Prompts is prompts.toml: every text Derby adds to what agents get.
//
//go:embed prompts.toml
var Prompts []byte
