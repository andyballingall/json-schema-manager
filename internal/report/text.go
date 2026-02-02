package report

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/andyballingall/json-schema-manager/internal/schema"
)

// TextReporter implements schema.Reporter for plain text output.
type TextReporter struct {
	Verbose   bool
	UseColour bool
}

const (
	colReset     = "\033[0m"
	colRed       = "\033[31m"
	colGreen     = "\033[32m"
	colGrey      = "\033[90m"
	colWhite     = "\033[37m"
	colBoldRed   = "\033[1;31m"
	colBoldGreen = "\033[1;32m"
	colBoldWhite = "\033[1;37m"
)

// cs returns a string which will render with the given colour
// if colourisation is enabled.
func (tr *TextReporter) cs(c, s string) string {
	if !tr.UseColour {
		return s
	}
	return c + s + colReset
}

func (tr *TextReporter) Write(w io.Writer, r *schema.TestReport) error {
	divider := strings.Repeat("-", 40)

	fmt.Fprintf(w, "%s\n", divider)
	fmt.Fprint(w, tr.cs(colBoldWhite, "JSM TEST REPORT\n\n"))
	fmt.Fprintf(w, "%s %s\n", tr.cs(colGrey, "Started: "), tr.cs(colWhite, r.StartTime.Format("15:04:05")))
	fmt.Fprintf(w, "%s %s\n", tr.cs(colGrey, "Duration:"), tr.cs(colWhite, r.EndTime.Sub(r.StartTime).String()))
	fmt.Fprintf(w, "%s\n", divider)

	// Collect all keys to ensure sorted output
	keysMap := make(map[schema.Key]bool)
	for k := range r.PassedTests {
		keysMap[k] = true
	}
	for k := range r.FailedTests {
		keysMap[k] = true
	}

	keys := make([]schema.Key, 0, len(keysMap))
	for k := range keysMap {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	totalPassed := 0
	totalFailed := 0

	for _, k := range keys {
		keyCol := colWhite
		passed := r.PassedTests[k]
		failed := r.FailedTests[k]
		totalPassed += len(passed)
		totalFailed += len(failed)

		statusText := "PASS"
		statusCol := colGreen
		if len(failed) > 0 {
			statusText = "FAIL"
			statusCol = colRed
			keyCol = colRed
		}

		status := tr.cs(statusCol, "["+statusText+"]")
		keyStr := tr.cs(keyCol, string(k)+schema.SchemaSuffix)
		suffix := fmt.Sprintf("(pass: %d, fail: %d)", len(passed), len(failed))

		fmt.Fprintf(w, "%s %s %s\n", status, keyStr, tr.cs(statusCol, suffix))

		if tr.Verbose {
			// Show for each schema, the tests that passed and failed, along with any errors
			for _, spec := range passed {
				fmt.Fprintf(w, "  %s %s (%s)\n",
					tr.cs(colGreen, "✓"),
					tr.cs(colGrey, spec.TestInfo.Path),
					tr.cs(colGreen, spec.ResultLabel()))
			}

			for _, spec := range failed {
				fmt.Fprintf(w, "  %s %s (%s):\n",
					tr.cs(colRed, "✗"),
					tr.cs(colGrey, spec.TestInfo.Path),
					tr.cs(colRed, spec.ResultLabel()))
				fmt.Fprintf(w, "    %v\n", spec.Err)
			}
		} else {
			// Just show the failures
			for _, spec := range failed {
				fmt.Fprintf(w, "  %s %s (%s):\n",
					tr.cs(colRed, "✗"),
					tr.cs(colGrey, spec.TestInfo.Path),
					tr.cs(colRed, spec.ResultLabel()))
				fmt.Fprintf(w, "    %v\n", spec.Err)
			}
		}
	}

	fmt.Fprintf(w, "%s\n", divider)
	summaryLabel := tr.cs(colBoldWhite, "Test summary: ")
	summaryStats := fmt.Sprintf("%d passed, %d failed", totalPassed, totalFailed)
	statsColor := colBoldGreen
	if totalFailed > 0 {
		statsColor = colBoldRed
	}
	fmt.Fprintf(w, "%s%s\n", summaryLabel, tr.cs(statsColor, summaryStats))
	fmt.Fprintf(w, "%s\n", divider)

	return nil
}
