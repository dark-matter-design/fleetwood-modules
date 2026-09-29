# Fleetwood Modules

Modules for [Fleetwood](https://github.com/dark-matter-design/fleetwood), and the signed repository
Fleetwood installs them from. The repository is published with GitHub Pages, so it can be added to a
Fleetwood instance as an ordinary custom repository over HTTPS.

## Add this repository to Fleetwood

In Fleetwood, go to Modules, then Repositories, then Add Repository:

| Field | Value |
| --- | --- |
| Repository Name | Fleetwood Modules |
| Repository URL | `https://dark-matter-design.github.io/fleetwood-modules` |
| Minisign Public Key | `RWSNisIOX9UCZzdVq5CZoankPwHm5FJjQ4GgI6+wFKydD23pLVuVN/IJ` |

The URL is the directory holding `index.json`; Fleetwood appends the filename itself. With the
public key in place the repository is trusted as signed, and installs do not raise the unsigned
warning. Leave the key out and every install from it asks you to accept an unproven origin instead.

The same key is published at
[`/minisign.pub`](https://dark-matter-design.github.io/fleetwood-modules/minisign.pub). Prefer the
copy in this README: a key fetched from the server it is meant to vouch for proves nothing on its
own.

## Modules

| Module | ID | Version | What it is |
| --- | --- | --- | --- |
| [Hello Fleet](modules/hello-fleet) | `com.fleetwood.hello` | 0.1.0 | Reference module: a declarative page, a widget and a `say-hello` job type |

Hello Fleet is a copy of the reference module in the Fleetwood repository, published here so the
module store can be exercised against a real repository. Fleetwood keeps its own copy under
`modules/examples/hello-fleet` for its tests, and that copy remains the source of truth. Once real
modules live here, this copy can be dropped.

## Layout

```
modules/<name>/     one module: manifest.yaml, main.go, go.mod, icon.svg, LICENSE, CHANGELOG.md, docs/
docs/               the published repository: index.json, its signature, and each signed .fwmod
.keys/              signing keys, never committed
```

## Building the repository

Building needs the Go toolchain and the `fleetwood-mod` CLI, both of which live in the Fleetwood
checkout. Fleetwood is private and cannot be fetched with `go run`, so point `FLEETWOOD` at a local
clone; it defaults to `../fleetwood`.

```sh
make keygen              # once: creates .keys/modsign.{key,pub}
make publish             # builds every module, packs, signs, writes docs/
make verify              # checks docs/ against its own index
make publish FLEETWOOD=/path/to/fleetwood
```

`make publish` rewrites `docs/` and bakes `BASE` into `index.json`, so the base URL has to match
where the repository is actually served. Commit `docs/` and push; Pages serves it from the default
branch. To serve it somewhere else, override the base:

```sh
make publish BASE=http://192.168.1.170:8099
```

Adding a module means adding a directory under `modules/`; the Makefile picks it up and names the
package `<directory>-<manifest version>.fwmod`.

## Signing key

The key in `.keys/` is a development key with no password, generated for this demo repository. It is
gitignored and stays on the machine that publishes. Anything with real users wants a passworded key
kept somewhere durable, because losing it means every client that trusted it has to be updated by
hand.

## Licence

MIT. See [LICENSE](LICENSE).
