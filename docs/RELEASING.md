# Releasing a new version

## Version bump checklist

1. Update `VERSION` file with the new version number
2. Update `prose.1` man page header (the `.TH` line contains the version string)
3. Build to verify ldflags injection: `make build` — check the output says the right version
4. Run tests: `make test`
5. Commit: `git commit -am "v<VERSION>"`
6. Tag: `git tag v<VERSION>`
7. Push: `git push && git push --tags`

## Where version lives

| Location | Format | Updated how |
|----------|--------|-------------|
| `VERSION` | `1.18.1` | Manually edit |
| `prose.1` line 1 | `"prose 1.18.1"` | Manually edit |
| Binary | embedded | Automatic via `make build` (ldflags reads VERSION) |
| `go install` | `@v1.18.1` | Automatic from git tag |

## Notes

- The `Makefile` reads `VERSION` and passes it to `-ldflags "-X main.Version=$(VERSION)"` at build time. No need to touch the Makefile for a version bump.
- `go install github.com/JackWReid/prose/cmd/prose@latest` picks up the latest tagged release.
- Patch versions (x.y.Z) are for fixes and non-feature changes. Minor versions (x.Y.0) are for new features.
