package grafanactl.utils

resource_is_alert(resource) if {
	resource.name != ""
	resource.rules != ""
}
