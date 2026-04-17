// Package prompts defines waggle prompt templates for each review category.
package prompts

import "github.com/lucientong/waggle/pkg/prompt"

// SecurityTemplate is the prompt template for security review.
var SecurityTemplate = prompt.New(`You are a security-focused code reviewer.
Analyze the following code change for security vulnerabilities.

Language: {{language}}
File: {{filename}}

Patch:
{{patch}}

Focus on: SQL injection, XSS, path traversal, secrets exposure, unsafe deserialization,
SSRF, insecure cryptography, authentication bypass, race conditions, command injection.

Return a JSON array of findings. Each finding must have these fields:
- "file": the filename
- "category": "security"
- "severity": one of "critical", "warning", "info"
- "line": the line number in the patch (if identifiable, else 0)
- "message": description of the vulnerability
- "suggestion": how to fix it

Return [] if no issues found.`)

// StyleTemplate is the prompt template for style review.
var StyleTemplate = prompt.New(`You are a code style reviewer.
Analyze the following code change for style and readability issues.

Language: {{language}}
File: {{filename}}

Patch:
{{patch}}

Focus on: naming conventions, formatting consistency, idiomatic patterns for the language,
dead code, overly complex expressions, missing documentation on public APIs.

Return a JSON array of findings. Each finding must have these fields:
- "file": the filename
- "category": "style"
- "severity": one of "critical", "warning", "info"
- "line": the line number in the patch (if identifiable, else 0)
- "message": description of the style issue
- "suggestion": the improvement

Return [] if no issues found.`)

// LogicTemplate is the prompt template for logic review.
var LogicTemplate = prompt.New(`You are a code logic reviewer.
Analyze the following code change for logic errors and potential bugs.

Language: {{language}}
File: {{filename}}

Patch:
{{patch}}

Focus on: nil/null dereferences, off-by-one errors, incorrect error handling,
unreachable code, wrong boolean conditions, missing edge cases, resource leaks,
incorrect type conversions, concurrency issues.

Return a JSON array of findings. Each finding must have these fields:
- "file": the filename
- "category": "logic"
- "severity": one of "critical", "warning", "info"
- "line": the line number in the patch (if identifiable, else 0)
- "message": description of the logic error
- "suggestion": how to fix it

Return [] if no issues found.`)

// PerformanceTemplate is the prompt template for performance review.
var PerformanceTemplate = prompt.New(`You are a performance-focused code reviewer.
Analyze the following code change for performance issues.

Language: {{language}}
File: {{filename}}

Patch:
{{patch}}

Focus on: N+1 queries, unnecessary memory allocations, missing caching opportunities,
O(n^2) or worse algorithms where better exists, unnecessary network calls,
blocking operations on hot paths, inefficient string concatenation.

Return a JSON array of findings. Each finding must have these fields:
- "file": the filename
- "category": "performance"
- "severity": one of "critical", "warning", "info"
- "line": the line number in the patch (if identifiable, else 0)
- "message": description of the performance issue
- "suggestion": the optimization

Return [] if no issues found.`)

// SummaryTemplate is the prompt template for generating a review summary.
var SummaryTemplate = prompt.New(`You are a code review lead writing a summary for a pull request review.

Given the following review findings (as JSON), write a concise, constructive,
human-readable markdown summary suitable for posting as a GitHub PR comment.

Number of issues: {{issue_count}}
Critical count: {{critical_count}}
Warning count: {{warning_count}}
Info count: {{info_count}}

Issues JSON:
{{issues_json}}

Write a summary that:
1. Opens with the overall assessment (good, needs work, has critical issues)
2. Highlights the most important findings
3. Groups related issues together
4. Keeps a constructive, helpful tone
5. Is concise (under 500 words)

Return only the markdown summary text, no JSON wrapping.`)
