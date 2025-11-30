package service

import (
	"html/template"
	"io"
	"strings"
	"time"

	"github.com/ludo-technologies/jscan/domain"
	"github.com/ludo-technologies/jscan/internal/version"
)

// HTMLData represents the data for HTML template
type HTMLData struct {
	GeneratedAt    string
	Duration       int64
	Version        string
	Complexity     *domain.ComplexityResponse
	DeadCode       *domain.DeadCodeResponse
	Summary        *domain.AnalyzeSummary
	HasComplexity  bool
	HasDeadCode    bool
}

// WriteHTML writes the analysis result as HTML
func (f *OutputFormatterImpl) WriteHTML(
	complexityResponse *domain.ComplexityResponse,
	deadCodeResponse *domain.DeadCodeResponse,
	writer io.Writer,
	duration time.Duration,
) error {
	now := time.Now()

	// Build summary
	summary := &domain.AnalyzeSummary{}

	if complexityResponse != nil {
		summary.ComplexityEnabled = true
		summary.TotalFunctions = complexityResponse.Summary.TotalFunctions
		summary.AverageComplexity = complexityResponse.Summary.AverageComplexity
		summary.HighComplexityCount = complexityResponse.Summary.HighRiskFunctions
		summary.AnalyzedFiles = complexityResponse.Summary.FilesAnalyzed
	}

	if deadCodeResponse != nil {
		summary.DeadCodeEnabled = true
		summary.DeadCodeCount = deadCodeResponse.Summary.TotalFindings
		summary.CriticalDeadCode = deadCodeResponse.Summary.CriticalFindings
		summary.WarningDeadCode = deadCodeResponse.Summary.WarningFindings
		summary.InfoDeadCode = deadCodeResponse.Summary.InfoFindings
		if deadCodeResponse.Summary.TotalFiles > summary.TotalFiles {
			summary.TotalFiles = deadCodeResponse.Summary.TotalFiles
		}
	}

	// Calculate health score
	_ = summary.CalculateHealthScore()

	data := HTMLData{
		GeneratedAt:   now.Format("2006-01-02 15:04:05"),
		Duration:      duration.Milliseconds(),
		Version:       version.Version,
		Complexity:    complexityResponse,
		DeadCode:      deadCodeResponse,
		Summary:       summary,
		HasComplexity: complexityResponse != nil,
		HasDeadCode:   deadCodeResponse != nil,
	}

	funcMap := template.FuncMap{
		"join": func(elems []string, sep string) string {
			return strings.Join(elems, sep)
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"scoreQuality": func(score int) string {
			switch {
			case score >= domain.ScoreThresholdExcellent:
				return "excellent"
			case score >= domain.ScoreThresholdGood:
				return "good"
			case score >= domain.ScoreThresholdFair:
				return "fair"
			default:
				return "poor"
			}
		},
		"gradeClass": func(grade string) string {
			switch grade {
			case "A":
				return "grade-a"
			case "B":
				return "grade-b"
			case "C":
				return "grade-c"
			case "D":
				return "grade-d"
			default:
				return "grade-f"
			}
		},
	}

	tmpl := template.Must(template.New("analyze").Funcs(funcMap).Parse(htmlTemplate))
	return tmpl.Execute(writer, data)
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>jscan Analysis Report</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }
        .header {
            background: white;
            border-radius: 10px;
            padding: 30px;
            margin-bottom: 20px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.1);
        }
        .header h1 {
            color: #667eea;
            margin-bottom: 10px;
        }
        .header .subtitle {
            color: #666;
            font-size: 14px;
        }
        .score-badge {
            display: inline-block;
            padding: 10px 20px;
            border-radius: 50px;
            font-size: 24px;
            font-weight: bold;
            margin: 10px 0;
        }
        .grade-a { background: #4caf50; color: white; }
        .grade-b { background: #8bc34a; color: white; }
        .grade-c { background: #ff9800; color: white; }
        .grade-d { background: #ff5722; color: white; }
        .grade-f { background: #f44336; color: white; }

        .tabs {
            background: white;
            border-radius: 10px;
            overflow: hidden;
            box-shadow: 0 10px 30px rgba(0,0,0,0.1);
        }
        .tab-buttons {
            display: flex;
            background: #f5f5f5;
        }
        .tab-button {
            flex: 1;
            padding: 15px;
            border: none;
            background: transparent;
            cursor: pointer;
            font-size: 16px;
            transition: all 0.3s;
        }
        .tab-button.active {
            background: white;
            color: #667eea;
            font-weight: bold;
        }
        .tab-content {
            display: none;
            padding: 30px;
        }
        .tab-content.active {
            display: block;
        }

        .metric-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin: 20px 0;
        }
        .metric-card {
            background: #f8f9fa;
            padding: 20px;
            border-radius: 8px;
            text-align: center;
        }
        .metric-value {
            font-size: 32px;
            font-weight: bold;
            color: #667eea;
        }
        .metric-label {
            color: #666;
            margin-top: 5px;
        }

        .table {
            width: 100%;
            border-collapse: collapse;
            margin: 20px 0;
        }
        .table th, .table td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #ddd;
        }
        .table th {
            background: #f8f9fa;
            font-weight: 600;
        }

        .risk-low { color: #4caf50; }
        .risk-medium { color: #ff9800; }
        .risk-high { color: #f44336; }

        .severity-critical { color: #f44336; }
        .severity-warning { color: #ff9800; }
        .severity-info { color: #2196f3; }

        .score-bars {
            margin: 20px 0;
        }
        .score-bar-item {
            margin-bottom: 24px;
        }
        .score-bar-header {
            display: flex;
            justify-content: space-between;
            margin-bottom: 6px;
            font-size: 14px;
        }
        .score-label {
            font-weight: 600;
            color: #333;
        }
        .score-value {
            font-weight: 700;
            color: #667eea;
        }
        .score-bar-container {
            width: 100%;
            height: 12px;
            background: #e0e0e0;
            border-radius: 6px;
            overflow: hidden;
        }
        .score-bar-fill {
            height: 100%;
            transition: width 0.3s ease;
            border-radius: 6px;
        }
        .score-excellent { background: linear-gradient(90deg, #4caf50, #66bb6a); }
        .score-good { background: linear-gradient(90deg, #8bc34a, #9ccc65); }
        .score-fair { background: linear-gradient(90deg, #ff9800, #ffa726); }
        .score-poor { background: linear-gradient(90deg, #f44336, #ef5350); }
        .score-detail {
            margin-top: 4px;
            font-size: 12px;
            color: #666;
        }

        .tab-header-with-score {
            display: flex;
            align-items: center;
            justify-content: space-between;
            margin-bottom: 20px;
            padding-bottom: 12px;
            border-bottom: 2px solid #e0e0e0;
        }

        .score-badge-compact {
            display: inline-block;
            padding: 6px 14px;
            border-radius: 16px;
            font-size: 13px;
            font-weight: 700;
            color: white;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>jscan Analysis Report</h1>
            <p class="subtitle">Generated: {{.GeneratedAt}} | Duration: {{.Duration}}ms | Version: {{.Version}}</p>
            <div class="score-badge {{gradeClass .Summary.Grade}}">
                Health Score: {{.Summary.HealthScore}}/100 (Grade: {{.Summary.Grade}})
            </div>
        </div>

        <div class="tabs">
            <div class="tab-buttons">
                <button class="tab-button active" onclick="showTab('summary', this)">Summary</button>
                {{if .HasComplexity}}
                <button class="tab-button" onclick="showTab('complexity', this)">Complexity</button>
                {{end}}
                {{if .HasDeadCode}}
                <button class="tab-button" onclick="showTab('deadcode', this)">Dead Code</button>
                {{end}}
            </div>

            <div id="summary" class="tab-content active">
                <h2>Analysis Summary</h2>

                <h3 style="margin-top: 20px; margin-bottom: 16px; color: #2c3e50;">Quality Scores</h3>
                <div class="score-bars">
                    {{if .HasComplexity}}
                    <div class="score-bar-item">
                        <div class="score-bar-header">
                            <span class="score-label">Complexity</span>
                            <span class="score-value">{{.Summary.ComplexityScore}}/100</span>
                        </div>
                        <div class="score-bar-container">
                            <div class="score-bar-fill score-{{scoreQuality .Summary.ComplexityScore}}" style="width: {{.Summary.ComplexityScore}}%"></div>
                        </div>
                        <div class="score-detail">Avg: {{printf "%.1f" .Summary.AverageComplexity}}, High-risk: {{.Summary.HighComplexityCount}}</div>
                    </div>
                    {{end}}

                    {{if .HasDeadCode}}
                    <div class="score-bar-item">
                        <div class="score-bar-header">
                            <span class="score-label">Dead Code</span>
                            <span class="score-value">{{.Summary.DeadCodeScore}}/100</span>
                        </div>
                        <div class="score-bar-container">
                            <div class="score-bar-fill score-{{scoreQuality .Summary.DeadCodeScore}}" style="width: {{.Summary.DeadCodeScore}}%"></div>
                        </div>
                        <div class="score-detail">{{.Summary.DeadCodeCount}} issues, {{.Summary.CriticalDeadCode}} critical</div>
                    </div>
                    {{end}}
                </div>

                <h3 style="margin-top: 24px; margin-bottom: 16px; color: #2c3e50;">File Statistics</h3>
                <div class="metric-grid">
                    <div class="metric-card">
                        <div class="metric-value">{{.Summary.AnalyzedFiles}}</div>
                        <div class="metric-label">Files Analyzed</div>
                    </div>
                    {{if .HasComplexity}}
                    <div class="metric-card">
                        <div class="metric-value">{{.Summary.TotalFunctions}}</div>
                        <div class="metric-label">Total Functions</div>
                    </div>
                    <div class="metric-card">
                        <div class="metric-value">{{printf "%.2f" .Summary.AverageComplexity}}</div>
                        <div class="metric-label">Avg Complexity</div>
                    </div>
                    {{end}}
                    {{if .HasDeadCode}}
                    <div class="metric-card">
                        <div class="metric-value">{{.Summary.DeadCodeCount}}</div>
                        <div class="metric-label">Dead Code Issues</div>
                    </div>
                    {{end}}
                </div>
            </div>

            {{if .HasComplexity}}
            <div id="complexity" class="tab-content">
                <div class="tab-header-with-score">
                    <h2 style="margin: 0;">Complexity Analysis</h2>
                    <div class="score-badge-compact score-{{scoreQuality .Summary.ComplexityScore}}">
                        {{.Summary.ComplexityScore}}/100
                    </div>
                </div>

                <div class="metric-grid">
                    <div class="metric-card">
                        <div class="metric-value">{{.Complexity.Summary.TotalFunctions}}</div>
                        <div class="metric-label">Total Functions</div>
                    </div>
                    <div class="metric-card">
                        <div class="metric-value">{{printf "%.2f" .Complexity.Summary.AverageComplexity}}</div>
                        <div class="metric-label">Average</div>
                    </div>
                    <div class="metric-card">
                        <div class="metric-value">{{.Complexity.Summary.MaxComplexity}}</div>
                        <div class="metric-label">Maximum</div>
                    </div>
                </div>

                <h3>Functions</h3>
                <table class="table">
                    <thead>
                        <tr>
                            <th>Function</th>
                            <th>File</th>
                            <th>Complexity</th>
                            <th>Risk</th>
                        </tr>
                    </thead>
                    <tbody>
                        {{range $i, $f := .Complexity.Functions}}
                        {{if lt $i 20}}
                        <tr>
                            <td>{{$f.Name}}</td>
                            <td>{{$f.FilePath}}</td>
                            <td>{{$f.Metrics.Complexity}}</td>
                            <td class="risk-{{$f.RiskLevel}}">{{$f.RiskLevel}}</td>
                        </tr>
                        {{end}}
                        {{end}}
                    </tbody>
                </table>
                {{if gt (len .Complexity.Functions) 20}}
                <p style="color: #666; margin-top: 10px;">Showing top 20 of {{len .Complexity.Functions}} functions</p>
                {{end}}
            </div>
            {{end}}

            {{if .HasDeadCode}}
            <div id="deadcode" class="tab-content">
                <div class="tab-header-with-score">
                    <h2 style="margin: 0;">Dead Code Detection</h2>
                    <div class="score-badge-compact score-{{scoreQuality .Summary.DeadCodeScore}}">
                        {{.Summary.DeadCodeScore}}/100
                    </div>
                </div>

                <div class="metric-grid">
                    <div class="metric-card">
                        <div class="metric-value">{{.DeadCode.Summary.TotalFindings}}</div>
                        <div class="metric-label">Total Issues</div>
                    </div>
                    <div class="metric-card">
                        <div class="metric-value">{{.DeadCode.Summary.CriticalFindings}}</div>
                        <div class="metric-label">Critical</div>
                    </div>
                    <div class="metric-card">
                        <div class="metric-value">{{.DeadCode.Summary.WarningFindings}}</div>
                        <div class="metric-label">Warnings</div>
                    </div>
                </div>

                {{if gt .DeadCode.Summary.TotalFindings 0}}
                <h3>Dead Code Issues</h3>
                <table class="table">
                    <thead>
                        <tr>
                            <th>File</th>
                            <th>Function</th>
                            <th>Lines</th>
                            <th>Severity</th>
                            <th>Reason</th>
                        </tr>
                    </thead>
                    <tbody>
                        {{range $file := .DeadCode.Files}}
                        {{range $func := $file.Functions}}
                        {{range $i, $finding := $func.Findings}}
                        {{if lt $i 20}}
                        <tr>
                            <td>{{$finding.Location.FilePath}}</td>
                            <td>{{$finding.FunctionName}}</td>
                            <td>{{$finding.Location.StartLine}}-{{$finding.Location.EndLine}}</td>
                            <td class="severity-{{$finding.Severity}}">{{$finding.Severity}}</td>
                            <td>{{$finding.Reason}}</td>
                        </tr>
                        {{end}}
                        {{end}}
                        {{end}}
                        {{end}}
                    </tbody>
                </table>
                {{else}}
                <p style="color: #4caf50; font-weight: bold; margin-top: 20px;">✓ No dead code detected</p>
                {{end}}
            </div>
            {{end}}
        </div>
    </div>

    <script>
        function showTab(tabName, el) {
            const tabs = document.querySelectorAll('.tab-content');
            tabs.forEach(tab => tab.classList.remove('active'));

            const buttons = document.querySelectorAll('.tab-button');
            buttons.forEach(btn => btn.classList.remove('active'));

            document.getElementById(tabName).classList.add('active');
            if (el) { el.classList.add('active'); }
        }
    </script>
</body>
</html>`
