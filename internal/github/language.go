package github

import (
	"path/filepath"
	"strings"
)

// extensionToLanguage maps file extensions to language names.
var extensionToLanguage = map[string]string{
	".go":    "go",
	".py":    "python",
	".pyw":   "python",
	".js":    "javascript",
	".jsx":   "javascript",
	".mjs":   "javascript",
	".ts":    "typescript",
	".tsx":   "typescript",
	".java":  "java",
	".kt":    "kotlin",
	".scala": "scala",
	".c":     "c",
	".h":     "c",
	".cpp":   "cpp",
	".cc":    "cpp",
	".cxx":   "cpp",
	".hpp":   "cpp",
	".cs":    "csharp",
	".rs":    "rust",
	".rb":    "ruby",
	".php":   "php",
	".swift": "swift",
	".m":     "objective-c",
	".mm":    "objective-c",
	".sh":    "shell",
	".bash":  "shell",
	".zsh":   "shell",
	".yaml":  "yaml",
	".yml":   "yaml",
	".json":  "json",
	".xml":   "xml",
	".html":  "html",
	".css":   "css",
	".scss":  "scss",
	".sql":   "sql",
	".r":     "r",
	".lua":   "lua",
	".dart":  "dart",
	".ex":    "elixir",
	".exs":   "elixir",
	".erl":   "erlang",
	".hs":    "haskell",
	".tf":    "terraform",
}

// InferLanguage returns the language name for a filename based on its extension.
// Returns empty string if unknown.
func InferLanguage(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if lang, ok := extensionToLanguage[ext]; ok {
		return lang
	}
	return ""
}
