package cmd

import (
	"os"
	"regexp"
	"sort"
	"strconv"
	"testing"

	"github.com/wenmar-pro/wenmar-cli/internal/errors"
)

// codeSet extracts the leading exit-code numbers from a table rendered as
// "N  meaning" lines (help topic) or "| N | meaning |" rows (markdown).
var (
	topicCodeRe    = regexp.MustCompile(`(?m)^\s{2}(\d{1,2})\s`)
	markdownCodeRe = regexp.MustCompile(`(?m)^\|\s*(\d{1,2})\s*\|`)
)

func codeSet(re *regexp.Regexp, text string) []int {
	seen := map[int]bool{}
	for _, m := range re.FindAllStringSubmatch(text, -1) {
		n, _ := strconv.Atoi(m[1])
		if n >= 0 && n <= 10 {
			seen[n] = true
		}
	}
	out := make([]int, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Ints(out)
	return out
}

func TestExitCodeTableSync(t *testing.T) {
	want := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	// 1. The help topic lists all 11 codes.
	var topic string
	for _, ht := range helpTopics {
		if ht.name == "exit-codes" {
			topic = ht.content
		}
	}
	if topic == "" {
		t.Fatal("exit-codes topic missing from helpTopics")
	}
	if got := codeSet(topicCodeRe, topic); !equalInts(got, want) {
		t.Errorf("help exit-codes topic lists %v, want %v", got, want)
	}

	// 2. The internal/errors constants match the contract values.
	consts := map[string]int{
		"success": errors.ExitSuccess, "generic": errors.ExitGeneric,
		"auth": errors.ExitAuth, "notfound": errors.ExitNotFound,
		"validation": errors.ExitValidation, "ratelimit": errors.ExitRateLimit,
		"server": errors.ExitServer, "conflict": errors.ExitConflict,
		"forbidden": errors.ExitForbidden, "partial": errors.ExitPartial,
		"offline": errors.ExitOffline,
	}
	for name, v := range map[string]int{
		"success": 0, "generic": 1, "auth": 2, "notfound": 3, "validation": 4,
		"ratelimit": 5, "server": 6, "conflict": 7, "forbidden": 8,
		"partial": 9, "offline": 10,
	} {
		if consts[name] != v {
			t.Errorf("errors.Exit%s = %d, want %d", name, consts[name], v)
		}
	}

	// 3. README documents the same set.
	readme, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	if got := codeSet(markdownCodeRe, string(readme)); !equalInts(got, want) {
		t.Errorf("README exit-code table lists %v, want %v", got, want)
	}

	// 4. SKILL.md documents the same set.
	skill, err := os.ReadFile("../skills/wenmar/SKILL.md")
	if err != nil {
		t.Fatalf("read SKILL.md: %v", err)
	}
	if got := codeSet(markdownCodeRe, string(skill)); !equalInts(got, want) {
		t.Errorf("SKILL.md exit-code table lists %v, want %v", got, want)
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
