# Security Policy

## Supported Versions

We release patches for security vulnerabilities in the following versions:

| Version | Supported          |
| ------- | ------------------ |
| main    | :white_check_mark: |
| 0.1.x   | :white_check_mark: |
| < 0.1   | :x:                |

**Note:** As kube-changejob is currently in early development, we recommend always using the latest release or main branch for the most up-to-date security fixes.

## Reporting a Vulnerability

We take the security of kube-changejob seriously. If you believe you have found a security vulnerability, please report it to us as described below.

### Please DO NOT:

- Open a public GitHub issue
- Disclose the vulnerability publicly before it has been addressed
- Use the vulnerability for any purpose other than reporting it

### Please DO:

**Report security vulnerabilities using GitHub's Security Advisory feature:**

1. Go to https://github.com/nusnewob/kube-changejob/security/advisories/new
2. Fill out the advisory form with as much detail as possible
3. Click "Submit report"

**Alternatively, you can:**

- Email the maintainers directly (check the repository for current maintainer contact information)
- Provide detailed steps to reproduce the vulnerability
- Include the version(s) affected
- Describe the potential impact

### What to Include in Your Report

To help us understand and address the issue quickly, please include:

- **Description**: A clear description of the vulnerability
- **Impact**: What an attacker could achieve by exploiting this vulnerability
- **Reproduction Steps**: Detailed steps to reproduce the issue
- **Affected Versions**: Which versions of kube-changejob are affected
- **Environment**: Kubernetes version, platform, and any other relevant details
- **Proof of Concept**: If possible, include a minimal proof of concept
- **Suggested Fix**: If you have ideas on how to fix it (optional)

## Response Timeline

- **Acknowledgment**: We will acknowledge receipt of your vulnerability report within 3 business days
- **Initial Assessment**: We will provide an initial assessment within 7 business days
- **Status Updates**: We will keep you informed about our progress
- **Resolution**: We aim to resolve critical vulnerabilities within 30 days
- **Disclosure**: We will coordinate with you on public disclosure timing

## Security Update Process

When a security vulnerability is confirmed:

1. A fix will be developed in a private repository
2. A security advisory will be drafted
3. A new version will be released with the fix
4. The security advisory will be published
5. Users will be notified through:
   - GitHub Security Advisories
   - Release notes
   - GitHub Discussions (if applicable)

## Security Best Practices

When using kube-changejob, we recommend:

### Deployment Security

- **RBAC**: Use Kubernetes RBAC to limit the operator's permissions to only what's necessary
- **Network Policies**: Implement network policies to restrict network access
- **Pod Security Standards**: Apply appropriate Pod Security Standards to the operator namespace
- **Image Scanning**: Scan container images for vulnerabilities before deployment
- **Private Registry**: Use a private container registry for production deployments

### ChangeJob Configuration Security

- **Least Privilege**: Configure Jobs with minimal required permissions
- **Resource Limits**: Always set resource limits on spawned Jobs
- **Secret Management**: Use Kubernetes Secrets properly and consider using secret management solutions
- **Validation**: Validate all ChangeJob manifests before applying them
- **Namespace Isolation**: Deploy ChangeJobs in appropriate namespaces with proper isolation

### Monitoring and Auditing

- **Audit Logging**: Enable Kubernetes audit logging
- **Monitoring**: Monitor operator logs for suspicious activity
- **Alerting**: Set up alerts for unusual ChangeJob patterns
- **Regular Updates**: Keep kube-changejob updated to the latest version

### Supply Chain Security

- **Verify Images**: Verify container image signatures when available
- **Dependency Scanning**: Regularly scan dependencies for vulnerabilities
- **SBOM**: Review the Software Bill of Materials (SBOM) if available

## Known Security Considerations

### Kubernetes Permissions

The kube-changejob operator requires certain Kubernetes permissions to function. Review the RBAC configuration in `config/rbac/` to understand what permissions are granted.

### Job Execution

ChangeJobs trigger the creation of Kubernetes Jobs. Ensure that:

- Job templates are from trusted sources
- Jobs run with appropriate service accounts
- Jobs have resource limits to prevent resource exhaustion

### Watch Resources

The operator watches Kubernetes resources for changes. Be aware that:

- Sensitive data in watched resources may be logged
- Rapid changes to watched resources could trigger many Jobs
- Consider rate limiting and backoff strategies

## Security Disclosure Policy

After a security issue is fixed:

- We will publish a security advisory on GitHub
- The advisory will include:
  - Description of the vulnerability
  - Affected versions
  - Fixed versions
  - Severity rating (using CVSS)
  - Workarounds (if applicable)
  - Credit to reporters (if desired)

## Contact

For security-related questions that are not vulnerability reports:

- Open a GitHub Discussion
- Review existing security documentation
- Check the main README.md for general contact information

## Attribution

We would like to thank the security researchers and users who responsibly disclose vulnerabilities to help make kube-changejob more secure.

---

**Last Updated:** December 2025
