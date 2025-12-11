# New Features: Captcha Verification and OAuth Access

This document describes two new security features added to Zoraxy:

1. **Captcha Verification** - Block bots and automated attacks with Google reCAPTCHA or Cloudflare Turnstile
2. **OAuth Access** - Protect endpoints with OAuth-based authentication (similar to Cloudflare Access)

---

## 1. Captcha Verification

### Overview

The captcha verification feature provides a managed captcha solution to block malicious bots from accessing your sites. It supports both Google reCAPTCHA and Cloudflare Turnstile.

### Features

- **Dual Provider Support**: Choose between Google reCAPTCHA or Cloudflare Turnstile
- **Global Protection**: Apply captcha verification across all proxied sites
- **Session Management**: Verified users receive temporary sessions (configurable duration)
- **Beautiful Challenge Page**: Modern, responsive captcha challenge page
- **Easy Configuration**: Simple API-based configuration

### API Endpoints

#### Admin Endpoints (Require Authentication)

- `GET /api/captcha/config` - Get current captcha configuration (secret key hidden)
- `POST /api/captcha/update` - Update captcha configuration
- `GET /api/captcha/session/check` - Check if a client has a valid session

#### Public Endpoints

- `POST /api/captcha/verify` - Verify a captcha token
- `GET /.zoraxy/captcha/challenge` - Captcha challenge page

### Configuration

#### Google reCAPTCHA

```json
{
  "enabled": true,
  "provider": "google_recaptcha",
  "site_key": "your-site-key",
  "secret_key": "your-secret-key",
  "threshold": 0.5,
  "expiry_time": 3600
}
```

- **threshold**: Score threshold for reCAPTCHA v3 (0.0-1.0, default: 0.5)
- **expiry_time**: Session duration in seconds (default: 3600 = 1 hour)

#### Cloudflare Turnstile

```json
{
  "enabled": true,
  "provider": "cloudflare_turnstile",
  "site_key": "your-site-key",
  "secret_key": "your-secret-key",
  "expiry_time": 3600
}
```

### How It Works

1. When captcha is enabled, all incoming requests are checked for a valid captcha session
2. If no valid session exists, the user is redirected to the captcha challenge page
3. After successfully completing the captcha, a session is created for the user's IP address
4. The user is redirected back to their original destination
5. Session is valid for the configured expiry time

### Example Usage

#### Enable Captcha with Google reCAPTCHA

```bash
curl -X POST http://localhost:8000/api/captcha/update \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": true,
    "provider": "google_recaptcha",
    "site_key": "6LeIxAcTAAAAAJcZVRqyHh71UMIEGNQ_MXjiZKhI",
    "secret_key": "6LeIxAcTAAAAAGG-vFI1TnRWxMZNFuojJ4WifJWe",
    "threshold": 0.5,
    "expiry_time": 3600
  }'
```

#### Enable Captcha with Cloudflare Turnstile

```bash
curl -X POST http://localhost:8000/api/captcha/update \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": true,
    "provider": "cloudflare_turnstile",
    "site_key": "1x00000000000000000000AA",
    "secret_key": "1x0000000000000000000000000000000AA",
    "expiry_time": 3600
  }'
```

#### Disable Captcha

```bash
curl -X POST http://localhost:8000/api/captcha/update \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": false
  }'
```

### Special Paths

The following paths are automatically excluded from captcha verification:
- `/.well-known/*` - For ACME challenges and other well-known URIs
- `/.zoraxy/*` - For internal Zoraxy endpoints
- `/api/captcha/*` - For captcha API endpoints

---

## 2. OAuth Access

### Overview

OAuth Access provides endpoint protection using OAuth 2.0 authentication, similar to Cloudflare Access. It allows you to require users to authenticate via an OAuth provider (Google, GitHub, Azure AD, etc.) before accessing protected resources.

### Features

- **Multi-Provider Support**: Configure multiple OAuth providers
- **Domain Restrictions**: Limit access by email domain (e.g., only @company.com)
- **Email Whitelist**: Allow specific email addresses
- **Email Verification**: Require verified emails
- **Session Management**: Secure cookie-based sessions
- **Per-Endpoint Protection**: Enable OAuth on specific proxy endpoints
- **PKCE Support**: Enhanced security for OAuth flows

