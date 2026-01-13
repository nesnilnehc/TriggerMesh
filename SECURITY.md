# Security Policy

## Supported Versions

We actively maintain the following versions of TriggerMesh and provide security updates:

| Version | Supported          |
| ------- | ------------------ |
| 1.0.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability, please do not report it in a public issue.

### How to Report

Please report security vulnerabilities through the following methods:

1. **GitHub Security Advisories** (Recommended):
   - Visit the repository's Security tab
   - Click "Report a vulnerability"
   - Fill in the vulnerability details

2. **Email** (If GitHub Security Advisories is not available):
   - Create an issue with `[SECURITY]` prefix
   - Describe the vulnerability in detail
   - Include possible exploitation methods (if applicable)
   - Provide remediation suggestions (if applicable)

### Report Content

Please include the following information:

- Detailed description of the vulnerability
- Affected components or features
- Potential exploitation methods
- Steps to reproduce
- Impact assessment (severity level)
- Remediation suggestions (if applicable)

### Process

1. **Acknowledgment**: We will acknowledge receipt of your report within 48 hours
2. **Initial Assessment**: We will perform an initial assessment within 7 days
3. **Detailed Analysis**: If needed, we will complete a detailed analysis within 14 days
4. **Remediation Plan**: We will develop a remediation plan and notify you
5. **Release**: After remediation is complete, we will release a security update
6. **Disclosure**: After release, we may publicly disclose the vulnerability (with your permission)

### Vulnerability Severity

We use the following criteria to assess vulnerability severity:

- **Critical**: May lead to complete system compromise
- **High**: May lead to sensitive data disclosure or service interruption
- **Medium**: May lead to limited data disclosure or functionality restrictions
- **Low**: Minor impact, difficult to exploit

### Security Best Practices

When using TriggerMesh, please follow these security best practices:

1. **API Keys**:
   - Use strong random API Keys
   - Rotate API Keys regularly
   - Do not commit API Keys to version control

2. **Jenkins Credentials**:
   - Use the principle of least privilege for Jenkins Token configuration
   - Rotate Jenkins Tokens regularly
   - Do not hardcode credentials in configuration files

3. **Network**:
   - Use HTTPS in production environments
   - Restrict network access to the TriggerMesh API
   - Use firewall rules to limit access

4. **Updates**:
   - Regularly update TriggerMesh to the latest version
   - Pay attention to security announcements and changelogs

### Security Updates

Security updates will be released through:

- GitHub Releases
- CHANGELOG.md
- Security Advisories (if applicable)

Thank you for helping us keep TriggerMesh secure!
