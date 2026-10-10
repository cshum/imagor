# Working in this repository

For automated agents and anyone new here. This is the repository-specific
knowledge that does not show up at a glance from the code. User-facing
documentation lives in `docs/`.

## Tests and goldens

- The suite needs libvips and the version matters. The project builds against the
  one in the `Dockerfile` base image, currently `vips8.18.6-r14`. Against a
  different libvips the `label*` and `text*` golden comparisons fail on rendering
  differences alone. Judge a run against `master`, not against zero.
- CI's **Docker Test** job runs the suite in that image. It settles a golden
  change; a local run does not.
- Goldens are in `testdata/golden/`, with per-architecture variants in
  `testdata/golden_arm64/`. Running the suite can leave a new file there, and an
  untracked one blocks a later `git checkout`.
- CI commits golden updates back to the branch as `test: update golden files`, so
  a pull request can gain a commit while you work on it.

## Documentation

- `docs/` is published on every push to `master` rather than on release, so the
  published documentation describes `master` and runs ahead of the latest
  release. A behaviour change and its documentation ship at different times.
- The filter reference in `docs/docs/filters.md` carries the accepted argument
  counts in its headings, written `name(args)` with square brackets for optional
  arguments.

## Code

- Filters are handled in more than one place: switches that run before the
  dispatch loop, and the loop itself. `disableFilters` is checked at each site,
  so a new site that misses it produces a filter that ignores the setting.
  Anything that counts filters has the same problem.
- `FilterFunc` in `processor/vipsprocessor` is a public extension point, so
  changing its signature breaks filters implemented outside this module.
- The `/meta` body is marshalled by the processor rather than by a response
  writer, so the fields a client reads are defined in `processor/vipsprocessor`.
