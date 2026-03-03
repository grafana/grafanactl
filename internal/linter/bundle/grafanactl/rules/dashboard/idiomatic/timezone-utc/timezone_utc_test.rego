package grafanactl.rules.dashboard.idiomatic["timezone-utc_test"]

import data.grafanactl.utils

import data.grafanactl.rules.dashboard.idiomatic["timezone-utc"] as rule

test_dashboard_v1_with_timezone_utc_is_accepted if {
	resource := {
	    "kind": "Dashboard",
	    "apiVersion": "dashboard.grafana.app/v1beta1",
	    "metadata": {"name": "test-dashboard"},
	    "spec": {"timezone": "utc"}
	}

	report := rule.report with input as resource

	report == set()
}

test_dashboard_v1_with_timezone_browser_is_rejected if {
	resource := {
	    "kind": "Dashboard",
	    "apiVersion": "dashboard.grafana.app/v1beta1",
	    "metadata": {"name": "test-dashboard"},
	    "spec": {"timezone": "browser"}
	}

	r := rule.report with input as resource

	r == {{
	    "category": "idiomatic",
	    "description": "Timezone should be utc. For reasons.",
	    "details": "timezone is 'browser', expected 'utc'",
	    "related_resources": [],
	    "resource_type": "dashboard",
	    "rule": "timezone-utc",
	    "severity": "error",
	}}
}

test_dashboard_v2_with_timezone_utc_is_accepted if {
	resource := {
	    "kind": "Dashboard",
	    "apiVersion": "dashboard.grafana.app/v2beta1",
	    "metadata": {"name": "test-dashboard"},
	    "spec": {
	        "timeSettings": {
	            "timezone": "utc"
	        }
	    }
	}

	report := rule.report with input as resource

	report == set()
}

test_dashboard_v2_with_timezone_browser_is_rejected if {
	resource := {
	    "kind": "Dashboard",
	    "apiVersion": "dashboard.grafana.app/v2beta1",
	    "metadata": {"name": "test-dashboard"},
	    "spec": {
	        "timeSettings": {
	            "timezone": "browser"
	        }
	    }
	}

	r := rule.report with input as resource

	r == {{
	    "category": "idiomatic",
	    "description": "Timezone should be utc. For reasons.",
	    "details": "timezone is 'browser', expected 'utc'",
	    "related_resources": [],
	    "resource_type": "dashboard",
	    "rule": "timezone-utc",
	    "severity": "error",
	}}
}
