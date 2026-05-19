package configapply

import (
	"fmt"
	"strings"
)

func countLines(s string) int {
	if s == "" {
		return 0
	}
	n := strings.Count(s, "\n")
	if !strings.HasSuffix(s, "\n") {
		n++
	}
	return n
}

func sharedPrefix(a, b string) int {
	la := strings.Split(a, "\n")
	lb := strings.Split(b, "\n")
	n := 0
	for n < len(la) && n < len(lb) && la[n] == lb[n] {
		n++
	}
	return n
}

func DiffSummary(before, after string) string {
	if before == after {
		return "no change"
	}
	bl := strings.Split(before, "\n")
	al := strings.Split(after, "\n")
	add, del := lineDiffCounts(bl, al)
	return fmt.Sprintf("+%d -%d", add, del)
}

func lineDiffCounts(before, after []string) (int, int) {
	bm := map[string]int{}
	for _, l := range before {
		bm[l]++
	}
	am := map[string]int{}
	for _, l := range after {
		am[l]++
	}
	var add, del int
	for line, n := range am {
		if bm[line] < n {
			add += n - bm[line]
		}
	}
	for line, n := range bm {
		if am[line] < n {
			del += n - am[line]
		}
	}
	return add, del
}

// UnifiedDiff renders a hunked patch using a Myers-style LCS.
// MVP: 3-line context, no hunk header line numbers (frontend can show side-by-side
// from Before/After if needed). Output uses unified diff sigils so Monaco's diff
// editor can also accept the raw text directly.
func UnifiedDiff(before, after string, contextLines int) string {
	if before == after {
		return ""
	}
	if contextLines < 0 {
		contextLines = 3
	}
	bl := strings.Split(before, "\n")
	al := strings.Split(after, "\n")
	ops := myersDiff(bl, al)

	var sb strings.Builder
	for i, op := range ops {
		switch op.kind {
		case opEqual:
			if shouldShowContext(ops, i, contextLines) {
				sb.WriteString(" ")
				sb.WriteString(op.line)
				sb.WriteString("\n")
			}
		case opDel:
			sb.WriteString("-")
			sb.WriteString(op.line)
			sb.WriteString("\n")
		case opAdd:
			sb.WriteString("+")
			sb.WriteString(op.line)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func shouldShowContext(ops []diffOp, idx, n int) bool {
	for i := idx - n; i <= idx+n; i++ {
		if i < 0 || i >= len(ops) {
			continue
		}
		if ops[i].kind != opEqual {
			return true
		}
	}
	return false
}

type opKind int

const (
	opEqual opKind = iota
	opDel
	opAdd
)

type diffOp struct {
	kind opKind
	line string
}

// myersDiff returns a simple O(n*m) LCS-based diff. Fine for config files; if we
// ever feed 10k-line content here, swap for a real Myers implementation.
func myersDiff(a, b []string) []diffOp {
	n, m := len(a), len(b)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	var ops []diffOp
	i, j := 0, 0
	for i < n && j < m {
		if a[i] == b[j] {
			ops = append(ops, diffOp{opEqual, a[i]})
			i++
			j++
		} else if dp[i+1][j] >= dp[i][j+1] {
			ops = append(ops, diffOp{opDel, a[i]})
			i++
		} else {
			ops = append(ops, diffOp{opAdd, b[j]})
			j++
		}
	}
	for i < n {
		ops = append(ops, diffOp{opDel, a[i]})
		i++
	}
	for j < m {
		ops = append(ops, diffOp{opAdd, b[j]})
		j++
	}
	return ops
}
