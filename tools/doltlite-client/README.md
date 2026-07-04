# DoltLite Client

`doltlite-client` is the direct diagnostic client for Gas City DoltLite bead
stores. It opens `.beads/metadata.json`, selects the active
`.beads/doltlite/<database>.db`, and runs SQL through the libdoltlite-linked
SQLite driver.

The DoltLite storage and SQL contract lives in the local
[DoltLite README](../../../doltlite/README.md). Read that before changing
client behavior or pack maintenance commands.

## Commands

```bash
doltlite-client -city /path/to/city info
doltlite-client -city /path/to/city query "SELECT COUNT(*) FROM issues"
doltlite-client -city /path/to/city exec "UPDATE issues SET metadata = metadata WHERE id = ?" gp-example.1
doltlite-client -city /path/to/city show gp-example.1
doltlite-client -city /path/to/city set-metadata gp-example.1 key=value
doltlite-client -city /path/to/city close gp-example.1 "reason"
```

`query` and `exec` are SQL pass-through commands. Use them as the oracle when
checking whether DoltLite-backed reads and writes work without a Dolt server.

## Maintenance Probes

Use DoltLite SQL from the README for native maintenance probes:

```sql
PRAGMA wal_checkpoint;
SELECT dolt_gc();
```

Do not use configurable checkpoint modes such as
`PRAGMA wal_checkpoint(TRUNCATE)` against DoltLite-format databases; DoltLite
rejects them.