### API Endpoints

#### Admin Endpoints (Require Authentication)

- `GET /api/oauth-access/providers/list` - List all configured OAuth providers
- `GET /api/oauth-access/providers/get?id=<provider-id>` - Get specific provider
- `POST /api/oauth-access/providers/add` - Add or update an OAuth provider
- `POST /api/oauth-access/providers/delete?id=<provider-id>` - Delete a provider
- `GET /api/oauth-access/stats` - Get usage statistics

#### Public Endpoints

- `GET /.zoraxy/oauth/login?redirect=<url>` - Initiate OAuth login
- `GET /.zoraxy/oauth/callback` - OAuth callback handler
- `GET /.zoraxy/oauth/logout?redirect=<url>` - Logout and clear session
- `GET /api/oauth-access/check` - Check authentication status

### Configuration

#### OAuth Provider Configuration

```json
{
  "id": "google",
  "name": "Google OAuth",
  "enabled": true,
  "client_id": "your-client-id.apps.googleusercontent.com",
  "client_secret": "your-client-secret",
  "auth_url": "https://accounts.google.com/o/oauth2/v2/auth",
  "token_url": "https://oauth2.googleapis.com/token",
  "userinfo_url": "https://www.googleapis.com/oauth2/v2/userinfo",
  "scopes": ["openid", "email", "profile"],
  "redirect_url": "https://your-domain.com/.zoraxy/oauth/callback",
  "allowed_domains": ["example.com", "company.com"],
  "allowed_emails": ["admin@example.com"],
  "session_duration": 3600,
  "require_email_verify": true
}
```

### Common OAuth Providers

#### Google

```json
{
  "id": "google",
  "name": "Google",
  "auth_url": "https://accounts.google.com/o/oauth2/v2/auth",
  "token_url": "https://oauth2.googleapis.com/token",
  "userinfo_url": "https://www.googleapis.com/oauth2/v2/userinfo",
  "scopes": ["openid", "email", "profile"]
}
```

#### GitHub

```json
{
  "id": "github",
  "name": "GitHub",
  "auth_url": "https://github.com/login/oauth/authorize",
  "token_url": "https://github.com/login/oauth/access_token",
  "userinfo_url": "https://api.github.com/user",
  "scopes": ["read:user", "user:email"]
}
```

#### Microsoft Azure AD

```json
{
  "id": "azure",
  "name": "Azure AD",
  "auth_url": "https://login.microsoftonline.com/{tenant}/oauth2/v2.0/authorize",
  "token_url": "https://login.microsoftonline.com/{tenant}/oauth2/v2.0/token",
  "userinfo_url": "https://graph.microsoft.com/v1.0/me",
  "scopes": ["openid", "email", "profile"]
}
```

### How It Works

1. Configure an OAuth provider through the API
2. Enable OAuth Access authentication on a proxy endpoint by setting `AuthMethod` to `4` (AuthMethodOAuthAccess)
3. When a user visits the protected endpoint without authentication:
   - They are redirected to `/.zoraxy/oauth/login`
   - Zoraxy redirects them to the OAuth provider (Google, GitHub, etc.)
   - User authenticates with the OAuth provider
   - OAuth provider redirects back to `/.zoraxy/oauth/callback` with an authorization code
   - Zoraxy exchanges the code for an access token and retrieves user information
   - User is validated against allowed domains/emails
   - A session cookie is set
   - User is redirected to their original destination
4. Subsequent requests use the session cookie for authentication

### Example Usage

#### Configure Google OAuth Provider

