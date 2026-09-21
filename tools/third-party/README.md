# Third-party notices

After changing dependencies, run from the repository root:

```sh
npx hereby update:third-party
```

This requires network access, Node, Go, [uv](https://docs.astral.sh/uv/), and
dependencies installed with `npm ci`. It refreshes the dependency inventory,
NOTICE, and `cgmanifest.json`. Review and commit the changes in `generated/`.

When importing new DOM libraries, first update `domGenerator` in
`inputs/dependencies.json` to the generator commit used.
For copied source, update its repository, commit, and tag under
`copied` in that file.

To regenerate the outputs from the existing inventory without network access:

```sh
npx hereby generate:third-party
```
