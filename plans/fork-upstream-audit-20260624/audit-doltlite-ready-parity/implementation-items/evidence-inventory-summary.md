# Evidence Inventory Summary

Schema: `gc.build.implementation-summary.v1`
Bead: `gc-9s5d`
Work item: `gc-1ljf` (`Audit DoltLite readiness evidence inventory`)

## Source Identity

- Source workspace: `gascity`
- Source path: `/data/projects/doltlite-gascity/gascity/.gc/workspaces/gascity/packs/gascity`
- Source change ID: `snrynqzxtknnntlruwytvklunnsxqtly`
- Source description: `source: audit DoltLite readiness evidence inventory`
- Source status before this summary: clean, no source file changes

## Document Identity

- Document workspace: `default`
- Artifact root: `plans/fork-upstream-audit-20260624/audit-doltlite-ready-parity`
- Document path: `plans/fork-upstream-audit-20260624/audit-doltlite-ready-parity/implementation-items/evidence-inventory-summary.md`
- Document change ID: `oqvnlwpvkkmnzzskzykmuoxqyknszsnx`
- Content hash: recorded in `manifest.json` and bead metadata after finalizing this file

## Scope

This item builds the current evidence inventory for the DoltLite readiness
audit. It does not change source code. Later audit lanes should treat archived
documents as checklist inputs and use the current source paths below as the
evidence surface.

## Confirmed Current Paths

Primary code and provider surfaces present in the source workspace:

- `cmd/gc/providers.go`
- `cmd/gc/beads_provider_lifecycle.go`
- `cmd/gc/bd_env.go`
- `cmd/gc/dolt_runtime_publication.go`
- `cmd/gc/hook_cross_store.go`
- `cmd/gc/store_target_exec.go`
- `cmd/gc/cmd_doctor.go`
- `cmd/gc/cmd_doctor_drift.go`
- `cmd/gc/dolt_standalone_conflict.go`
- `cmd/gc/dolt_cleanup_drop_planner.go`
- `cmd/gc/doltlite_store_native.go`
- `internal/beads/factory.go`
- `internal/beads/bdstore.go`
- `internal/beads/doltlite_read_store.go`
- `internal/beads/doltlite_count.go`
- `internal/beads/native_dolt_store.go`
- `internal/beads/caching_store.go`
- `internal/beads/contract/connection.go`
- `internal/beads/contract/preflight.go`
- `internal/beads/contract/preflight_checker.go`
- `internal/beads/exec/exec.go`
- `internal/beads/exec/testdata/conformance.sh`
- `tools/doltlite-client/README.md`
- `schemas/beads-doltlite/health/result.schema.json`

Operator-facing pack surfaces present in the source workspace:

- `examples/beads-doltlite/pack.toml`
- `examples/beads-doltlite/health_command_test.go`
- `examples/beads-doltlite/doctor/check-gc-doltlite-link/run.sh`
- `examples/beads-doltlite/doctor/check-sqlite3/run.sh`
- `examples/beads-doltlite/doctor/check-doltlite-metadata/run.sh`
- `examples/beads-doltlite/doctor/check-doltlite-read-fastpath/run.sh`
- `examples/beads-doltlite/doctor/check-doltlite-health/run.sh`
- `examples/beads-doltlite/commands/gc/run.sh`
- `examples/beads-doltlite/commands/client/run.sh`
- `examples/beads-doltlite/commands/health/run.sh`
- `examples/beads-doltlite/commands/health/schemas/result.schema.json`
- `examples/beads-doltlite/commands/flatten/run.sh`
- `examples/beads-doltlite/commands/sqlitebrowser/run.sh`
- `examples/beads-doltlite/assets/scripts/runtime.sh`

Historical checklist documents:

- Present in the default document workspace:
  `engdocs/contributors/dolt-regression-audit.md`
- Present in the default document workspace:
  `engdocs/archive/analysis/feature-parity.md`
- Present in the default document workspace:
  `engdocs/archive/analysis/gastown-upstream-audit.md`
- Missing from the sparse source workspace; use them as document-side checklist
  inputs, not current source evidence.

## Focused Tests To Inspect

Recommended `cmd/gc` tests:

