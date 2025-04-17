# yeet

A lightweight service manager that works over SSH.

## Overview

yeet is a client-side utility for catch, a simple service manager designed to deploy and manage services on remote Linux machines without requiring systemd configuration. It provides a streamlined workflow to deploy binaries and containers via SSH and manage them as services.

## Components

The system consists of two main components:

1. **yeet** - The client-side utility used to deploy and manage services
2. **catch** - The core service manager that runs on the remote server

## Installation

### Client Installation

To install the yeet client:

```bash
go install github.com/yeetrun/yeet/cmd/yeet@latest
```

### Server Setup

Initialize a new remote server:

```bash
# Install catch on a remote host
yeet init <remote>
```

## Usage

### Deploy a Service

#### Using yeet

```bash
# Push and run a Docker container
yeet push <service> <image>

# Run a local binary on the remote server
yeet run <service> <binary>
```

### Available Commands

#### yeet client commands

```bash
yeet [command]
```

Core commands:

- `init <remote>` - Install catch on a remote host
- `push <svc> <image>` - Push a container image to the remote
- `docker-host` - Print out the docker host
- `update` - Update catch on the remote server

Service management:

- `<service> start` - Start a service
- `<service> stop` - Stop a service
- `<service> restart` - Restart a service
- `<service> status` - Show service status
- `<service> logs` - View service logs
- `<service> remove` - Remove a service

Yeet supports all catch commands (env, edit, stage, etc.) and runs them against a remote catch server.

### Networking

yeet supports different networking modes that can be configured with the `--net` flag:

```bash
# Deploy with Tailscale networking
yeet run myservice --net=ts --ts-auth-key=tskey-123

# Deploy with macvlan networking
yeet run myservice --net=macvlan --macvlan-parent=eth0
```

## Alternative: Using SSH Directly

If you prefer to use SSH directly instead of the yeet client, you can use the following methods:

```bash
# Deploy by copying a binary
scp <binary> <service>@<host>:

# Deploy from stdin with arguments
cat <binary> | ssh <service>@<host> run [flags] [args...]
```

Example:

```bash
scp myservice myservice@server:
# or with arguments
cat myservice | ssh myservice@server run --restart=true --net=ts
```

### Available catch commands via SSH

SSH to your service to use these commands:

```bash
ssh <service>@<host> <command>
```

Core commands:

- `run` - Install a service with the binary received from stdin
- `start` - Start a service
- `stop` - Stop a service
- `restart` - Restart a service
- `status` - Show service status
- `logs` - View service logs
- `remove` - Remove a service

Configuration:

- `env` - Manage environment variables
- `edit` - Edit service configuration
- `stage` - Stage service configuration changes
  - `stage show` - Show staged configuration
  - `stage clear` - Clear staged configuration
  - `stage commit` - Apply staged configuration

Network options:

- `ip` - Manage IP address configuration
- `ts` - Manage Tailscale networking
- `mount` - Mount a filesystem
- `umount` - Unmount a filesystem

Service management:

- `enable` - Enable service autostart
- `disable` - Disable service autostart
- `rollback` - Rollback to previous service version
- `cron` - Manage scheduled tasks
- `events` - View service events
- `version` - Show the catch version

## Security Warning

⚠️ **Important**: Currently, all services managed by catch run as root. This presents security risks and is not recommended for production environments with untrusted services. We plan to implement proper user isolation in a future release.

## Updating

Update catch on the remote server:

```bash
# Using yeet
yeet update

# Or using SSH directly
go build ./cmd/catch && cat catch | ssh root@catch@remote update
```
