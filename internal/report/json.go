// Package report provides reporting functionality for JSM.
package report

import (
	"encoding/json"
	"io"
	"time"

	"github.com/bitshepherds/json-schema-manager/internal/schema"
)

// JSONReporter implements schema.Reporter for JSON output.
type JSONReporter struct{}

type jsonSpec struct {
	Path        string `json:"path"`
	TestDocType string `json:"type"`
	Error       string `json:"error,omitempty"`
}

type jsonSchemaResults struct {
	Passed []jsonSpec `json:"passed"`
	Failed []jsonSpec `json:"failed"`
}

type jsonOutput struct {
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Duration  string `json:"duration"`
	Stats     struct {
		TotalPassed int `json:"totalPassed"`
		TotalFailed int `json:"totalFailed"`
	} `json:"stats"`
	Results map[schema.Key]jsonSchemaResults `json:"results"`
}

func (jr *JSONReporter) Write(w io.Writer, r *schema.TestReport) error {
	out := jsonOutput{
		StartTime: r.StartTime.Format(time.RFC3339),
		EndTime:   r.EndTime.Format(time.RFC3339),
		Duration:  r.EndTime.Sub(r.StartTime).String(),
		Results:   make(map[schema.Key]jsonSchemaResults),
	}

	for k, specs := range r.PassedTests {
		res := out.Results[k]
		for _, s := range specs {
			res.Passed = append(res.Passed, jsonSpec{
				Path:        s.TestInfo.Path,
				TestDocType: string(s.TestDocType),
			})
		}
		out.Results[k] = res
		out.Stats.TotalPassed += len(specs)
	}

	for k, specs := range r.FailedTests {
		res := out.Results[k]
		for _, s := range specs {
			errMsg := ""
			if s.Err != nil {
				errMsg = s.Err.Error()
			}
			res.Failed = append(res.Failed, jsonSpec{
				Path:        s.TestInfo.Path,
				TestDocType: string(s.TestDocType),
				Error:       errMsg,
			})
		}
		out.Results[k] = res
		out.Stats.TotalFailed += len(specs)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
