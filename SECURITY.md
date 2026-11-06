# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| Latest release | Yes |
| Older releases | No |

Only the latest release, as running on dezetrein.com, gets security fixes.

## Reporting a Vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**

Report them privately by emailing **git@pvleeuwen.com**. Include:

- A description of the vulnerability
- Steps to reproduce or a proof of concept
- The affected component (backend API, website)
- Any potential impact you've identified

### What to Expect

- **Acknowledgement** within 3 business days
- **Status update** within 10 business days
- We will work with you to understand and resolve the issue before any public disclosure

### Scope

This policy covers the code in this repository and the site at dezetrein.com. Issues with the NS
API itself should go to NS through the [NS API portal](https://apiportal.ns.nl).

## Security Best Practices for Contributors

- Never commit secrets, API keys or credentials, and never commit `.env` files
- Validate all input at system boundaries
- Never log or store a user's location; it is used in memory for one request only
- Keep calls to the NS API behind the shared cache, never one upstream call per user request
