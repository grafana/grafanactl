# Releasing

Releases are handled by `goreleaser`, configured in the [`.goreleaser.yaml`](../.goreleaser.yaml)
file and running in the [`release.yaml`](../.github/workflows/release.yaml) GitHub action.

Trigger the release pipeline by creating and pushing a tag: `git tag v{version} && git push origin v{version}`

The documentation will be built and published as well.

If you wish to build and publish the documentation without releasing `grafafanactl`,
manually start the [`publish-docs.yaml`](../.github/workflows/publish-docs.yaml) workflow.
