# Stockyard Fence

API key vault for teams. Store, rotate, and audit API keys in a single binary.

## What it does

Fence sits between your team and your API keys. Instead of sharing keys in `.env` files, Slack messages, or password managers, Fence stores them encrypted and lets team members access them through scoped tokens.

Every key access is logged. Keys rotate on schedule. Revocation is instant.

## Features

- **Encrypted storage** — AES-256-GCM at rest, single SQLite file
- **Scoped access** — team members get tokens that resolve to the real key at request time
- **Access logging** — every key read is recorded with who, when, and from where
- **Auto-rotation** — schedule key rotation with configurable intervals
- **Revocation** — instantly revoke a team member's access without changing the underlying key
- **Single binary** — Go + embedded SQLite, no external dependencies
- **Self-hosted** — your keys never leave your infrastructure

## Quick start

```bash
curl -fsSL https://stockyard.dev/fence/install.sh | sh
fence serve
```

## Pricing

- **Free:** 10 keys, 2 team members
- **Pro ($9/mo):** Unlimited keys, unlimited members, auto-rotation, audit export

## Part of Stockyard

Fence is a standalone product from [Stockyard](https://stockyard.dev), the self-hosted LLM infrastructure platform.

## License

BSL 1.1
