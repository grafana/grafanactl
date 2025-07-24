---
title: Dashboards as code
---

With this workflow, you can define and manage dashboards as code, saving them to a version control system like Git. This is useful for teams that want to maintain a history of changes, collaborate on dashboard design, and ensure consistency across environments.

1. Use a dashboard generation script (for example, with the [Foundation SDK](https://github.com/grafana/grafana-foundation-sdk)). You can find an example implementation in the Grafana as code [hands-on lab repository](https://github.com/grafana/dashboards-as-code-workshop/tree/main/part-one-golang-starter).
1. Serve and preview the output of the dashboard generator locally:
   ```shell
   grafanactl config use-context YOUR_CONTEXT  # for example "dev"
   grafanactl resources serve --script 'go run scripts/generate-dashboard.go' --watch './scripts'
   ```
1. When the output looks correct, generate dashboard manifest files:
   ```shell
   go run scripts/generate-dashboard.go --generate-resource-manifests --output './resources'
   ```
1. Push the generated resources to your Grafana instance:
   ```shell
   grafanactl config use-context YOUR_CONTEXT  # for example "dev"
   grafanactl resources push -d ./resources/
   ```
