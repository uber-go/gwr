v0.7.1

- Deadlock fix
- Make `(*"internal/marshaled".DataSource).Active` less contentious

v0.7.0
- Many improvements to the tracer
  - Ability to walk up the scope chain with the scope.Root() and scope.Parent()
    methods.
  - Track begin and end time for every scope.
  - Added a standalone example fibonacci test
- Added a default stringer text format for sources that define no text template.
- Added a simple formatted reporter, which can be used to wire up a GWR source
  to a file or log stream (any Printf-like func).
- Removed the extraneous "OK" response from the RESP monitor command; this
  makes it easier to process JSON output.
- Added ability to drain a source; this allows a test, or other non-daemon
  program, to block until one or more GWR sources have finished sending any
  pending items to their watcher(s).
- Added a convenience implementation of marshaled data source format around a
  single function.

v0.6.5
- Fixed a bug in tracer that was causing it to active even with no watchers.

v0.6.4
- Fix potential problems with marshaled data source state under concurrency
- Fixed text output from /meta/nouns
- Improve tracer text output
- Other minor fixes to tests and documenetation

v0.6.3
- Fix potential concurrency problem for tap trace ids
- Fix subtle problems around marshaled data source channels
- Fix potential race condition

v0.6.2
- Initial intended release