```bash
curl -X POST http://localhost:8000/api/oauth-access/providers/add \
  -H "Content-Type: application/json" \
  -d '{
    "id": "google",
    "name": "Google OAuth",
    "enabled": true,
    "client_id": "your-client-id.apps.googleusercontent.com",
    "client_secret": "your-client-secret",
    "auth_url": "https://accounts.google.com/o/oauth2/v2/auth",
    "token_url": "https://oauth2.googleapis.com/token",
    "userinfo_url": "https://www.googleapis.com/oauth2/v2/userinfo",
    "scopes": ["openid", "email", "profile"],
    "redirect_url": "https://your-domain.com/.zoraxy/oauth/callback",
    "allowed_domains": ["example.com"],
    "session_duration": 7200,
    "require_email_verify": true
  }'
```

#### Enable OAuth Access on a Proxy Endpoint

When creating or updating a proxy endpoint, set the authentication method to OAuth Access:

```json
{
  "authentication_provider": {
    "auth_method": 4,
    "oauth_access_enabled": true
  }
}
```

The `auth_method` values are:
- `0` - No authentication
- `1` - Basic Auth
- `2` - Forward Auth
- `3` - OAuth2 (existing SSO)
- `4` - OAuth Access (new feature)

#### Check Authentication Status

```bash
curl -b cookies.txt http://localhost:8000/api/oauth-access/check
```

Response:
```json
{
  "authenticated": true,
  "user_email": "user@example.com",
  "user_name": "John Doe",
  "expires_at": "2025-12-11T22:00:00Z"
}
```

### Security Considerations

#### Captcha
- Use HTTPS to protect captcha tokens in transit
- Choose appropriate session expiry times based on your security requirements
- For reCAPTCHA v3, adjust the threshold based on your traffic patterns
- Monitor captcha sessions to detect potential bypass attempts

#### OAuth Access
- Always use HTTPS in production to protect OAuth tokens
- Keep client secrets secure and rotate them regularly
- Use `require_email_verify: true` to ensure emails are verified by the OAuth provider
- Set appropriate `session_duration` based on your security requirements
- Use `allowed_domains` or `allowed_emails` to restrict access
- Regularly review active sessions and revoke suspicious ones

### Troubleshooting

#### Captcha

**Issue**: Captcha verification fails
- Check that site_key and secret_key are correct
- Verify the provider (google_recaptcha or cloudflare_turnstile) is set correctly
- For reCAPTCHA v3, adjust the threshold if legitimate users are being blocked

**Issue**: Sessions expire too quickly
- Increase the `expiry_time` parameter (in seconds)

#### OAuth Access

**Issue**: OAuth callback fails
- Verify the `redirect_url` matches exactly what's configured in your OAuth provider
- Check that the OAuth provider's client ID and secret are correct
- Ensure the `scopes` requested are supported by your OAuth provider

**Issue**: Users can't access after authentication
- Check `allowed_domains` and `allowed_emails` configuration
- Verify `require_email_verify` is set appropriately
- Check browser cookies are enabled

**Issue**: "Invalid state" error
- This usually means the OAuth state has expired (10-minute timeout)
- Ask the user to try the login process again

---

## Integration Example

You can combine both features for maximum security:

1. Enable global captcha to block bots
2. Enable OAuth Access on specific sensitive endpoints
3. Users must first pass the captcha check, then authenticate via OAuth

This provides a two-layer security approach:
- Layer 1 (Captcha): Prevents automated bot attacks
- Layer 2 (OAuth Access): Ensures only authorized users can access protected resources

---

## Development

### Module Structure

#### Captcha Module
```
src/mod/captcha/
├── captcha.go    - Core captcha manager
└── handlers.go   - HTTP handlers
```

#### OAuth Access Module
```
src/mod/auth/oauthaccess/
├── oauthaccess.go - Core OAuth Access manager
└── handlers.go    - HTTP handlers and middleware
```

### Database Tables

- `captcha` - Stores captcha configuration
- `oauth_access` - Stores OAuth provider configurations and sessions

### Contributing

When adding new features to these modules:

1. Captcha: Update the `CaptchaConfig` struct and add new provider types
2. OAuth Access: Add new OAuth provider templates in the documentation
3. Always maintain backward compatibility
4. Add tests for new functionality
5. Update this documentation

---

## License

These features are part of Zoraxy and follow the same AGPL-3.0 license.
