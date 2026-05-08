# Security Policy

## Supported Versions

agentd is pre-1.0 software. Security fixes are handled on the default branch and released from the latest public code.

## Reporting a Vulnerability

Do not open a public issue for suspected vulnerabilities.

Report security concerns by emailing the project maintainer or by using GitHub private vulnerability reporting if it is enabled for the repository. Include:

- A concise description of the issue.
- Steps to reproduce or a proof of concept.
- Affected commit, version, or configuration.
- Any relevant logs with secrets removed.

## Secret Handling

agentd treats provider credentials and local access paths as sensitive operational inputs.

- Do not commit `.env` or `.env.*` files.
- Do not place API key values in Agent Definition Markdown.
- Do not paste credentials into issues, pull requests, examples, or logs.
- Keep `OPENAI_API_KEY` as an environment variable name only in committed files.

If you accidentally commit a secret, rotate it immediately and rewrite repository history before public release.
