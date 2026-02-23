package alerts

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRulesResponseParsing(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantGroups int
		wantTotals map[string]int64
		validate   func(t *testing.T, resp *rulesResponse)
	}{
		{
			name: "empty groups",
			input: `{
				"data": {
					"groups": []
				}
			}`,
			wantGroups: 0,
		},
		{
			name: "single group with one rule",
			input: `{
				"data": {
					"groups": [
						{
							"name": "test-group",
							"folderUid": "folder-1",
							"rules": [
								{
									"state": "firing",
									"name": "High CPU",
									"health": "ok",
									"lastEvaluation": "2025-01-01T00:00:00Z",
									"alerts": [],
									"uid": "rule-1",
									"folderUid": "folder-1",
									"labels": {"severity": "critical"},
									"type": "alerting"
								}
							],
							"interval": 60,
							"lastEvaluation": "2025-01-01T00:00:00Z"
						}
					],
					"totals": {"firing": 1, "inactive": 5}
				}
			}`,
			wantGroups: 1,
			wantTotals: map[string]int64{"firing": 1, "inactive": 5},
			validate: func(t *testing.T, resp *rulesResponse) {
				t.Helper()

				group := resp.Data.RuleGroups[0]
				assert.Equal(t, "test-group", group.Name)
				assert.Equal(t, "folder-1", group.FolderUID)
				assert.Equal(t, float64(60), group.Interval)
				assert.Len(t, group.Rules, 1)

				rule := group.Rules[0]
				assert.Equal(t, "firing", rule.State)
				assert.Equal(t, "High CPU", rule.Name)
				assert.Equal(t, "ok", rule.Health)
				assert.Equal(t, "rule-1", rule.UID)
				assert.Equal(t, "folder-1", rule.FolderUID)
				assert.Equal(t, "critical", rule.Labels["severity"])
				assert.Equal(t, "alerting", rule.Type)
			},
		},
		{
			name: "multiple groups with alerts",
			input: `{
				"data": {
					"groups": [
						{
							"name": "group-a",
							"folderUid": "f1",
							"rules": [
								{
									"state": "inactive",
									"name": "Rule A",
									"health": "ok",
									"lastEvaluation": "2025-01-01T00:00:00Z",
									"alerts": [
										{
											"labels": {"instance": "server-1"},
											"annotations": {"summary": "test alert"},
											"state": "firing",
											"activeAt": "2025-01-01T00:00:00Z",
											"value": "100"
										}
									],
									"uid": "r-a",
									"folderUid": "f1",
									"type": "alerting"
								}
							],
							"interval": 30,
							"lastEvaluation": "2025-01-01T00:00:00Z"
						},
						{
							"name": "group-b",
							"folderUid": "f2",
							"rules": [],
							"interval": 120,
							"lastEvaluation": "2025-01-01T00:00:00Z"
						}
					]
				}
			}`,
			wantGroups: 2,
			validate: func(t *testing.T, resp *rulesResponse) {
				t.Helper()

				rule := resp.Data.RuleGroups[0].Rules[0]
				require.Len(t, rule.Alerts, 1)

				alert := rule.Alerts[0]
				assert.Equal(t, "firing", alert.State)
				assert.Equal(t, "server-1", alert.Labels["instance"])
				assert.Equal(t, "test alert", alert.Annotations["summary"])
				assert.Equal(t, "100", alert.Value)
				assert.NotNil(t, alert.ActiveAt)

				assert.Equal(t, "group-b", resp.Data.RuleGroups[1].Name)
				assert.Empty(t, resp.Data.RuleGroups[1].Rules)
			},
		},
		{
			name: "rule with totals",
			input: `{
				"data": {
					"groups": [
						{
							"name": "g",
							"folderUid": "f",
							"rules": [
								{
									"state": "firing",
									"name": "R",
									"health": "ok",
									"lastEvaluation": "2025-01-01T00:00:00Z",
									"uid": "u",
									"folderUid": "f",
									"type": "alerting",
									"totals": {"firing": 3, "pending": 1}
								}
							],
							"interval": 60,
							"lastEvaluation": "2025-01-01T00:00:00Z"
						}
					]
				}
			}`,
			wantGroups: 1,
			validate: func(t *testing.T, resp *rulesResponse) {
				t.Helper()

				rule := resp.Data.RuleGroups[0].Rules[0]
				assert.Equal(t, int64(3), rule.Totals["firing"])
				assert.Equal(t, int64(1), rule.Totals["pending"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp rulesResponse
			err := json.Unmarshal([]byte(tt.input), &resp)
			require.NoError(t, err)

			assert.Len(t, resp.Data.RuleGroups, tt.wantGroups)

			if tt.wantTotals != nil {
				assert.Equal(t, tt.wantTotals, resp.Data.Totals)
			}

			if tt.validate != nil {
				tt.validate(t, &resp)
			}
		})
	}
}

func TestAlertsResponseParsing(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantAlerts int
		validate   func(t *testing.T, alerts []alertInstance)
	}{
		{
			name: "empty alerts",
			input: `{
				"data": {
					"alerts": []
				}
			}`,
			wantAlerts: 0,
		},
		{
			name: "single firing alert",
			input: `{
				"data": {
					"alerts": [
						{
							"labels": {"alertname": "HighCPU", "severity": "critical"},
							"annotations": {"summary": "CPU usage is high"},
							"state": "firing",
							"activeAt": "2025-06-15T10:30:00Z",
							"value": "95.5"
						}
					]
				}
			}`,
			wantAlerts: 1,
			validate: func(t *testing.T, alerts []alertInstance) {
				t.Helper()

				alert := alerts[0]
				assert.Equal(t, "firing", alert.State)
				assert.Equal(t, "HighCPU", alert.Labels["alertname"])
				assert.Equal(t, "critical", alert.Labels["severity"])
				assert.Equal(t, "CPU usage is high", alert.Annotations["summary"])
				assert.Equal(t, "95.5", alert.Value)
				assert.NotNil(t, alert.ActiveAt)

				expected := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
				assert.Equal(t, expected, *alert.ActiveAt)
			},
		},
		{
			name: "multiple alerts with nil activeAt",
			input: `{
				"data": {
					"alerts": [
						{
							"labels": {"alertname": "A"},
							"annotations": {},
							"state": "firing",
							"activeAt": "2025-01-01T00:00:00Z",
							"value": "1"
						},
						{
							"labels": {"alertname": "B"},
							"annotations": {"desc": "test"},
							"state": "pending",
							"value": "2"
						}
					]
				}
			}`,
			wantAlerts: 2,
			validate: func(t *testing.T, alerts []alertInstance) {
				t.Helper()

				assert.Equal(t, "firing", alerts[0].State)
				assert.NotNil(t, alerts[0].ActiveAt)

				assert.Equal(t, "pending", alerts[1].State)
				assert.Nil(t, alerts[1].ActiveAt)
				assert.Equal(t, "test", alerts[1].Annotations["desc"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp alertsResponse
			err := json.Unmarshal([]byte(tt.input), &resp)
			require.NoError(t, err)

			assert.Len(t, resp.Data.Alerts, tt.wantAlerts)

			if tt.validate != nil {
				tt.validate(t, resp.Data.Alerts)
			}
		})
	}
}

func TestParseStateHistoryFrame(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantEntries int
		wantErr     bool
		validate    func(t *testing.T, entries []stateHistoryEntry)
	}{
		{
			name: "realistic data frame with multiple entries",
			input: `{
				"schema": {
					"fields": [
						{"name": "time", "type": "time", "typeInfo": {"frame": "time.Time"}},
						{"name": "line", "type": "other", "typeInfo": {"frame": "json.RawMessage"}},
						{"name": "labels", "type": "other", "typeInfo": {"frame": "json.RawMessage"}}
					]
				},
				"data": {
					"values": [
						[1700003000000, 1700002000000, 1700001000000],
						[
							{"schemaVersion":1,"previous":"Normal","current":"Alerting","error":"","values":{"A":42.5},"condition":"A","dashboardUID":"dash-1","panelID":1,"fingerprint":"abc123","ruleTitle":"High CPU","ruleID":100,"ruleUID":"rule-1","labels":{"severity":"critical"}},
							{"schemaVersion":1,"previous":"Alerting","current":"Normal","ruleTitle":"High CPU","ruleID":100,"ruleUID":"rule-1","labels":{"severity":"critical"}},
							{"schemaVersion":1,"previous":"Normal","current":"Alerting","values":{"B":99.9},"condition":"B","ruleTitle":"Disk Full","ruleID":200,"ruleUID":"rule-2","labels":{"severity":"warning"}}
						],
						[{}, {}, {}]
					]
				}
			}`,
			wantEntries: 3,
			validate: func(t *testing.T, entries []stateHistoryEntry) {
				t.Helper()

				// Sorted by timestamp descending
				assert.Equal(t, "High CPU", entries[0].RuleTitle)
				assert.Equal(t, "Normal", entries[0].Previous)
				assert.Equal(t, "Alerting", entries[0].Current)
				assert.Equal(t, "dash-1", entries[0].DashboardUID)
				assert.Equal(t, int64(1), entries[0].PanelID)
				assert.Equal(t, "abc123", entries[0].Fingerprint)
				assert.Equal(t, "rule-1", entries[0].RuleUID)
				assert.Equal(t, int64(100), entries[0].RuleID)
				assert.Equal(t, "A", entries[0].Condition)
				assert.InDelta(t, 42.5, entries[0].Values["A"], 0.001)
				assert.Equal(t, "critical", entries[0].Labels["severity"])
				assert.Equal(t, 1, entries[0].SchemaVersion)
				assert.Equal(t, time.UnixMilli(1700003000000), entries[0].Timestamp)

				assert.Equal(t, "High CPU", entries[1].RuleTitle)
				assert.Equal(t, "Alerting", entries[1].Previous)
				assert.Equal(t, "Normal", entries[1].Current)
				assert.Equal(t, time.UnixMilli(1700002000000), entries[1].Timestamp)

				assert.Equal(t, "Disk Full", entries[2].RuleTitle)
				assert.Equal(t, "Normal", entries[2].Previous)
				assert.Equal(t, "Alerting", entries[2].Current)
				assert.Equal(t, "rule-2", entries[2].RuleUID)
				assert.Equal(t, "warning", entries[2].Labels["severity"])
				assert.Equal(t, time.UnixMilli(1700001000000), entries[2].Timestamp)
			},
		},
		{
			name: "empty frame with no fields",
			input: `{
				"schema": {
					"fields": []
				},
				"data": {
					"values": []
				}
			}`,
			wantEntries: 0,
		},
		{
			name: "frame with fields but empty values",
			input: `{
				"schema": {
					"fields": [
						{"name": "time", "type": "time"},
						{"name": "line", "type": "other"},
						{"name": "labels", "type": "other"}
					]
				},
				"data": {
					"values": [[], [], []]
				}
			}`,
			wantEntries: 0,
		},
		{
			name: "frame missing line field",
			input: `{
				"schema": {
					"fields": [
						{"name": "time", "type": "time"},
						{"name": "labels", "type": "other"}
					]
				},
				"data": {
					"values": [
						[1700000000000],
						[{}]
					]
				}
			}`,
			wantEntries: 0,
		},
		{
			name: "malformed line entry skipped",
			input: `{
				"schema": {
					"fields": [
						{"name": "time", "type": "time"},
						{"name": "line", "type": "other"},
						{"name": "labels", "type": "other"}
					]
				},
				"data": {
					"values": [
						[1700002000000, 1700001000000],
						[
							"not a json object",
							{"schemaVersion":1,"previous":"Normal","current":"Alerting","ruleTitle":"Valid Rule","ruleUID":"rule-ok"}
						],
						[{}, {}]
					]
				}
			}`,
			wantEntries: 1,
			validate: func(t *testing.T, entries []stateHistoryEntry) {
				t.Helper()

				assert.Equal(t, "Valid Rule", entries[0].RuleTitle)
				assert.Equal(t, "rule-ok", entries[0].RuleUID)
				assert.Equal(t, time.UnixMilli(1700001000000), entries[0].Timestamp)
			},
		},
		{
			name: "entry with error field and empty optional fields",
			input: `{
				"schema": {
					"fields": [
						{"name": "time", "type": "time"},
						{"name": "line", "type": "other"},
						{"name": "labels", "type": "other"}
					]
				},
				"data": {
					"values": [
						[1700000000000],
						[
							{"schemaVersion":1,"previous":"Normal","current":"Error","error":"datasource timeout","ruleTitle":"Broken Rule","ruleUID":"rule-err"}
						],
						[{}]
					]
				}
			}`,
			wantEntries: 1,
			validate: func(t *testing.T, entries []stateHistoryEntry) {
				t.Helper()

				assert.Equal(t, "Error", entries[0].Current)
				assert.Equal(t, "datasource timeout", entries[0].Error)
				assert.Equal(t, "Broken Rule", entries[0].RuleTitle)
				assert.Empty(t, entries[0].DashboardUID)
				assert.Zero(t, entries[0].PanelID)
				assert.Empty(t, entries[0].Fingerprint)
				assert.Nil(t, entries[0].Labels)
				assert.Nil(t, entries[0].Values)
			},
		},
		{
			name:    "invalid JSON input",
			input:   `not json at all`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, err := parseStateHistoryFrame([]byte(tt.input))
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, entries, tt.wantEntries)

			if tt.validate != nil {
				tt.validate(t, entries)
			}
		})
	}
}
