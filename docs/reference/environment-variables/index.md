# Environment variables reference

## `GRAFANA_ORG_ID`

OrgID specifies the organization targeted by this config.
Note: required when targeting an on-prem Grafana instance.
See StackID for Grafana Cloud instances.

## `GRAFANA_PASSWORD`

Password to use when using with basic authentication.
Optional.

## `GRAFANA_SERVER`

Server is the address of the Grafana server (https://hostname:port/path).
Required.

## `GRAFANA_STACK_ID`

StackID specifies the Grafana Cloud stack targeted by this config.
Note: required when targeting a Grafana Cloud instance.
See OrgID for on-prem Grafana instances.

## `GRAFANA_TOKEN`

APIToken is a service account token.
See https://grafana.com/docs/grafana/latest/administration/service-accounts/#add-a-token-to-a-service-account-in-grafana
Note: if defined, the API Token takes precedence over basic auth credentials.
Optional.

## `GRAFANA_USER`

User to authenticate as with basic authentication.
Optional.
