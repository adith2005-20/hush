# Hush

A lightweight, self-hosted secrets manager for developers. Think "Git for secrets."

## Features

- **Client-side encryption** - Server never sees your secrets
- **Simple setup** - No Docker, no dependencies, just one binary
- **Cross-platform** - Works on Windows, Linux, macOS
- **Self-hosted** - Your secrets, your server

## Quick Start

### 1. Server Setup (One-time)

```bash
# On your server (Homelab, VPS, etc.)
./hushd init
# Save the token that's generated

./hushd start
```

### 2. Client Setup

```bash
# On your dev machine
./hush login http://your-server:55555 <token-from-init>

# In your project directory
./hush init myproject
./hush set DATABASE_URL=postgresql://localhost/db
./hush set API_KEY=secret123
./hush pull
```

Your `.env` file is now created with your secrets!

## Commands

### Server (`hushd`)
```bash
hushd init    # Initialize server (first-time setup)
hushd start   # Start the server
```

### Client (`hush`)
```bash
hush login <server-url> <token>   # Authenticate with server
hush init <project-name>          # Initialize project
hush set KEY=value                # Add/update secret
hush list                         # List all secret keys
hush pull                         # Download secrets to .env
```

## Example Workflow

**Developer A (first time):**
```bash
cd ~/myproject
hush init myproject
hush set DB_PASSWORD=supersecret
hush set STRIPE_KEY=sk_live_123
```

**Developer B (joining the project):**
```bash
cd ~/myproject
hush login http://server:55555 <token>
hush pull
```

## Building from Source

```bash
# Clone the repo
git clone https://github.com/adith2005-20/hush
cd hush

# Build
go mod tidy
go build -o hush ./cmd/hush
go build -o hushd ./cmd/hushd
```

## How It Works

1. **Secrets are encrypted client-side** with AES-256-GCM before leaving your machine
2. **Server stores encrypted blobs** and can't read them
3. **Master key** stays on your machine in `~/.config/hush/master.key`
4. **Zero-knowledge architecture** - even if the server is compromised, secrets stay safe

## Configuration Files

- `hush.yaml` - Project config (in your project directory)
- `~/.config/hush/credentials.yaml` - Your server credentials
- `~/.config/hush/master.key` - Your encryption key (never share this!)

## Why Hush?

- **Not 1Password** - Purpose-built for developers, not consumer passwords
- **Not HashiCorp Vault** - Simpler, no complex setup, perfect for homelabs
- **Not .env files in Git** - Seriously?


## Contributing

PRs welcome! This is a hobby project built for the homelab community.

---

Built with ❤️ for self-hosters and homelab enthusiasts
