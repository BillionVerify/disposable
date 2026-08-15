# Release Checklist

Use this checklist before publishing a new tag.

1. Confirm `data/domains.txt`, `data/wildcards.txt`, and `data/exceptions.txt`
   contain normalized, sorted, unique ASCII entries.
2. Run:

   ```bash
   go vet ./...
   go test ./... -count=1 -race
   go build ./...
   cargo fmt --check
   cargo clippy --all-targets -- -D warnings
   cargo test --all-targets
   cargo test --doc
   cargo package --locked
   ```

3. Review `THIRD_PARTY_NOTICES.md` if any upstream source or dependency changed.
4. For a Go release, update the version tag:

   ```bash
   git tag v0.YYYY.MMDD
   git push origin v0.YYYY.MMDD
   ```

5. Confirm the tag resolves through the Go module proxy before announcing it.
6. For a Rust release:
   - update `package.version` in `Cargo.toml`;
   - run `cargo publish --dry-run --locked`;
   - tag the same commit as `rust-v<version>` and push the tag;
   - run `cargo publish --locked` with the BillionVerify crates.io owner credentials;
   - confirm the package and generated documentation are available before
     changing the README to recommend `cargo add disposable`.
