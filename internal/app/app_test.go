package app

// Behavioral snapshot tests covering app.go's package-level helpers and lifecycle hooks.
//
// Deliberately left uncovered here:
//   - startup(ctx): the full bootstrap sequence, it opens the real
//     config/database, makes a live network call via core.InitTidalEndpoints,
//     and starts goroutines bound to runtime.EventsEmit(ctx, ...). Without an
//     actual Wails runtime context, any runtime.* call panics via log.Fatalf
//     (see wails/v2/pkg/runtime/runtime.go). This is an integration-level
//     entrypoint rather than something unit-testable.
//   - shutdown(ctx): stops the download manager and persists config/db.
//     Aside from trivial nil-guards, testing it meaningfully needs the same
//     real dependencies startup does.
