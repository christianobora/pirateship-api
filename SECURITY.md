# Security policy

## Reporting a vulnerability

Please report vulnerabilities through a private GitHub security advisory for
this repository. Do not open a public issue for a vulnerability.

Include the affected version, impact, and a minimal reproduction. Remove all
addresses, names, email addresses, phone numbers, tracking numbers, label URLs,
payment details, and session values before submitting a report.

## Credential handling

This library never needs direct access to browser profiles or credential
stores. Applications provide an `http.Client` and remain responsible for
interactive authentication, cookie-jar protection, TLS policy, and session
revocation. Do not serialize or log an authenticated client's cookie jar.

Only HTTPS endpoints are accepted. Custom GraphQL headers are not forwarded to
label artifact hosts.
