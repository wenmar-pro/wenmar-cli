# Releasing

1. Ensure CI is green on main (fmt/vet/lint/vulncheck/test/surface-diff/regen-drift).
2. Bump nothing by hand — goreleaser derives the version from the tag.
3. Tag: `git tag v0.x.0 && git push origin v0.x.0`.
4. The Release workflow runs: test job → goreleaser (builds, SBOMs,
   cosign-signs checksums, attests, drafts the release, updates the
   Homebrew tap and Scoop bucket).
5. Review the draft release, then publish.
6. The install-cli script picks up the new version immediately (it resolves
   latest from the releases page).

## Prerequisites (one-time)

The Homebrew tap and Scoop bucket publish steps need PATs configured as
secrets on the wenmar-cli repo BEFORE the first tagged release after this
lands — otherwise that release's publish step fails:

1. Create `wenmar-pro/homebrew-tap` and `wenmar-pro/scoop-bucket` repos (empty).
2. Create a PAT with `repo` scope for the tap commits; add it as the
   `HOMEBREW_TAP_TOKEN` and `SCOOP_BUCKET_TOKEN` secrets on the wenmar-cli repo.
