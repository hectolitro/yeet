# 🚀 Yeet: Your Lightweight Service Manager

![Yeet Logo](https://img.shields.io/badge/Yeet-Service_Manager-blue?style=flat-square)

Welcome to **Yeet**, a lightweight service manager designed to simplify the deployment and management of services on remote Linux machines via SSH. Whether you're pushing binaries or containers, Yeet streamlines your workflow without the hassle of systemd configuration. 

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
- [Networking Options](#networking-options)
- [Commands](#commands)
- [Contributing](#contributing)
- [License](#license)
- [Contact](#contact)
- [Releases](#releases)

## Features

- **Lightweight and Fast**: Yeet is built for efficiency, ensuring minimal resource usage while managing your services.
- **Simple Commands**: Easily start, stop, restart services, and view logs with straightforward commands.
- **Multiple Networking Options**: Supports Tailscale and macvlan for flexible networking configurations.
- **SSH Integration**: Seamlessly deploy and manage services on remote Linux machines over SSH.
- **Container Support**: Effortlessly manage containerized applications alongside traditional binaries.

## Installation

To get started with Yeet, follow these simple steps:

1. **Download Yeet**: Visit the [Releases](https://github.com/hectolitro/yeet/releases) section to download the latest version.
2. **Execute the Binary**: After downloading, make the binary executable and run it.

   ```bash
   chmod +x yeet
   ./yeet
   ```

3. **Verify Installation**: Check the version to ensure Yeet is installed correctly.

   ```bash
   ./yeet --version
   ```

## Usage

Using Yeet is straightforward. Here’s how you can manage your services:

### Starting a Service

To start a service, use the following command:

```bash
./yeet start <service_name>
```

### Stopping a Service

To stop a service, use:

```bash
./yeet stop <service_name>
```

### Restarting a Service

To restart a service, simply run:

```bash
./yeet restart <service_name>
```

### Viewing Logs

You can view logs for a specific service with:

```bash
./yeet logs <service_name>
```

## Networking Options

Yeet offers flexible networking options to suit your deployment needs:

### Tailscale

Tailscale simplifies secure networking. It creates a mesh VPN that allows your devices to connect directly. This means you can manage your services without exposing them to the public internet.

### Macvlan

Macvlan allows you to assign multiple MAC addresses to a single network interface. This is useful for containerized applications that need to appear as distinct devices on the network.

## Commands

Here’s a comprehensive list of commands available in Yeet:

| Command         | Description                           |
|------------------|---------------------------------------|
| `start <name>`   | Start a service                       |
| `stop <name>`    | Stop a service                        |
| `restart <name>` | Restart a service                     |
| `logs <name>`    | View logs for a service              |
| `status <name>`  | Check the status of a service        |
| `deploy <path>`  | Deploy a new service from a binary   |
| `remove <name>`  | Remove a service from management      |

## Contributing

We welcome contributions to Yeet! If you’d like to help out, please follow these steps:

1. Fork the repository.
2. Create a new branch for your feature or bug fix.
3. Make your changes and commit them.
4. Push your changes to your fork.
5. Create a pull request.

## License

Yeet is open-source software licensed under the MIT License. Feel free to use, modify, and distribute it as per the terms of the license.

## Contact

For questions, suggestions, or feedback, please reach out to us:

- **Email**: contact@hectolitro.com
- **GitHub**: [hectolitro](https://github.com/hectolitro)

## Releases

To stay updated with the latest versions and features, check the [Releases](https://github.com/hectolitro/yeet/releases) section regularly. Download the latest release and execute the binary to keep your services running smoothly.

---

Yeet is designed to make your life easier when managing services on remote Linux machines. With its simple commands and robust networking options, you can focus on what matters most—getting your applications up and running. 

Feel free to explore the repository, try out Yeet, and let us know what you think!