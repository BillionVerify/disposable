# Release Checklist

Use this checklist before publishing a new tag.

1. Confirm `data/domains.txt`, `data/wildcards.txt`, and `data/exceptions.txt`
   contain normalized, sorted, unique ASCII entries.
2. Run:

   ```bash
   go vet ./...
   go test ./... -count=1 -race
   go build ./...
   ```

3. Review `THIRD_PARTY_NOTICES.md` if any upstream source or dependency changed.
4. Update the version tag:

   ```bash
   git tag v0.YYYY.MMDD
   git push origin v0.YYYY.MMDD
   ```

5. Confirm the tag resolves through the Go module proxy before announcing it.
