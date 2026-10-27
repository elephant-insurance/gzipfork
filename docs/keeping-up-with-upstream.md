# Keeping gzipfork up to date with gin-contrib/gzip

`gzipfork` is a fork of [gin-contrib/gzip](https://github.com/gin-contrib/gzip) (MIT, so it is
fine under the AGPL ban). It exists to turn on **compressed requests** and to count them. This
document says how to stay current with upstream, and why the better answer may be to stop
maintaining the fork at all.

## State of the fork (checked 2026-10-27)

| Fact | Value |
|---|---|
| Fork vs upstream | 11 commits ahead, 119 behind (`git log --oneline master ^upstream/master`) |
| Last common commit | 3dac13b, June 2022 |
| Fork's real change | about 60 lines in `handler.go` and `options.go`; every other diff is the `gzip` to `gzipfork` package rename, go.mod, or the deleted README |
| Who imports it | Only `go-microservice-arch/v3/gzip/handler.go`, as `gzipfork/v3`. The 37 go.mod references in the local org checkouts all point at `gzipfork/v3`, which services pick up through the arch. |
| Root module | No local go.mod requires `github.com/elephant-insurance/gzipfork` (the non-`/v3` path). Everything uses `v3/`. Confirm no repo outside these checkouts uses it before deleting. |

The fork's whole contribution:

1. `DecompressFn` returns `(before, after int)` so the middleware can count bytes.
2. Five package-level counters (`CompressedRequests`, `CompressedBytes`, `DecompressedBytes`,
   `UncompressedRequests`, `UncompressedBytes`), which the arch prints in `gzip.Diagnostics()`.
3. A case-insensitive `shouldDecompress` that accepts any `Content-Encoding` ending in `gzip`.

The actual decompression (base64 decode, then gunzip) is **not** in the fork. It lives in the
arch's `unzip` function and is passed in through `WithDecompressFn`.

## The fork is no longer needed for the feature

Upstream has since added request decompression: `WithDecompressFn(func(c *gin.Context))` and
`WithDecompressOnly()` (upstream `options.go`, PR #32 "gzip decompress supports"). It ships no
default decompressor, which is why the feature looked disabled. The arch already supplies its own,
so only the counters and the function signature differ.

## Option A (recommended): retire the fork

Use upstream `github.com/gin-contrib/gzip` directly from the arch and move the counters there.
Nothing is left to merge, and updates become an ordinary version bump.

Change `go-microservice-arch/v3/gzip/handler.go` only:

```go
import (
	"sync/atomic"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

var compressedRequests, uncompressedRequests, compressedBytes, decompressedBytes, uncompressedBytes atomic.Int64

// unzip keeps its current body, minus the named returns. Before returning it does:
//   compressedRequests.Add(1); compressedBytes.Add(int64(before)); decompressedBytes.Add(int64(after))

func Gzip(requiredConfig cfg.Configurator) gin.HandlerFunc {
	// ...existing enableCompressedRequests logging...
	gz := gzip.Gzip(gzip.DefaultCompression, gzip.WithDecompressFn(unzip))
	return func(c *gin.Context) {
		if c.Request.Header.Get("Content-Encoding") == "" && c.Request.ContentLength > 0 {
			uncompressedRequests.Add(1)
			uncompressedBytes.Add(c.Request.ContentLength)
		}
		gz(c)
	}
}
```

`Diagnostics()` then reads these counters instead of `ogz.*`. Then remove the `gzipfork/v3`
requirement from the arch's go.mod. Once the arch ships, archive this repo.

**This sketch has not been compiled or run.** Before shipping, check these behaviour differences,
because upstream's *response* compression has changed since 2022:

- It does not compress responses with status 400 or above.
- It rewrites strong ETags to weak ones (`W/...`).
- It no longer skips `text/event-stream` requests. Use `WithCustomShouldCompressFn` if any service
  streams SSE.
- It matches `Content-Encoding` with a case-sensitive `Contains("gzip")`. The fork lowercases it
  first. If a client can send `GZIP`, have `unzip` or a pre-middleware normalise the header.
- It supports `WithMinLength`, so small responses can be left uncompressed.

Per the ele-go rules, this is an architecture change. Open it as a PR against
`go-microservice-arch` (or write it up under its `docs/feature-requests/`), not as a local patch
in a service.

## Option B: keep the fork and sync it with git

Use this only if Option A is blocked. All of it is plain git. GitHub's "Sync fork" button is the
wipe-out you saw, so skip it.

### Why merge, not rebase

Consumers resolve the module by tag or pseudo-version. Rebasing rewrites commits that are already
published, so use `git merge upstream/master`. It keeps your history and only conflicts where
both sides changed the same lines.

### Reduce the conflict surface first (one-time)

Every sync conflicts on every file you touched. Shrink that list:

1. **Keep `package gzip`.** Renaming to `package gzipfork` touches every `.go` file, and each of
   them conflicts on every merge. The module path already separates the fork, and the arch
   imports it under the alias `ogz` anyway.
2. **Move fork-only code into its own files** (`stats.go`, `decompress.go`). Leave `handler.go`
   with just the few hook lines. Then upstream refactors of `handler.go` conflict on only those lines.
3. **Delete the unused root module.** Keep only `v3/` (see "State of the fork").
4. Turn on conflict memory and the clearer conflict style:
   `git config rerere.enabled true && git config merge.conflictstyle zdiff3`

### First sync: re-apply the delta onto upstream

The fork is 119 commits behind and upstream rewrote `handler.go`/`options.go` (an `Options` struct
became an `Option` interface plus a private `config`). A line-by-line merge would conflict in most
files, so re-apply the ~60-line delta onto current upstream and record the old history as merged,
without force-pushing:

```sh
git fetch upstream --tags
git verify-commit HEAD                    # must be signed before you commit anything
git worktree add ../gzipfork-sync -b sync/upstream-$(date +%F) upstream/master
cd ../gzipfork-sync
# re-apply the delta by hand: fork-only files, plus the `before, after` DecompressFn contract
git merge -s ours --no-commit master     # keeps master's history, takes OUR (upstream+delta) tree
git commit -S -m "sync with gin-contrib/gzip $(git describe --tags upstream/master)"
```

Merge `sync/...` into `master` as a fast-forward or PR, keep `v3/` as its own module, and tag a
new `v3.x.y`. The arch then bumps the pin.

### Every sync after that (a few minutes)

```sh
git fetch upstream --tags
git log --oneline master..upstream/master        # review what is coming
git worktree add ../gzipfork-sync -b sync/upstream-$(date +%F) master
cd ../gzipfork-sync
git merge --no-ff --no-commit upstream/master
# resolve conflicts, then:
go build ./... && go vet ./... && go test ./...  # in the root and in v3/
git commit -S                                    # signed merge commit
```

Then refresh dependencies with the ele-go `refresh-deps.sh` (not a `go get -u` loop over the
`go.mod` files, which hangs on private modules), push the branch, open a PR, merge, and tag.

Use upstream *tags* where you can (`git merge v1.2.8`) rather than tip-of-master, so you only take
released changes.

### Automating the nag

A scheduled workflow (weekly) that runs `git fetch upstream && git rev-list --count
master..upstream/master` and opens an issue when the count is above zero keeps this from piling
up again. The 119-commit gap is exactly what that would have prevented.

## Decision

Do **Option A**. It removes the fork, the sync process, the arch to fork to arch dependency cycle
(the reason `v3/clicker.go` is a copied clicker), and the maintenance burden you are unhappy about.
Fall back to Option B only if the response-compression differences above cannot be accepted.
