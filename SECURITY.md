# Security Policy

## Scope

Rowback is a local, single-user command-line / desktop tool. It reads a
mysqldump `.sql` file from a path the user supplies and writes restore SQL back
to disk. When run with `-serve`, it starts a web UI bound to **loopback only**
(`127.0.0.1`) — it is not intended to be exposed to a network. File paths and
SQL passed to the tool are trusted user input, not attacker-controlled.

## Supported Versions

Only the latest released `v0.x` version receives security fixes.

| Version | Supported |
| ------- | --------- |
| latest `0.x` | ✅ |
| older | ❌ |

## Reporting a Vulnerability

Please report suspected vulnerabilities privately via
[GitHub Security Advisories](https://github.com/albertovincenzi/rowback/security/advisories/new)
rather than opening a public issue.

Include:

- affected version / commit
- a description and, ideally, a minimal reproduction
- the impact you believe it has

We aim to acknowledge reports within 7 days and to ship a fix or mitigation for
confirmed issues as soon as practical. Please do not publicly disclose until a
fix is available.
