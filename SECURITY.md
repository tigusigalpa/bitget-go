# Security policy

## Reporting a vulnerability

Please do not open a public issue for a suspected vulnerability. Email
[sovletig@gmail.com](mailto:sovletig@gmail.com) with a short description,
affected versions, impact, and reproducible steps. Do not include API keys,
secret keys, passphrases, signatures, or account data.

We will acknowledge the report, investigate it privately, and coordinate a
fix and disclosure timeline with you.

## Credentials

This SDK never needs credentials at package-import or client-construction
time. Keep API credentials in a secret manager or environment variables,
prefer Demo keys while developing, bind keys to an IP address where possible,
and grant only the permissions your application needs.
