# Working in this repository

For automated agents and anyone new here. Only what the code and the published
documentation do not already show. Where a fact has a source, this points at the
source rather than restating a value that will change.

## Tests and goldens

- The suite needs libvips, and the golden comparisons depend on its version. That
  version is pinned in the `Dockerfile` base image, so take it from there. Against
  a different one, the `label*` and `text*` goldens fail on rendering differences
  alone, so judge a run against `master` rather than against zero failures.
- CI's **Docker Test** job runs the suite in that image. It is what settles a
  golden change; a local run is not a substitute.
- Goldens are in `testdata/golden/`, with per-architecture variants in
  `testdata/golden_arm64/`. Running the suite can leave a new file there, and an
  untracked one blocks a later `git checkout`.
- CI commits golden updates back to the branch as `test: update golden files`, so
  a pull request can gain a commit while you work on it.
- The package shares one vips instance, started in `TestMain`. A test that shuts
  its processor down shuts vips down with it and breaks every test that runs
  after, which is why some tests deliberately do not.

## Documentation

- `docs/` is published on every push to `master` rather than on release, so the
  published documentation describes `master` and runs ahead of the latest
  release. A behaviour change and its documentation ship at different times.

## Code

- Filters are handled in more than one place: switches that run before the
  dispatch loop, and the loop itself. `disableFilters` is checked at each site,
  so a new site that misses it produces a filter that ignores the setting.
  Anything that counts filters has the same problem.
- The filter reference and the processor's `FilterMap` are independent lists.
  Some registered filters are undocumented, and some documented filters are
  handled by a plugin rather than by this processor.
- The `/meta` body is marshalled by the processor rather than by a response
  writer, so the fields a client reads are defined in `processor/vipsprocessor`.
- An option lives in three places read by different people: the `With...` option's
  own comment carries the precise rule, and the flag help together with
  `docs/docs/configuration.md` carries what a user needs.
