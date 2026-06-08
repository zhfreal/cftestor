# cftestor: Cloudflare CDN IP Scanner & Verifier

`cftestor` is a high-performance utility designed to find and verify the best-performing Cloudflare CDN edge nodes for your specific network environment. By performing sequential latency and throughput tests, it identifies IPs with the lowest latency and highest download speeds, bypassing ISP throttling and regional routing inefficiencies.

## How It Works

`cftestor` uses a two-stage evaluation pipeline to filter candidate IPs:

1.  **Delay Test (DT)**:
    *   Performs a secure connection test via **SSL/TLS handshake** or **HTTPS GET**.
    *   Measures the true "Time to Secure Connection," validating both routing speed and the node's availability for public traffic.
    *   Supports multiple attempts per IP to calculate stability (pass rate) and standard deviation.

2.  **Download Test (DLT)**:
    *   For IPs that pass the DT stage, `cftestor` performs a real-world throughput test.
    *   Downloads a sample file to calculate the actual average speed in KB/s.
    *   Supports multiple parallel workers to maximize scanning efficiency.

IPs that pass both stages are considered "qualified" and are reported to the user or stored for future use.

## Features

*   **Diverse Sampling**: Fairly samples from multiple CIDR ranges and individual IP lists simultaneously.
*   **TLS Fingerprinting**: Simulates various browser fingerprints (Chrome, Firefox, Edge, Safari) to bypass SNI-based filtering.
*   **Real-time Reporting**: Live result updates via terminal as each test completes.
*   **Reliable Timeouts**: Strict context-based timeout enforcement to prevent worker starvation on slow nodes.
*   **Multiple Output Formats**: Save results to CSV files or SQLite3 databases for automated processing.
*   **Cache Control**: Optional `-C`, `--no-cache` flag to bypass CDN/proxy caching for custom test URLs (automatically disabled for default URLs to protect origin bandwidth).

## Quick Start

### Build from Source

Ensure you have [Go](https://go.dev/) installed:

```bash
git clone https://github.com/zhfreal/cftestor.git
cd cftestor
go build -o cftestor .
```

### Basic Usage

Run with default settings (scans built-in Cloudflare IPv4 ranges):

```bash
./cftestor
```

### Advanced Usage Examples

**Find 20 IPs with latency < 200ms and speed > 10MB/s:**
```bash
./cftestor -k 200 -l 10000 -r 20
```

**Perform Delay Test only (fast discovery):**
```bash
./cftestor --dt-only -r 50
```

**Scan specific CIDR ranges and save to database:**
```bash
./cftestor -s 104.16.0.0/16 -s 172.64.0.0/13 -s example.com:443 --to-db -f results.db
```

**Continuous monitoring (loop every 5 minutes):**
```bash
./cftestor --loop 3 --loop-interval 300
```

## CLI Reference

```text
Usage: cftestor [options]

Core Options:
    -s, --ip           strings    Specify IP, CIDR, or host:port, including DNS names. Can be used multiple times.
    -i, --in           string     Path to a file containing IPs, CIDRs, or host:port entries (one per line).
    -p, --port         strings    Port(s) to test for IP/CIDR inputs (e.g., "443", "80-443"). Default: 443.
    -a, --test-all                Test all provided IPs until none remain.
    -r, --result       int        Maximum number of qualified IPs to find. Default: 10.
        --fast                    Use a limited set of internal IPs for quick scanning.

Delay Test (DT) Options:
    -m, --dt-thread    int        Number of concurrent DT threads. Default: 20.
    -t, --dt-timeout   int        Timeout per DT (ms).
    -c, --dt-count     int        Number of DT attempts per IP. Default: 4.
        --dt-via       string     Protocol: "https", "tls", or "ssl". Default: https.
        --ev-dt                   Enable full evaluation using all attempts.
    -k, --ev-dt-delay  int        Maximum allowed average delay (ms). Default: 600.

Download Test (DLT) Options:
    -n, --dlt-thread   int        Number of concurrent DLT threads. Default: 1.
    -d, --dlt-period   int        Maximum duration per DLT (seconds). Default: 10.
    -l, --speed        float      Minimum required speed (KB/s). Default: 6000.
    -C, --no-cache                Bypass CDN/Proxy caching.

Output Options:
    -w, --to-file                 Save results to CSV.
    -e, --to-db                   Save results to SQLite3.
    -V, --debug                   Print detailed debug logs.
```

## Database Schema (Table: `CFTD`)

The SQLite3 output includes comprehensive metrics for every qualified IP:
*   `TestTime`: Timestamp of the verification.
*   `IP`: The verified edge node address.
*   `DA`: Average Delay (ms).
*   `DLS`: Average Download Speed (KB/s).
*   `LOC`: Geographic location (Colo code).
*   `DTPR`: Delay Test pass rate (stability).

---

## Acknowledgments

This project is inspired by and built upon work from:
*   [CloudflareScanner](https://github.com/Spedoske/CloudflareScanner)
*   [CloudflareSpeedTest](https://github.com/XIU2/CloudflareSpeedTest)
*   [utls](https://github.com/refraction-networking/utls)
