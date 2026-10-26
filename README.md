# attn's sync engine

This repository is attn's embedded file synchronization engine. It started at
Syncthing v2.1.6, commit `4461a1ce354d3990660966ad70176e00246adb61`.
The `main` and `attn` branches contain the engine; attn pins `v0.1.0` with a plain
`require github.com/victorarias/syncthing v0.1.0`.

The embedder owns authentication and transport. The engine exchanges the file
sync protocol over a `net.Conn` the embedder supplies. It opens no listener,
dials no peers, and performs no TLS or certificate exchange.

## Embed the engine

Start with `attnsync.Configuration(localID)`. Add each authenticated peer to
`cfg.Devices`, and add send-receive folders to `cfg.Folders`. Copy the device and
folder defaults from `cfg.Defaults` when adding entries.

```go
engine, err := attnsync.New(indexDirectory, cfg, localID, logger)
if err != nil {
    return err
}
stopped := make(chan error, 1)
go func() { stopped <- engine.Serve(ctx) }()
<-engine.Ready()

subscription := engine.Subscribe(events.AllEvents)
defer subscription.Unsubscribe()
// Drain subscription.C() for this engine's lifetime.

// The bridge has already authenticated peerID.
err = engine.ServeConn(bridgeContext, conn, peerID)
```

`New` takes an absolute index directory and a configuration value. `Serve` owns
the engine until cancellation and closes its database before returning. Run it
once per engine. `Ready` signals model initialization. Watch index and
folder-state events for first sync completion. Each engine routes logs to its
supplied `*slog.Logger`. Nil means discard. Importing the module leaves the process's default logger and
HTTP transport alone.

Create the folder marker yourself, once when initializing a folder. The default
marker is `.attn/notebook-sync`. The engine checks its presence and does not
recreate it. Folder paths must be absolute, and markers must be relative paths under their folders. Syncthing's default
`.stfolder` marker is refused.

Set `FolderConfiguration.IgnorePatterns` in memory. The engine never reads or
writes `.stignore`. Patterns support Syncthing's ignore syntax except `#include`,
which is rejected before the configuration is applied. The defaults are
`/.attn`, `.*.tmp.*` (atomic-write temporary files), `.seed-transfer-*`, and
`(?d).DS_Store`.

`Configure` replaces the in-memory configuration and waits for it to apply.
Use it to add or remove peers and folders, or change ignore patterns. The
embedder stores configuration; the engine writes no config.xml.

`FolderState` returns sync state, outstanding files and bytes, and the folder
error. `Completion` reports a peer's progress. `Subscribe` exposes the engine's
events, including state, index updates, file changes and disconnects.
`IsConflict` identifies `*.sync-conflict-*` file names from those events. The
embedder owns conflict counts and notifications.

## Build and verify

```sh
go test ./...
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build ./lib/attnsync
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./lib/attnsync
```

Native cgo builds use SQLite through mattn/go-sqlite3. Builds without cgo use
modernc.org/sqlite. Both keep SQLite temporary storage in memory. Index files
stay in the supplied directory; pulled-file temporary files stay in the folder.
No asset generation or build tag is required.

The two-process driver is in `script/attn/probe`. Give it a copied Notebook
snapshot and a fresh run directory. The Linux run uses Landlock to confine
writes to the supplied state and folder paths:

```sh
go build -o /tmp/attnsync-probe ./script/attn/probe
python3 script/attn/probe/verify.py --fixture /path/to/notebook-copy \
  --root /path/to/new-run --probe /tmp/attnsync-probe --sandbox
```

It checks hashes, edits, reconnects, conflict preservation, disconnect latency
and file leftovers, then stops its child processes. Its `events.json` diagnostic
contains names from the snapshot; keep that file private.

## Maintain the fork

attn maintains this engine independently. The API evolves independently, and the branches do not rebase onto upstream. To take an upstream security fix, inspect its diff against
the base commit above, port the affected engine code, and run the retained tests
and the two-process sync probe above before tagging the next version.

Removed code includes the applications and GUI, transport management, discovery,
relays, NAT traversal, TLS utilities, upgrades, usage-reporting services, the
legacy LevelDB database, packaging, and upstream system integration tests.
`script/attn/trim.py` records the dependency-based trimming recipe. Protocol,
model, scanner, filesystem, SQLite, configuration and ignore-matching tests stay
with the engine.

The patched glob matcher is kept in `internal/glob`, based on
`github.com/calmh/glob` commit `1d823af5017b`. Keeping that patch here lets an
embedder use a plain module requirement without copying upstream's `replace`
directives. Its original MIT terms remain in `internal/glob/LICENSE`.

Syncthing files and attn's changes retain MPL-2.0 headers. See [LICENSE](LICENSE).
