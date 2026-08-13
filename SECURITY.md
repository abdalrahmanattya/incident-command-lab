# Security policy

This service accepts security reports through a private maintainer channel before
public disclosure. Do not include credentials, customer data, or cloud
identifiers in reports. The Compose stack is localhost-only synthetic data;
never expose its anonymous Grafana viewer or local database to the Internet.

Cloud acceptance requires protected OIDC environments, short-lived resources,
and destroy verification. Model output is untrusted advisory text and cannot
execute remediation.
