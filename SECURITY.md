# Security

## Reporting a vulnerability

Use GitHub's private vulnerability reporting: open the Security tab of this repository and choose "Report a
vulnerability". If that is not available to you, email tkipper@gmail.com. Please don't open a public issue.

Include what you found, how to reproduce it, and its impact. Redact real device addresses and credentials
first. You can expect a first reply within a few days. This is a project maintained by one person, so fixes
are best effort, and there is no bug bounty.

## Trust model

- **Credentials travel in the URL path.** The Go services take device credentials per request, following the
  OpenAV `socketKey` convention. Run them only on a trusted local network, never exposed to the internet. The
  demo stack binds its ports to 127.0.0.1.
- **Container logs contain device passwords.** Our own driver code masks the password in its log lines
  (`source/redact.go`), but the upstream OpenAV framework (`source/microservice-framework`) logs the full socket
  key, password included, on every GET and PUT and in many error messages, and its `GET /<address>/errors`
  endpoint returns those messages. Restrict who can read container logs and who can reach the services.
- **The EC20's VISCA port has no authentication.** Raw VISCA on TCP 5678 ignores credentials, so anyone who can
  reach the camera can move it. Network placement is the control.
- **EC20 CGI login tokens are bound to credentials.** A cached token is keyed by host plus a SHA-256 of the
  credentials, lasts at most 10 minutes, and is never reused for a caller with different credentials.
- **Device credentials go to the devices over plain HTTP.** The Pearl service uses HTTP Basic, and the EC20 CGI
  layer uses an MD5 digest. Neither service offers TLS to the device yet.
- **`openav-mcp`** resolves credentials from `OPENAV_DEVICES` so the calling agent never sees one, and exports
  control tools by default. Start it with `--read-only` for any deployment that must not drive hardware; disabled
  tools are both hidden and refused. A credential appearing in a tool result or log line is a high-priority report.

## Known issues

- **CVE-2026-55677 in `labstack/echo` v3.3.10** (URL path decoding discrepancy), reported by Trivy 0.76.2 against
  both service binaries as "affected" with no fixed version. The upstream OpenAV framework depends on echo v3;
  the fix is a framework move to echo v4.
  Given that credentials ride in the URL path, treat the services as trusted-network only.
- **Lab addresses in history.** Earlier commits contain bench device addresses and the EC20 factory default login.
  The current tree uses only documentation addresses (192.0.2.0/24) and placeholder credentials.
