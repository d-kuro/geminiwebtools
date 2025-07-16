# Release Notes

## v0.0.1 - Initial Release (Experimental)

⚠️ **Experimental Version** - This is an experimental release and may include breaking changes in future versions. The API is not yet stable and should be used with caution in production environments.

### Features

🔍 **Web Search**
- AI-powered web search using Google's Gemini model with grounding support
- Citation processing and source attribution
- Configurable search parameters

🌐 **Web Fetch**
- Intelligent web content fetching with AI processing
- Fallback to direct HTTP when AI processing is unavailable
- Content size limits and URL validation

🔐 **OAuth2 Authentication**
- Compatible with Google OAuth2 authentication flow
- Browser-based authentication support
- Credential storage with file system backend

⚙️ **Configuration & Security**
- Flexible configuration system with sensible defaults
- Private IP protection
- Content size limits (configurable)
- Secure transport configuration

### Getting Started

```bash
go get github.com/d-kuro/geminiwebtools
```

See the [README](README.md) for usage examples and detailed documentation.

### Requirements

- Go 1.24 or later
- Google OAuth2 credentials (compatible with gemini-cli)
- Internet connection for authentication and API calls

### Breaking Changes Notice

**Important**: As this is an experimental release (v0.0.1), future versions may include breaking changes to the API. We recommend:

- Pinning to this specific version in production use
- Following release notes closely for migration guidance
- Testing thoroughly before upgrading

---

For issues and feature requests, please visit: https://github.com/d-kuro/geminiwebtools/issues