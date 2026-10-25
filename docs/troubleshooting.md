# Troubleshooting Alertmanager connections

Slash commands connect from the Mattermost plugin process to the Alertmanager
URL configured in System Console. Successfully opening that URL in a browser,
or reaching it from the Mattermost host, does not establish that the plugin's
container has the same DNS, routing, proxy settings or certificate trust.

## Certificate verification errors

Errors such as `certificate signed by unknown authority`, a hostname mismatch
or an expired certificate are returned immediately. Retrying the same request
cannot repair them. Previously these failures were retried until the shared
15-second deadline, which hid the certificate error behind
`context deadline exceeded`.

For private CAs, make the issuing CA available in the trust store used by the
Mattermost plugin process. Check that the server sends the intermediate
certificates and that the URL's hostname matches the certificate. Keep TLS
verification enabled; using HTTP just to bypass certificate checks does not
repair the HTTPS configuration.

## Connection and TLS handshake timeouts

The client has a 15-second deadline covering retries, redirects and reading the
response body. Transient failures still retry within that budget. A timeout
alone does not identify a certificate problem: inspect DNS, routing, firewalls,
proxy configuration and upstream responsiveness from the plugin's environment.
The default Go transport honors `HTTP_PROXY`, `HTTPS_PROXY` and `NO_PROXY`.

Use the exact configured HTTPS endpoint, with the same network access and CA
trust as the plugin, to inspect the connection and response:

```sh
curl --verbose --connect-timeout 10 --max-time 15 \
  https://alertmanager.example.com/api/v2/status
```

Check HTTP-to-HTTPS redirects separately with `curl --location`; configuring the
final HTTPS URL avoids spending the request budget on redirects. Redirects must
also preserve the request method for silence deletion (HTTP 307 or 308).

A Mattermost log such as `error fetching OG image data` from `app/post_metadata.go`
belongs to link-preview fetching. It is a separate request from the plugin's
Alertmanager API call and does not by itself establish why a slash command failed.

When reporting a persistent failure, include the plugin and Mattermost versions,
deployment type, whether the URL uses HTTPS directly or redirects, whether a
private CA or proxy is involved, and the complete connection error with sensitive
values removed.
