# Third-Party Notices

This repository contains or depends on third-party software.

## Vendored source

### ModSecurity

- Component: `ModSecurity/`
- Upstream: `owasp-modsecurity/ModSecurity`
- License: Apache License 2.0
- Local license file: `ModSecurity/LICENSE`

The `ModSecurity/` directory is kept as vendored upstream source for future modification. If files inside that tree are changed and then redistributed, the modified files should carry clear notices stating that they were changed, as required by Apache-2.0.

## Runtime container dependencies

### ModSecurity CRS nginx image

- Component: Docker image `owasp/modsecurity-crs:nginx-alpine`
- Purpose: nginx + ModSecurity connector + OWASP Core Rule Set runtime for the WAF data plane

When redistributing container images or bundled artifacts, include the applicable upstream license texts and notices for the shipped components.
