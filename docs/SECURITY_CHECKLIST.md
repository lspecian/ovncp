# Security Checklist for Developers

## Pre-Development

- [ ] Review security requirements for the feature
- [ ] Identify sensitive data that will be handled
- [ ] Plan authentication and authorization requirements
- [ ] Review relevant OWASP guidelines
- [ ] Check for existing security patterns in codebase

## During Development

### Input Validation
- [ ] Validate all user inputs on the server side
- [ ] Use allowlists rather than denylists for validation
- [ ] Validate data types, lengths, formats, and ranges
- [ ] Sanitize inputs that will be displayed to users
- [ ] Reject requests with unexpected parameters

### Authentication & Authorization
- [ ] Verify authentication for all protected endpoints
- [ ] Implement proper authorization checks
- [ ] Use existing auth middleware rather than custom implementations
- [ ] Check permissions at the resource level
- [ ] Implement rate limiting for authentication endpoints

### Data Protection
- [ ] Never log sensitive information (passwords, tokens, PII)
- [ ] Use parameterized queries for all database operations
- [ ] Encrypt sensitive data before storing
- [ ] Use secure random generators for tokens/IDs
- [ ] Implement proper session management

### API Security
- [ ] Add rate limiting to new endpoints
- [ ] Implement request size limits
- [ ] Validate Content-Type headers
- [ ] Return consistent error messages (don't leak information)
- [ ] Add appropriate security headers

### Error Handling
- [ ] Catch and handle all errors appropriately
- [ ] Log errors with context but without sensitive data
- [ ] Return generic error messages to clients
- [ ] Ensure stack traces are not exposed
- [ ] Implement proper timeout handling

### Third-Party Dependencies
- [ ] Review security of new dependencies
- [ ] Check for known vulnerabilities
- [ ] Verify dependency licenses
- [ ] Use official/verified packages only
- [ ] Keep dependencies up to date

## Testing

### Security Testing
- [ ] Write unit tests for auth/authz logic
- [ ] Test input validation with malicious inputs
- [ ] Verify rate limiting works correctly
- [ ] Test error handling scenarios
- [ ] Check for information disclosure in responses

### Common Vulnerability Tests
- [ ] Test for SQL injection
- [ ] Test for XSS vulnerabilities
- [ ] Test for CSRF (if applicable)
- [ ] Test for authentication bypass
- [ ] Test for authorization flaws
- [ ] Test for sensitive data exposure

## Code Review

### Security Review Checklist
- [ ] No hardcoded secrets or credentials
- [ ] Proper input validation implemented
- [ ] Authentication and authorization verified
- [ ] Sensitive data properly protected
- [ ] Error handling doesn't leak information
- [ ] Logging doesn't include sensitive data
- [ ] Security headers properly set
- [ ] Rate limiting applied where needed
- [ ] Database queries are parameterized
- [ ] External calls have timeouts

## Pre-Deployment

### Configuration
- [ ] Verify all secrets are in environment variables
- [ ] Ensure debug mode is disabled
- [ ] Check that test endpoints are removed
- [ ] Verify TLS/HTTPS is enforced
- [ ] Confirm security headers are enabled

### Documentation
- [ ] Document any new security considerations
- [ ] Update API documentation with auth requirements
- [ ] Document rate limits for new endpoints
- [ ] Add security notes to README if needed
- [ ] Update threat model if architecture changed

## Post-Deployment

### Monitoring
- [ ] Verify security alerts are working
- [ ] Check that audit logging is functioning
- [ ] Monitor for unusual activity patterns
- [ ] Review rate limit metrics
- [ ] Ensure error rates are normal

### Verification
- [ ] Run security scanner against deployment
- [ ] Verify HTTPS/TLS is working correctly
- [ ] Test authentication flows in production
- [ ] Confirm security headers are present
- [ ] Check that sensitive endpoints are protected

## Incident Response

If a security issue is discovered:

1. [ ] Do not attempt to fix it alone
2. [ ] Report to security team immediately
3. [ ] Document all details about the issue
4. [ ] Preserve any relevant logs
5. [ ] Follow incident response procedures
6. [ ] Participate in post-mortem

## Regular Maintenance

### Weekly
- [ ] Review security alerts and logs
- [ ] Check for new dependency vulnerabilities
- [ ] Monitor rate limit effectiveness

### Monthly
- [ ] Review and update security policies
- [ ] Audit user permissions
- [ ] Review audit logs for anomalies
- [ ] Update security dependencies

### Quarterly
- [ ] Conduct security training
- [ ] Review and update threat model
- [ ] Perform penetration testing
- [ ] Review security metrics

## Security Resources

### Internal Resources
- Security Team: security@ovncp.example.com
- Security Wiki: https://wiki.ovncp.example.com/security
- Security Slack: #security

### External Resources
- [OWASP Cheat Sheets](https://cheatsheetseries.owasp.org/)
- [Go Security](https://golang.org/doc/security)
- [NIST Guidelines](https://www.nist.gov/cybersecurity)

## Quick Security Fixes

### If you find hardcoded secrets:
```bash
# Remove from code
# Rotate the secret immediately
# Add to .env file
# Update deployment configs
```

### If you find SQL injection vulnerability:
```go
// Bad
query := fmt.Sprintf("SELECT * FROM users WHERE id = %s", userID)

// Good
query := "SELECT * FROM users WHERE id = $1"
db.Query(query, userID)
```

### If you find XSS vulnerability:
```javascript
// Bad
element.innerHTML = userInput;

// Good
element.textContent = userInput;
// or use a sanitization library
```

### If you find missing authentication:
```go
// Add auth middleware
router.GET("/api/sensitive", AuthRequired(), handler)
```

### If you find missing rate limiting:
```go
// Add rate limit middleware
router.POST("/api/endpoint", RateLimit(100, 60), handler)
```

## Remember

- **Security is everyone's responsibility**
- **When in doubt, ask the security team**
- **It's better to be paranoid than compromised**
- **Security bugs have highest priority**
- **Never bypass security controls, even for testing**