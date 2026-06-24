# Changelog

## [Unreleased] - 2026-03-28

### Added
- Mobile auth support: `Authorization: Bearer` header accepted across all auth endpoints
- Login and register responses now include tokens in JSON body (`tokens.accessToken`, `tokens.refreshToken`) for mobile clients
- `ExtractBearerToken()` utility for parsing Authorization headers
- Refresh token endpoint accepts Bearer header (mobile) with cookie fallback (web)
- Check-session endpoint accepts Bearer header and refresh token from request body

### Changed
- JWT secret reads from `JWT_SECRET` environment variable (falls back to default for dev)
- Cookie domain reads from `COOKIE_DOMAIN` environment variable (falls back to `localhost`)
- Auth middleware checks Authorization header first, then cookies

### Documentation
- Rewrote root `README.md` for Omni-Server itself, with microservices architecture, local setup, current status, and experimental/in-development framing
- Removed duplicate root markdown docs after folding core setup and system overview into `README.md`

### Security
- Removed hardcoded JWT secret — now configurable via environment
- Removed hardcoded cookie domain — now configurable via environment
