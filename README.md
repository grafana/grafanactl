# Grafana CLI

> [!WARNING]
> **`grafanactl` is being deprecated.** We're bringing all our learnings and experience into the new, improved CLI tool [gcx](https://github.com/grafana/gcx).
>
> To migrate from `grafanactl` to `gcx`, search-and-replace `grafanactl` with `gcx`. For `grafanactl resources serve`, use `gcx dev serve` instead.

Grafana CLI (_grafanactl_) is a command-line tool designed to simplify interaction with Grafana instances.

It enables users to authenticate, manage multiple environments, and perform administrative tasks through Grafana's REST API — all from the terminal.

Whether you're automating workflows in CI/CD pipelines or switching between staging and production environments, Grafana CLI provides a flexible and scriptable way to manage your Grafana setup efficiently.

## Documentation

See [the documentation](https://grafana.github.io/grafanactl/) to learn how to
install, configure and use the Grafana CLI.

## Maturity

> [!WARNING]
> **This repository is currently *in public preview*, which means that it is still under active development.**
> Bugs and issues are handled solely by Engineering teams. On-call support or SLAs are not available.

This project should be considered as "public preview". While it is used by Grafana Labs, it is still under active development.

Additional information can be found in [Release life cycle for Grafana Labs](https://grafana.com/docs/release-life-cycle/).

## Contributing

See our [contributing guide](CONTRIBUTING.md).
