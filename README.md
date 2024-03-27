# Wasmlisher

Wasmlisher is a robust service designed to interact with NATS streaming systems, leveraging WebAssembly (Wasm) modules for stream processing. It subscribes to specified streams, dynamically loads Wasm files to process the stream data, and then publishes the formatted streams back. This tool is perfect for environments where data streams need to be efficiently processed and transformed in real-time.

## Features

- **Dynamic Stream Subscription**: Subscribe to various streams as specified in the configuration.
- **WebAssembly Integration**: Utilize Wasm modules for the flexible and powerful processing of stream data.
- **Automatic Configuration Reloads**: Automatically reloads its configuration at a specified interval, allowing for dynamic adjustments without service restart.

## Getting Started

### Prerequisites

- Go 1.21
- Access to a NATS server.
- WebAssembly (Wasm) modules for processing data streams.

### Installation

Clone the repository and build the Wasmlisher:

```bash
git@gitlab.com:syntropynet/amberdm/publisher/wasmlisher.git
cd wasmlisher
go mod tidy
make build
```

### Configuration

Wasmlisher is configured through command-line flags, environment variables, and a configuration file. The primary settings include NATS connection details, WebAssembly modules for processing, and stream configurations.

A typical config from file or endpoint output might look like this:

```json
[
  {
    "input": "syntropy.osmosis.tx",
    "output": "wasmlisher.osmosis.swap",
    "file": "/home/wasmslisher/wasm/tx.wasm",
    "type": "filesystem"
  },
  {
    "input": "syntropy.osmosis.block",
    "output": "wasmlisher.osmosis.block",
    "file": "/home/wasmslisher/wasm/block.wasm",
    "type": "filesystem"
  }
]
```

### Running Wasmlisher

To start Wasmlisher, use the following command template:

```bash
./wasmlisher -K <NATS_SUB_NKEY> -W <NATS_SUB_JWT> -N <NATS_PUB_URL> -w <NATS_PUB_JWT> -k <NATS_PUB_NKEY> -n <NATS_SUB_URL> --name "wasmlisher" --config "/path/to/conf.json" --cfInterval <CONFIG_RELOAD_INTERVAL> start
```

Replace the placeholders (e.g., `<NATS_SUB_NKEY>`, `<NATS_SUB_JWT>`, `/path/to/conf.json`) with your actual NATS credentials, configuration file path, and other relevant details.

### Example

```bash
./wasmlisher -K SUAMEQ43VTGBXZAUU7VSATP3LILQTAKET6XCSWJIYRIFZJ4RGYBLIGZXX -W exampleXAiOiJKV1QiLCJhbGciOiJlZDI1NTE5LW5rZXkifQ.eyJqdGkiOiJKUkJEV0hISEUzUE9PSTdNVVpQNlVJQ0NGTTZJQ1JRM0NGSVNUWFY1QUdXNjVPMjdJSkdRIiwiaWF0IjoxNzExNTI5NjgzLCJpc3MiOiJBRDVHUENaVVFLRVhaTlNMTEZaUklDVjIySE1QQlhCQ0NFV0c3TEdZQkRPRTJWN1ZBMlBBWjQzVyIsInN1YiI6IlVETFVWR0hFSVRRWEk1NkE3TFpNR0lDWVhUQVlGSVdZRTNYUEE0SFRWVk1IVUFaTVhJR1VOUUNGIiwibmF0cyI6eyJwdWIiOnt9LCJzdWIiOnt9LCJzdWJzIjotMSwiZGF0YSI6LTEsInBheWxvYWQiOi0xLCJ0eXBlIjoidXNlciIsInZlcnNpb24iOjJ9fQ.gLMxfYahCMX7wNwQrKm1rkhO4z2hMysEqm-hJjnyGBAb1LlUMFNfPQ_HfQAv0GUEkR9e8urlcJfwohHw2ZBkCA -N nats://europe-west3-gcp-dal-devnet-brokernode-cluster01.syntropynet.com -w exampleAiOiJKV1QiLCJhbGciOiJlZDI1NTE5LW5rZXkifQ.eyJqdGkiOiJVU1oyQkZJRk9PRjRFSlFXSjJTSVU --name "wasmlisher" --config "/path/config.json" --cfInterval 30 start
```