- `cmd/gc/cmd_doctor_test.go`: `TestDoctorSkipsDoltChecksTreatsExecGcBeadsBdAsBdContract`, `TestBuildDoctorChecksSkipsManagedDoltChecksForDoltliteBackend`
- `cmd/gc/cmd_doctor_drift_test.go`: `TestDoltDriftCheckCleanManagedCityIsOK`, `TestDoltDriftCheckUsesProviderStateWhenPublishedStateIsMissing`, `TestDoltDriftCheckDetectsLiveRigLocalDolt`, `TestDoltDriftCheckTreatsLivePIDWithoutMatchingPortAsStale`, `TestDoltDriftCheckDetectsStaleRigLocalInfo`, `TestDoltDriftCheckDetectsPortFileDrift`
- `cmd/gc/cmd_doctor_dolt_local_only_test.go`: `TestDoDoctorRegistersLocalOnlyRemoteCheckForActiveManagedRigs`, `TestDoDoctorSkipsLocalOnlyCheckWhenGCDoltSkip`
- `cmd/gc/beads_provider_lifecycle_test.go`: `TestEnsureBeadsProvider_file`, `TestEnsureBeadsProvider_exec`, `TestProviderLifecycleProcessEnvProjectsCanonicalDoltPaths`, `TestProviderLifecycleProcessEnvDoesNotPropagateStrayTestModeEnv`
- `cmd/gc/bd_env_test.go`: `TestCityRuntimeProcessEnvStripsAmbientGCDolt`, `TestCityRuntimeProcessEnvUsesNativeOpenEnvSnapshotGuard`, `TestRecoverManagedBDCommandUsesNativeOpenEnvSnapshotGuard`
- `cmd/gc/store_target_exec_test.go`: `TestProviderUsesBdStoreContract`, `TestGcExecLifecycleInitProcessEnvDoesNotProjectCanonicalFilesOwnedFlagForGcBeadsBd`, `TestGcExecStoreEnvProjectsGCBinForGcBeadsBd`, `TestResolveConfiguredExecStoreTargetCity`
- `cmd/gc/dolt_lifecycle_race_test.go`: `TestStartManagedDoltFailsClosedWhenDataDirLockHeld`, `TestStopManagedDoltRefusesSIGKILLWhileLockHeld`, `TestStopManagedDoltWaitsForLockReleaseAfterExit`
- `cmd/gc/cmd_dolt_cleanup_test.go`: `TestRunDoltCleanup_JSONOutputsResolvedPort`, `TestRunDoltCleanup_HumanOutputShowsPortAndFallbackWarning`, `TestRunDoltCleanup_ForceReapsBareDeletedCwd`, `TestRunDoltCleanup_ForceKillsOrphans`
- `cmd/gc/gc_beads_bd_lint_test.go`: `TestDoltliteRuntimeConfigUsesSQLiteParameters`, `TestDoltliteInitCreatesBackendDirectoryBeforeBdInit`, `TestDoltliteMaintenanceDueUsesPortableStatFallback`
- `cmd/gc/dolt_start_managed_test.go`: `TestGCBeadsBDScript_DoesNotDefaultDoltGCScheduler`, `TestGCBeadsBDScript_DoesNotMutateDoltInternals`

Recommended `internal/beads` tests:

- `internal/beads/factory_test.go`: `TestOpenStoreAtForCityDoltliteOpensNativeStore`, `TestOpenStoreAtForCityExecBdContractFallbackUsesExecStore`, `TestOpenStoreAtForCityNativeOpenFailureFallsBackWithDiagnostic`
- `internal/beads/doltlite_read_store_test.go`: `TestDoltliteReadStoreListsSessionBeads`, `TestDoltliteReadStoreOpensExistingDoltliteDBWithStaleDoltMetadata`, `TestDoltliteReadStoreReadyUsesDoltlite`, `TestDoltliteReadStoreReadyCacheInvalidatesOnExternalWrite`
- `internal/beads/doltlite_count_test.go`: `TestDoltliteCountMatchesList`, `TestDoltliteCountUnsupportedShapes`, `TestDoltliteBoundedListIsCreatedDescPrefix`
- `internal/beads/native_dolt_store_test.go`: `TestNativeDoltStoreCreateDelegatesToUpstreamStorage`, `TestNativeDoltStoreMapsUpstreamStatusesToGasCityContract`
- `internal/beads/native_dolt_store_conformance_test.go`: `TestNativeDoltStoreConformance`
- `internal/beads/contract/preflight_checker_test.go`: `TestPreflightPassesOnHealthyDolt`, `TestPreflightAcceptsExecGcBeadsBdProviderPath`, `TestProviderUsesBDContract`
- `internal/beads/exec/exec_test.go`: `TestReady`, `TestExecStoreConformance`, `TestRunSanitizesAmbientLegacyAndStoreTargetEnv`
- `internal/beads/exec/br_script_test.go`: `TestGcBeadsBrReadyIncludeEphemeralFailsLoudlyUntilSupported`
- `internal/beads/exec/br_test.go`: `TestBrProviderConformance`

