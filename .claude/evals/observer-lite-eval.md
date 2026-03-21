# Observer-Lite Eval Suite

Regression tests for `observer-lite` agent. Run these before and after modifying observer logic to ensure detection accuracy.

Inspired by [Anthropic's skill testing methodology](https://claude.com/blog/improving-skill-creator-test-measure-and-refine-agent-skills): define test inputs with expected outcomes, measure pass rate.

---

## Test Case 1: Hardcoded Secret (expect [BLOCKER])

**Input:** Create a temporary Go file with a hardcoded password:

```go
// /tmp/eval-test/bad_secret.go
package main

const dbPassword = "SuperSecret123!"

func connect() {
    db.Open("postgres://admin:" + dbPassword + "@localhost:5432/mydb")
}
```

**Expected finding:**
```
[CRITICAL] — bad_secret.go:3 — Hardcoded password in const dbPassword — Use environment variable via os.Getenv()
```

**Pass criteria:** Observer-lite reports at least one [CRITICAL] or [BLOCKER] finding referencing the hardcoded password. Zero false negatives.

---

## Test Case 2: Missing Test for Exported Function (expect [WARNING])

**Input:** Create a Go file with an exported function and no corresponding test file:

```go
// /tmp/eval-test/untested.go
package main

// CalculateChecksum computes a CRC32 checksum for the given data.
func CalculateChecksum(data []byte) uint32 {
    var crc uint32 = 0xFFFFFFFF
    for _, b := range data {
        crc ^= uint32(b)
    }
    return crc
}
```

No `untested_test.go` exists.

**Expected finding:**
```
[WARNING] — untested.go:4 — Exported function CalculateChecksum has no test — Add test in untested_test.go
```

**Pass criteria:** Observer-lite reports at least one [WARNING] about the missing test. Zero false negatives.

---

## Test Case 3: Clean File (expect 0 findings)

**Input:** A clean Go file with no issues:

```go
// /tmp/eval-test/clean.go
package main

import (
    "fmt"
    "os"
)

func greet() {
    name := os.Getenv("USER_NAME")
    if name == "" {
        name = "World"
    }
    fmt.Printf("Hello, %s!\n", name)
}
```

With corresponding test file:

```go
// /tmp/eval-test/clean_test.go
package main

import "testing"

func TestGreet(t *testing.T) {
    // Just verify it doesn't panic
    greet()
}
```

**Expected:** Zero findings (no CRITICAL, WARNING, or INFO).

**Pass criteria:** Observer-lite produces a clean report with 0 findings. Zero false positives.

---

## How to Run

1. Create the test files in `/tmp/eval-test/`
2. Point observer-lite at `/tmp/eval-test/` instead of the project root
3. Check QUALITY.md output against expected findings
4. Score: 3/3 = pass, <3/3 = observer needs tuning

## Metrics to Track

| Run Date | Test 1 (Secret) | Test 2 (No Test) | Test 3 (Clean) | Score | Observer Version |
|----------|-----------------|-------------------|----------------|-------|-----------------|
| _not yet run_ | — | — | — | —/3 | — |

## When to Run

- After any modification to `observer-lite.md`
- After Claude model updates (regression check)
- Monthly (drift detection on the drift detector)
