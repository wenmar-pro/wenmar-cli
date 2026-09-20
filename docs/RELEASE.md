# Releasing

1. Ensure CI is green on main (fmt/vet/lint/vulncheck/test/surface-diff/regen-drift).
2. Bump nothing by hand — goreleaser derives the version from the tag.
3. Tag: `git tag v0.x.0 && git push origin v0.x.0`.
4. The Release workflow runs: test job → goreleaser (builds, SBOMs,
   cosign-signs checksums, attests, drafts the GitHub release).
5. Review the draft release, then publish.
6. The install-cli script picks up the new version immediately (it resolves
   latest from the releases page).

## Distribution

Releases are published to GitHub Releases only — no Homebrew tap, no Scoop
bucket, and no third-party package repositories. Users install via:

- the `install-cli` script (macOS / Linux / WSL2),
- a direct binary download from GitHub Releases,
- `go install github.com/wenmar-pro/wenmar-cli/cmd/wenmar@latest`,
- or a deb/rpm/apk release asset.

Because everything is hosted on GitHub Releases, the release workflow needs
no extra secrets beyond the default `GITHUB_TOKEN` (plus cosign attestation
permissions).
