// Package assets embeds the decoded native artwork and soundtrack.
package assets
import "embed"
//go:embed original/*
var Files embed.FS