Recommended `examples/beads-doltlite` tests:

- `examples/beads-doltlite/health_command_test.go`: `TestDoltliteHealthScriptDoesNotForceDefaultShellTimeout`, `TestDoltliteBuildScriptBuildsGCWithNativeReadTag`, `TestDoltliteSqlitebrowserCommandBuildsAgainstLibdoltlite`, `TestDoltliteGCLinkDoctorRequiresNativeReadBuildTag`, `TestDoltliteHealthJSONSchemaIsValidObject`, `TestDoltliteHealthScriptOutputsJSONOKWithoutJq`

## Focused Commands For Later Lanes

Do not run the full Go suite. Use these approved focused commands when a lane
needs executable confirmation:

```bash
go test ./cmd/gc -run 'Test(BdRuntimeEnv|ResolvedRuntimeCityDoltTarget|DoltDriftCheck|PublishManagedDoltRuntimeState|StartManagedDolt|StopCityManagedBeadsProvider|DoctorSkipsDolt|DoDoctor.*Dolt|OpenStoreAtForCityExec|ControllerState.*Exec|CrossStore|PassthroughEnv.*Dolt|CityRuntime.*ManagedDolt)'
go test ./internal/beads -run 'Test(BdStore.*Doltlite|BdStoreReady|DoltliteReadStore|DoltliteCount|NativeDoltStore|CachingStore.*Ready|CachingStore.*Stale)'
go test ./internal/beads/contract -run 'Test(ResolveDoltConnectionTarget|Preflight|ProviderUsesBDContract|EnsureCanonicalConfig|EnsureCanonicalMetadata)'
go test ./internal/beads/exec -run 'Test(Ready|ExecStoreConformance|GcBeadsBrReadyIncludeEphemeral)'
go test ./examples/beads-doltlite -run 'TestDoltlite'
```

Commands run for this inventory were static file/path discovery and jj status
checks only. No Go tests were run for this item.

## Gaps And Path Corrections

- `engdocs/contributors/dolt-regression-audit.md`,
  `engdocs/archive/analysis/feature-parity.md`, and
  `engdocs/archive/analysis/gastown-upstream-audit.md` are present in the
  default document workspace but absent from the sparse source workspace.
- `doltlite/README.md` is missing from both the source and default workspaces.
  The available local API reference in this checkout is
  `tools/doltlite-client/README.md`.
- No standalone `cmd/gc/doctor_dolt.go` exists. Current Dolt doctor evidence is
  split across `cmd/gc/cmd_doctor.go`, `cmd/gc/cmd_doctor_drift.go`,
  `cmd/gc/cmd_doctor_test.go`, `cmd/gc/cmd_doctor_drift_test.go`, and
  `cmd/gc/cmd_doctor_dolt_local_only_test.go`.
- No `internal/beads/exec/store.go` or `internal/beads/exec/br.go` exists.
  Current exec-provider implementation and test evidence is in
  `internal/beads/exec/exec.go`, `internal/beads/exec/exec_test.go`,
  `internal/beads/exec/br_script_test.go`, `internal/beads/exec/br_test.go`,
  and `internal/beads/exec/testdata/conformance.sh`.
- `.gc/scripts/checks/build-artifact-valid.sh` is not available in the
  launcher rig root; `.gc/scripts/` currently contains only
  `.gc/scripts/gc-beads-bd.sh`.

## Result

The current checkout has evidence surfaces for REQ-001, REQ-002, and REQ-006:
current provider/runtime files, bead-store implementation files, focused tests,
operator-facing pack scripts, command surfaces, and schemas are listed above.
Later lanes should use this inventory to build the regression coverage matrix,
provider-boundary review, and final readiness audit without assuming archived
documents prove current behavior.
