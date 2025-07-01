# Security Policy

## Supported Versions

OpenCode is currently in early development. We provide security updates for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 0.0.x   | :white_check_mark: |
| < 0.0   | :x:                |

**Note:** As this project is in early development, features may change, break, or be incomplete. Security patches will be applied to the latest version in the 0.0.x series.

## Reporting a Vulnerability

We take the security of OpenCode seriously. If you have discovered a security vulnerability in our project, we appreciate your help in disclosing it to us in a responsible manner.

### How to Report

1. **DO NOT** open a public issue on GitHub for security vulnerabilities
2. **Preferred Method**: Report security vulnerabilities using GitHub's private security advisory:
   - Go to the [Security tab](https://github.com/opencode-ai/opencode/security/advisories)
   - Click on "Report a vulnerability"
   - Fill out the form with detailed information
3. **Alternative Method**: Contact the maintainers directly through GitHub:
   - Primary maintainer: [@kujtimiihoxha](https://github.com/kujtimiihoxha)
   - You can also reach out via GitHub Discussions marked as private

### What to Include

Please provide the following information in your report:

- **Description**: A clear description of the vulnerability
- **Impact**: The potential impact of the vulnerability (e.g., code execution, data exposure, privilege escalation)
- **Affected Components**: Which parts of OpenCode are affected (e.g., LSP integration, MCP servers, tool execution)
- **Steps to Reproduce**: Detailed steps to reproduce the vulnerability
- **Proof of Concept**: If possible, include a minimal proof of concept
- **Suggested Fix**: If you have ideas on how to fix the issue, please include them

### Response Timeline

- **Initial Response**: Within 48 hours
- **Assessment**: Within 5 business days
- **Resolution Timeline**: Depending on severity:
  - Critical: Within 7 days
  - High: Within 14 days
  - Medium: Within 30 days
  - Low: Within 60 days

### What to Expect

1. **Acknowledgment**: We will acknowledge receipt of your vulnerability report
2. **Assessment**: We will investigate and validate the reported vulnerability
3. **Communication**: We will keep you informed about the progress
4. **Fix Development**: We will develop and test a fix
5. **Disclosure**: We will coordinate the disclosure timeline with you
6. **Credit**: With your permission, we will acknowledge your contribution in the security advisory

## Security Considerations for OpenCode

Given OpenCode's functionality, please pay special attention to:

### 1. Tool Execution Security
- The `bash` tool can execute arbitrary shell commands
- File system operations through `write`, `edit`, and `patch` tools
- External tool integration via MCP servers

### 2. API Key Security
- API keys for various AI providers (OpenAI, Anthropic, Google, etc.)
- AWS credentials for Bedrock integration
- Azure credentials for Azure OpenAI

### 3. Configuration Security
- Configuration files may contain sensitive information
- Environment variables containing API keys and credentials

### 4. LSP and MCP Server Security
- External process execution for language servers
- Communication with MCP servers (stdio and SSE)
- Potential for arbitrary code execution through these integrations

### 5. Data Storage Security
- SQLite database storing conversation history
- Local file system access for session management
- Potential exposure of sensitive project information

## Security Best Practices for Users

1. **API Key Management**:
   - Never commit API keys to version control
   - Use environment variables for sensitive credentials
   - Rotate API keys regularly

2. **Permission Management**:
   - Carefully review permission requests from the AI assistant
   - Be cautious with session-wide permissions (`Allow for session`)
   - Deny permissions for sensitive operations when uncertain

3. **Configuration Security**:
   - Protect your `.opencode.json` configuration files
   - Avoid storing sensitive data in configuration files
   - Use appropriate file permissions

4. **Tool Usage**:
   - Be aware that the AI can execute shell commands
   - Review commands before allowing execution
   - Limit access to production systems

5. **MCP Server Security**:
   - Only use trusted MCP servers
   - Verify the source and integrity of MCP server executables
   - Monitor MCP server communications

## Scope

The following are within scope for our security policy:

- The OpenCode CLI application
- Built-in tools and their implementations
- Configuration management
- Database operations
- LSP client implementation
- MCP protocol implementation
- File system operations
- External tool integrations

The following are **out of scope**:

- Third-party AI provider APIs (OpenAI, Anthropic, etc.)
- External MCP servers not distributed with OpenCode
- User-installed language servers
- Operating system security
- Network security beyond OpenCode's control

## Security Features

OpenCode implements several security features:

1. **Permission System**: All potentially dangerous operations require explicit user permission
2. **Session Isolation**: Each session is isolated with its own context
3. **Input Validation**: User inputs and AI responses are validated
4. **Secure Storage**: Local database encryption (planned feature)
5. **Audit Logging**: All tool executions are logged

## Contact

For any security-related questions or concerns:

- **Primary Maintainer**: [@kujtimiihoxha](https://github.com/kujtimiihoxha)
- **Security Advisories**: [GitHub Security Tab](https://github.com/opencode-ai/opencode/security/advisories)
- **General Discussion**: Use GitHub Discussions for non-sensitive security questions

For urgent security matters, you may also:
- Open a draft PR with a fix (mark it clearly as security-related)
- Contact maintainers through their GitHub profiles

## Acknowledgments

We would like to thank the following individuals for responsibly disclosing security issues:

- *Your name could be here!*

---

**Last Updated**: June 2025  
**Version**: 1.0
