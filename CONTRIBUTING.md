# Contributing Guidelines

This document is a guide to help you through the process of contributing to `grafanactl`.

## Development environment

`grafanactl` relies on [`devbox`](https://www.jetify.com/devbox/docs/) to manage all
the tools required to work on it.

A shell including all these tools is accessible via:

```console
$ devbox shell
```

This shell can be exited like any other shell, with `exit` or `CTRL+D`.

One-off commands can be executed within the devbox shell as well:

```console
$ devbox run go version
```

Packages can be installed using:

```console
$ devbox add go@1.24
```

Available packages can be found on the [NixOS package repository](https://search.nixos.org/packages).
