# Security Policy

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them via GitHub's private vulnerability reporting:

1. Go to the repository's **Security** tab in the top navigation
1. In the Security tab, click **Report a vulnerability**
1. Fill out the form with details

### What to Include

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

### Response Timeline

- **Acknowledgment:** Within 24 hours
- **Initial assessment:** Within 48 hours
- **Resolution target:** Depends on severity, but as soon as possible

### What Qualifies as a Security Issue

- Command injection vulnerabilities (e.g., via crafted snapshot names or disk labels)
- Privilege escalation through `tmutil` or `diskutil` interactions
- Path traversal or directory traversal
- Sensitive data exposure
- Credential exposure risks

### Out of Scope

- Issues in upstream dependencies (report to them directly)
- Issues requiring physical access to the machine
- Social engineering attacks
