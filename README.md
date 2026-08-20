![netprobe](assets/netprobe.png)

# netprobe

**netprobe** is a specialized, dual-stack network monitoring and firewall validation tool built for the [Evoke demoparty](https://www.evoke.eu).

It is designed to run on specific network interfaces to validate route isolation (e.g., ensuring guest networks cannot access organizer infrastructure) and service availability across both **IPv4** and **IPv6** simultaneously.

---

## 💡 The Core Concept: Segment-Driven Testing

Unlike traditional monitoring tools that run centrally and check if a server is generally "up", `netprobe` is designed to test the network from the perspective of **specific network segments (VLANs)**.

The best practice is to create **one configuration file per network segment** and run a dedicated instance of `netprobe` bound directly to that segment's network interface.

**Example Scenario:**
You want to ensure that your internal API is reachable from the Organizer network, but strictly blocked from the Guest network.

1. You create `config/orga-net.yaml` (expecting the API to be UP).
2. You create `config/guest-net.yaml` (expecting the API to be ISOLATED).
3. You run two instances of `netprobe` bound to their respective interfaces:

```bash
# Terminal 1: Testing from the Organizer VLAN
./netprobe --interface vlan10 --config config/orga-net.yaml

# Terminal 2: Testing from the Guest VLAN
./netprobe --interface vlan20 --config config/guest-net.yaml
```

---

## ✨ Features

- **True Dual-Stack Testing:** Automatically tests both IPv4 and IPv6 routes if the target domain supports them.
- **Interface Binding:** Binds strictly to a specified network interface (e.g., `vlan20` or `eth1`) to test exact network segments.
- **Security Validation:** Explicitly checks for isolation (`expectUp: false`). It raises a security alert if a service is reachable when it should be blocked by the firewall.
- **Smart Protocol Checks:**
  - **HTTP:** Validates status codes and proper HTTPS redirects.
  - **HTTPS:** Validates TLS handshakes and expected certificate validity (useful for internal self-signed certs).
  - **WebSockets:** Validates successful protocol upgrades through proxies (`101 Switching Protocols`).
- **12-Factor App Ready:** Configurable via CLI flags, Environment Variables, and `.env` files.

---

## 🚀 Usage

Run the compiled binary by providing the required flags for the network interface and the configuration file.

    ./netprobe --interface en0 --config config/probes.yaml --log-level debug

### CLI Parameters

| Flag / Parameter | Short | Default      | Description                                                             |
| :--------------- | :---: | :----------- | :---------------------------------------------------------------------- |
| `--interface`    | `-i`  | _(required)_ | The network interface to bind to (e.g., `eth0`, `wlan0`, `en0`).        |
| `--config`       | `-c`  | _(required)_ | Path to the YAML configuration file defining the targets.               |
| `--ip-family`    |       | `auto`       | IP family to use for checks. Options: `4`, `6`, or `auto` (dual-stack). |
| `--log-level`    |       | `info`       | Logging verbosity. Options: `debug`, `info`, `warn`, `error`.           |
| `--version`      | `-v`  |              | Prints the current build version and exits.                             |

---

## ⚙️ Environment Variables & `.env`

`netprobe` fully supports environment variables, which is especially useful for settings used in multiple environments (e.g. logging or ip_family).

The application automatically loads a `.env` file in the working directory if it exists.

### Parameter Overrides

Every CLI flag can be overridden using environment variables prefixed with `NETPROBE_`:

```env
# .env example
NETPROBE_INTERFACE=vlan50
NETPROBE_CONFIG=/app/config/party-net.yaml
NETPROBE_LOG_LEVEL=debug
NETPROBE_IP_FAMILY=auto
NETPROBE_OTEL_ENDPOINT=localhost:3317
```

---

## 📄 Configuration (`config.yaml`)

The probe configuration is defined in a YAML file. It describes the targets, the intervals, and the specific checks for **one specific network segment**.

For a quick start, check out the example configurations provided in the **[config/](config/) ** directory of this repository. You can copy an example and adjust it to your needs.

Because most check parameters have sensible defaults, you only need to specify what deviates from the standard behavior.

### Example Configuration

```yaml
name: "Evoke 2026 Core Infrastructure"
defaults:
  interval_seconds: 30
  timeout_seconds: 3

probes:
  # ---------------------------------------------------------
  # Target 1: Public Website (Expected to be fully reachable)
  # ---------------------------------------------------------
  - id: "public-website"
    hostname: "www.evoke.example.com"
    http:
      url: "http://www.evoke.example.com"
      expect_up: true
      expect_redirect_to_httpsHTTPS: true

    https:
      url: "https://www.evoke.example.com"
      expect_up: true
      expect_valid_cert: true

  # ---------------------------------------------------------
  # Target 2: Internal API (Expected to be BLOCKED / ISOLATED)
  # ---------------------------------------------------------
  - id: "internal-api"
    hostname: "api.party.evoke.example.com"
    https:
      url: "https://api.party.evoke.example.com"
      expect_up: false # Triggers a Security Alert if reachable!
      expect_valid_cert: false # Expected down, but if it leaks, cert might be self-signed.

  # ---------------------------------------------------------
  # Target 3: Live Stream WebSocket (Checking proxy upgrades)
  # ---------------------------------------------------------
  - id: "stream-socket"
    hostname: "stream.evoke.example.com"
    websocket:
      url: "wss://stream.evoke.example.com/ws"
      # WebSockets implicitly expect a successful connection and protocol upgrade.
```

### Check Types & Attributes

All check attributes are completely optional and will fall back to sensible defaults if omitted.

- **`http`**:
  - `expect_up` (boolean, default: `false`)
  - `expect_redirect_to_https` (boolean, default: `false`)
  - `url` (string, default: `http://<hostname>`)
- **`https`**:
  - `expect_up` (boolean, default: `false`)
  - `expect_valid_cert` (boolean, default: `false`)
  - `url` (string, default: `https://<hostname>`)
- **`websocket`**:
  - `url` (string, default: `wss://<hostname>/ws`)
  - _Note: WebSockets implicitly focus entirely on ensuring the proxy correctly handles the `Upgrade: websocket` headers._

## 📜 License & Acknowledgments

This project is made possible by several fantastic open-source libraries. For a complete list of third-party packages and their respective licenses, please refer to the [THIRD-PARTY-NOTICES.md](THIRD-PARTY-NOTICES.md) file.
