# Maintenance assessment (2026-10-25)

Mattermost continues to support server and web app plugins. The plugin API wire
format remains available; plugins run as Go subprocesses. Custom plugin uploads
must be enabled on the server. This repository remains Apache-2.0 licensed.

The old `mattermost-plugin-api` repository was archived on February 28, 2025.
Its helpers now live in `github.com/mattermost/mattermost/server/public/pluginapi`.
Both Go modules now use the public SDK v0.6.0 and require Go 1.27.0 or later.
Alertmanager is updated to v0.34.1. Its removed `types.Silence` export is replaced
with a local API v2 response type, covered by a wire-format test.

The baseline is now Mattermost 11.7, the supported ESR through May 15, 2027.
This deliberately drops the old manifest's 7.5 baseline. Mattermost 10.11 ESR
reached end of support on August 15, 2026. Run the Docker recipe before deploying
this change: compilation alone does not establish compatibility with a server.

Mattermost 11.1 introduced React 18; Mattermost 12 uses React 19. The plugin
continues to use the host's external React with the classic JSX transform.
The old React Bootstrap modal was replaced with a native dialog to remove its
legacy React API dependency. React 18 stays as the development dependency so the
source can target both the ESR and the React 19 host. Redux, mattermost-redux,
Superagent, Lodash and Node crypto polyfills were unused and removed. Tokens
now use browser `crypto.getRandomValues`. Node 24 is the build baseline.
ESLint stays on 9 because eslint-plugin-react 7.37.5 does not support ESLint 10;
upgrade them together once the upstream peer dependency permits it.

Other fixes include configuration cloning, channel-map locking, avoiding plugin
reactivation inside the initial configuration hook, immutable settings updates,
returning HTTP 500 on failed posts (so Alertmanager retries), preserving HTTP
response contexts until decoding finishes, closing failed retry responses,
and stopping silence-action handling after an upstream error.

The local helper is restricted to `http://localhost:8065`. It creates a known
administrator and overwrites the Alertmanager plugin's configuration in that
local instance. Do not use it against a real deployment. Data persists across
`make dev-down` / `make dev-up`; Docker volumes can be removed manually when a
fresh instance is needed.

Before releasing, test both the default ESR and the desired 12.x server:

- Open System Console, add/edit/delete configurations and regenerate tokens.
- Send firing and resolved notifications with `make dev-alert`.
- Run `/alertmanager status`, `/alertmanager alerts`, `/alertmanager silences`.
- Create a silence in Alertmanager and expire it with the command and button.
- Restart Mattermost and confirm the plugin activates with the saved config.

Validation completed locally on Mattermost 11.7.14 and 12.0.0 with Alertmanager
0.34.1: plugin upload/activation, firing and resolved bot
posts, and acceptance of the status, alerts and silences commands. Each server
used separate Docker volumes and PostgreSQL 17.7. The ESR was then upgraded to
PostgreSQL 17.10 and its smoke test passed again. It is left running for exploration.
The same API smoke test is available as `make dev-smoke` and in the manually
triggered integration workflow. It does not exercise browser interactions or
assert slash-command response contents; the UI and silence-action release
checks above remain manual.

Go race tests, ESLint, golangci-lint v2.15.0 (both modules), the five-platform
bundle build, Compose validation and GoReleaser configuration validation passed.
The npm audit reports no vulnerabilities. Current govulncheck reports no
vulnerabilities in reachable code or imported packages, with one advisory in a
required module that is not imported. Importing the public webhook data instead
of Alertmanager's notifier implementation removed unnecessary transport and
OpenTelemetry exporter dependencies and a reachable exporter advisory.

Runtime validation also caught two packaging/environment issues: macOS tar
metadata prevents manifest discovery on Linux, so bundle creation uses
`COPYFILE_DISABLE=1`; the official Mattermost image has no shell or curl, so its
healthcheck uses the bundled `mmctl --local system status` command.

Sources:

- [Plugin support and upload requirements](https://docs.mattermost.com/developers/integrate/plugins/using-and-managing-plugins)
- [Archived API helper and migration notice](https://github.com/mattermost/mattermost-plugin-api)
- [Mattermost release policy](https://docs.mattermost.com/product-overview/release-policy)
- [React 18 upgrade](https://forum.mattermost.com/t/upgrading-the-mattermost-web-app-to-react-18-v11-1/25000)
- [Mattermost 12 changes, including React 19](https://mattermost.com/blog/what-will-be-deprecated-in-mattermost-v12-0-october-2026/)
