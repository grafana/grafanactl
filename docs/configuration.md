---
title: Configuration
---

# Configuration

Grafana CLI can be configured in two ways: using environment variables or through a configuration file.

Environment variables can only describe a single context, and are best suited to CI environments.

Configuration files can store multiple contexts, providing a convenient way to switch between Grafana instances.

## Using environment variables

Grafana CLI interacts with Grafana via its REST API. Therefore, you need to establish authentication credentials.

The minimum requirement is to set the URL of the Grafana instance and the organization ID to use:

```shell
GRAFANA_SERVER='http://localhost:3000' GRAFANA_ORG_ID='1' grafanactl config check
```

Optionally, set the following values depending on your authentication method with the given Grafana instance:

* A [token](./reference/environment-variables/index.md#grafana_token) if using a [Grafana service account](https://grafana.com/docs/grafana/latest/administration/service-accounts/) (recommended)
* A [username](./reference/environment-variables/index.md#grafana_user) and [password](./reference/environment-variables/index.md#grafana_password) if using basic authentication

Next, consider [creating a context](#defining-contexts) to persist this configuration.

Once you have configured your authentication method, you are ready to use the Grafana CLI.

!!! note

    * Every supported environment variable is listed in our [reference documentation](./reference/environment-variables/index.md).
    * Check the [config file reference documentation](./reference/configuration/index.md) for details on all available config options.

## Defining contexts

Grafana CLI supports multiple contexts, thereby allowing easy switching between instances. By default, Grafana CLI uses the `default` context.

Configure the `default` context:

```shell
grafanactl config set contexts.default.grafana.server http://localhost:3000

# Set org-id when using OSS/Enterprise - skip when targeting Grafana Cloud
grafanactl config set contexts.default.grafana.org-id 1

# Authenticate with a service account token
grafanactl config set contexts.default.grafana.token service-account-token

# Or alternatively, use basic authentication
grafanactl config set contexts.default.grafana.user admin
grafanactl config set contexts.default.grafana.password admin
```

New contexts can be created in a similar way:

```shell
grafanactl config set contexts.staging.grafana.server https://staging.grafana.example
grafanactl config set contexts.staging.grafana.org-id 1
```

!!! note

    In both cases, `default` and `staging` refer to the name of the context being manipulated.

## Configuration file

Grafana CLI stores its configuration in a YAML file. Its location is determined as follows:

1. If the `--config` flag is set, then that file will be loaded. No other location will be considered.
2. If the `$XDG_CONFIG_HOME` environment variable is set, then it will be used: `$XDG_CONFIG_HOME/grafanactl/config.yaml`
3. If the `$HOME environment` variable is set, then it will be used: `$HOME/.config/grafanactl/config.yaml`
4. If the `$XDG_CONFIG_DIRS` environment variable is set, then it will be used: `$XDG_CONFIG_DIRS/grafanactl/config.yaml`

!!! tip

    The `grafanactl config check` command will display the configuration file currently in use.

## Useful commands

Check the configuration:

```shell
grafanactl config check
```

List existing contexts:

```shell
grafanactl config list-contexts
```

Switch to a different context:

```shell
grafanactl config use-context staging
```

See the entire configuration:

```shell
grafanactl config view
```
