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

func TestAlertAnnotationParsing(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		wantAnnotations int
		validate        func(t *testing.T, annotations []alertAnnotation)
	}{
		{
			name:            "empty annotations",
			input:           `[]`,
			wantAnnotations: 0,
		},
		{
			name: "single annotation",
			input: `[
				{
					"id": 100,
					"alertId": 42,
					"alertName": "CPU Alert",
					"newState": "alerting",
					"prevState": "normal",
					"time": 1700000000000,
					"timeEnd": 1700003600000,
					"dashboardId": 5,
					"dashboardUID": "dash-1",
					"panelId": 3,
					"text": "CPU usage exceeded threshold"
				}
			]`,
			wantAnnotations: 1,
			validate: func(t *testing.T, annotations []alertAnnotation) {
				t.Helper()

				a := annotations[0]
				assert.Equal(t, int64(100), a.ID)
				assert.Equal(t, int64(42), a.AlertID)
				assert.Equal(t, "CPU Alert", a.AlertName)
				assert.Equal(t, "alerting", a.NewState)
				assert.Equal(t, "normal", a.PrevState)
				assert.Equal(t, int64(1700000000000), a.Time)
				assert.Equal(t, int64(1700003600000), a.TimeEnd)
				assert.Equal(t, int64(5), a.DashboardID)
				assert.Equal(t, "dash-1", a.DashboardUID)
				assert.Equal(t, int64(3), a.PanelID)
				assert.Equal(t, "CPU usage exceeded threshold", a.Text)
			},
		},
		{
			name: "multiple annotations with state transitions",
			input: `[
				{
					"id": 1,
					"alertId": 10,
					"alertName": "Alert A",
					"newState": "alerting",
					"prevState": "normal",
					"time": 1700000000000,
					"timeEnd": 1700001000000
				},
				{
					"id": 2,
					"alertId": 10,
					"alertName": "Alert A",
					"newState": "normal",
					"prevState": "alerting",
					"time": 1700001000000,
					"timeEnd": 0
				},
				{
					"id": 3,
					"alertId": 20,
					"alertName": "Alert B",
					"newState": "alerting",
					"prevState": "pending",
					"time": 1700002000000,
					"timeEnd": 1700005000000
				}
			]`,
			wantAnnotations: 3,
			validate: func(t *testing.T, annotations []alertAnnotation) {
				t.Helper()

				assert.Equal(t, "alerting", annotations[0].NewState)
				assert.Equal(t, "normal", annotations[0].PrevState)

				assert.Equal(t, "normal", annotations[1].NewState)
				assert.Equal(t, "alerting", annotations[1].PrevState)
				assert.Equal(t, int64(0), annotations[1].TimeEnd)

				assert.Equal(t, int64(20), annotations[2].AlertID)
				assert.Equal(t, "Alert B", annotations[2].AlertName)
				assert.Equal(t, "pending", annotations[2].PrevState)
			},
		},
		{
			name: "annotation with zero/missing optional fields",
			input: `[
				{
					"id": 50,
					"alertId": 0,
					"alertName": "",
					"newState": "alerting",
					"prevState": "normal",
					"time": 1700000000000,
					"timeEnd": 0,
					"dashboardId": 0,
					"dashboardUID": "",
					"panelId": 0,
					"text": ""
				}
			]`,
			wantAnnotations: 1,
			validate: func(t *testing.T, annotations []alertAnnotation) {
				t.Helper()

				a := annotations[0]
				assert.Equal(t, int64(50), a.ID)
				assert.Zero(t, a.AlertID)
				assert.Empty(t, a.AlertName)
				assert.Zero(t, a.DashboardID)
				assert.Empty(t, a.DashboardUID)
				assert.Zero(t, a.PanelID)
				assert.Empty(t, a.Text)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var annotations []alertAnnotation
			err := json.Unmarshal([]byte(tt.input), &annotations)
			require.NoError(t, err)

			assert.Len(t, annotations, tt.wantAnnotations)

			if tt.validate != nil {
				tt.validate(t, annotations)
			}
		})
	}
}
