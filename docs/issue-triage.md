# Issue triage — 2026-10-25

Scope: all 12 open issues, including their comments, compared with main after
[PR #605](https://github.com/cpanato/mattermost-plugin-alertmanager/pull/605) and
the release-candidate validation in [PR #617](https://github.com/cpanato/mattermost-plugin-alertmanager/pull/617).
The candidate is prepared; publication awaits that PR's review and merge.
The list below is ordered by
effort. Priority reflects impact separately; routing correctness remains more
urgent than cosmetic changes even when both are easy to assess.

This is a disposition proposal. Issues have not been closed or commented on.
An old report alone is not evidence of resolution. The RC targets Mattermost
11.7+, so a fix verified there does not prove compatibility with the old 9.x
servers mentioned in several reports.

| Order | Issue | Effort / priority | Assessment and next action |
| --- | --- | --- | --- |
| 1 | [#113 — naming and README](https://github.com/cpanato/mattermost-plugin-alertmanager/issues/113) | Small / low | README now explicitly identifies and links Prometheus Alertmanager; refresh screenshots. Recommend closing as documentation addressed after the RC preparation merges. Keep the plugin ID stable to preserve installations; an ID rename is unnecessary migration risk. |
| 2 | [#1 — multiple servers, links, annotations](https://github.com/cpanato/mattermost-plugin-alertmanager/issues/1) | Small verification / medium | All requested capabilities exist: multiple configurations, GeneratorURL/ExternalURL links and annotation fields. Routing regression and current screenshots provide evidence. Recommend closing as implemented, linking the RC and noting its minimum server version. |
| 3 | [#20 — distinguish clusters](https://github.com/cpanato/mattermost-plugin-alertmanager/issues/20) | Small verification / medium | Separate configurations can route clusters to distinct channels; posts show configuration ID and receiver. Recommend closing as implemented after the RC is available, with a two-config example. Labels can also identify the cluster. |
| 4 | [#116 — context canceled](https://github.com/cpanato/mattermost-plugin-alertmanager/issues/116) | Small verification / high | A concrete bug canceled the HTTP context before decoding its body. Fixed in #605 and covered by a delayed-body test. Recommend closing as fixed in the RC with this specific explanation; proxy/network failures can still have independent causes. |
| 5 | [#394 — random channel routing](https://github.com/cpanato/mattermost-plugin-alertmanager/issues/394) | Small verification / high | Comments identify the reversed token comparison in v0.4.0 and confirm the existing main fix (#118). The RC includes it, rejects mismatched tokens and tests repeated routing across two configurations. Recommend closing when the RC is published, linking the fix and removing the old advice to quote token values. Users of Mattermost 9.4.1 must upgrade before installing this RC. |
| 6 | [#385 — plugin startup / GLIBC](https://github.com/cpanato/mattermost-plugin-alertmanager/issues/385) | Small investigation / high | Comments show GLIBC_2.32/2.34 dependencies in the old artifact. New binaries use CGO_ENABLED=0; the generated Linux amd64 binary was verified as statically linked and starts on the supported ESR. Request retest on a supported server. Do not claim the reported 9.x installation is fixed or silently close it solely because 9.x is unsupported. |
| 7 | [#504 — TLS / deadline exceeded](https://github.com/cpanato/mattermost-plugin-alertmanager/issues/504) | Small initial diagnostics, uncertain fix / high | Keep open; request reproducible TLS/proxy configuration, CA chain and connectivity from inside the Mattermost container, using a supported server and the RC. Retrying cannot fix certificate trust, DNS or a stalled handshake. The error shown may come from metadata fetching as well as the command client; distinguish those paths. Never suggest disabling TLS verification as a default. |
| 8 | [#7 — Basic Auth](https://github.com/cpanato/mattermost-plugin-alertmanager/issues/7) | Medium / medium-high | Keep open and review [PR #604](https://github.com/cpanato/mattermost-plugin-alertmanager/pull/604), rebasing onto the new SDK/client as needed. Verify secret handling/redaction, correct and incorrect credentials, redirects and all command methods. Ship separately after the RC, avoiding an untested authentication change in release preparation. |
| 9 | [#11 — severity colors](https://github.com/cpanato/mattermost-plugin-alertmanager/issues/11) | Small–medium / medium | Useful focused enhancement. Define a configurable label-to-color mapping and preserve resolved-state coloring. Decide grouped-alert behavior when severities differ. Existing status coloring does not satisfy this request. |
| 10 | [#174 — presentation improvements](https://github.com/cpanato/mattermost-plugin-alertmanager/issues/174) | Medium / medium | Partly overlaps #11 and #19. Keep open as a compact-message design task: emphasize summary, reduce optional source metadata and link the color work. Specify one representative grouped-alert example before implementation; do not label fully fixed by the dependency update. |
| 11 | [#18 — duplicate HA alerts](https://github.com/cpanato/mattermost-plugin-alertmanager/issues/18) | Medium diagnosis, large if plugin dedup is needed / medium | First obtain peer/cluster configuration, identical payloads and partition behavior. Alertmanager already performs grouping/deduplication. Diagnose independent replicas or failed gossip before adding plugin state. If genuine plugin-side idempotency is needed, define tenant/config scope, firing vs resolved keys, retry behavior and atomic shared storage for Mattermost HA. Keep open. |
| 12 | [#19 — custom templates](https://github.com/cpanato/mattermost-plugin-alertmanager/issues/19) | Large / medium | Useful but broad: define a template contract, default compatibility, validation, escaping, size limits, admin preview and safe failure behavior. A compact built-in format (#174) is a smaller first step. Keep open; no need to block the maintenance release on this feature. |

## Suggested sequence

1. Publish the tested RC and refresh the documentation/screenshots.
2. Close only issues with evidence of implementation: #113, #1, #20, #116 and #394, explaining the 11.7+ requirement. Request reproduction for #385 and #504.
3. Review/rebase Basic Auth PR #604. Implement severity colors as a separate change with a narrow contract.
4. Design compact messages, investigate HA reports, then consider full templates if the simpler format is insufficient.

For HA diagnosis, see the official [Alertmanager overview](https://prometheus.io/docs/alerting/latest/alertmanager/)
and [HA documentation](https://prometheus.io/docs/alerting/latest/ha/). Alertmanager
provides deduplication, but HA behavior must be assessed before promising strict
exactly-once delivery from an integration.
