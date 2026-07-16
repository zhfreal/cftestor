# cftestor: Cloudflare CDN IP Scanner & Verifier

`cftestor` finds Cloudflare CDN edge candidates that work well from your network by testing connection latency, stability, and download throughput. It can scan built-in Cloudflare ranges, user-provided IP/CIDR sources, or explicit `host:port` targets, then print results or save them to CSV/SQLite.

## How It Works

`cftestor` uses a two-stage test pipeline:

1. **Delay Test (DT)**
   - Tests candidate reachability and latency with HTTPS, TLS, or SSL.
   - Runs one or more attempts per candidate.
   - Can evaluate average delay, pass rate, and standard deviation.

2. **Download Test (DLT)**
   - Runs after DT for candidates that passed DT, unless `--dlt-only` is used.
   - Downloads a sample file and calculates average speed in KB/s.
   - Can run multiple attempts and concurrent workers.

A candidate is qualified when it passes every enabled stage. `--dt-only` reports candidates that pass DT. `--dlt-only` reports candidates that pass DLT.

## Features

- Built-in Cloudflare IPv4/IPv6 source ranges, plus custom IP, CIDR, and `host:port` input.
- DNS `host:port` targets such as `example.com:443`.
- TLS fingerprint options for Chrome, Firefox, Edge, and Safari.
- Live terminal reporting and optional compact silence mode.
- Loop retesting for candidates that already qualified.
- CSV and SQLite3 output for later processing.
- Optional `-C, --no-cache` for custom test URLs.

## Quick Start

### Build from Source

```bash
git clone https://github.com/zhfreal/cftestor.git
cd cftestor
go build -o cftestor .
```

### Basic Usage

```bash
./cftestor
```

### Examples

Find 20 final qualified candidates with average DT delay <= 200 ms and DLT speed >= 10 MB/s:

```bash
./cftestor -k 200 -l 10000 -r 20
```

Run Delay Test only for fast discovery:

```bash
./cftestor --dt-only -r 50
```

Scan custom CIDRs and a DNS host target, then save to SQLite:

```bash
./cftestor -s 104.16.0.0/16 -s 172.64.0.0/13 -s example.com:443 --to-db -f results.db
```

Retest qualified candidates for three confirmation cycles, refilling from the original source pool if fewer than `-r` remain:

```bash
./cftestor --loop 3 --loop-interval 300 -r 10
```

Save results to CSV:

```bash
./cftestor --to-file -o results.csv
```

## Loop and Result Behavior

`-r, --result` is the target number of final qualified results. `--loop` does not simply repeat the whole scan from scratch. It first retests candidates that already qualified, which is useful for confirming that results still pass over a larger time scale.

If loop retesting removes too many candidates, `cftestor` continues scanning from the original source pool to find replacement candidates. If `--supplement` is enabled, and the initial source pool is exhausted before the target result count (`--result`) is met, the tool will automatically fall back to load candidates from broader pools in the following order:
- **If user-provided target IPs (`-s` or `-i`) were used**: Falls back to the built-in `--fast` ranges, and then to the built-in `full` lists.
- **If `--fast` ranges were used**: Falls back to the built-in `full` lists.
- **If neither was used**: Only scans the default `full` lists (no further fallbacks).

The run stops when it reaches `--result`, exhausts all available (including supplemented) source pools, or reaches `--test-timeout`.

`--test-all` overrides the result target and keeps testing until no candidates remain.

## CLI Reference

```text
Usage: cftestor [options]

Core Options:
    -s, --ip           strings    Specify IP, CIDR, or host:port. Examples: "-s 1.0.0.1", "-s 1.0.0.1/24",
                                  "-s 1.1.1.1:2053", "-s example.com:443". Can be provided multiple times.
    -i, --in           string     Path to a file containing IPs, CIDRs, or host:port entries (one per line).
    -p, --port         strings    Specify port(s) to test for IP/CIDR inputs. Supports single ports, ranges, and lists
                                  (e.g., "443", "80-443", "443,8443"). Default: 443.
    -a, --test-all                Test all provided IPs until none remain. Default: off.
    -r, --result       int        Target number of final qualified results. Default: 10.
        --fast                    Use a limited set of internal Cloudflare IPs for quick scanning. If no target IPs are provided, dynamically fetches active CIDRs.
    -4, --ipv4                    Test IPv4 only. Default: on (if no IPs specified).
    -6, --ipv6                    Test IPv6 only. Default: off. DNS hosts are resolved by the dialer.
        --fetch-ipv4   string     Fetch active Cloudflare IPv4 CIDRs dynamically, save to file, and exit.
        --fetch-ipv6   string     Fetch active Cloudflare IPv6 CIDRs dynamically, save to file, and exit.
        --fetch-cf-domains string Fetch, verify, and save top domains using Cloudflare CDN to a file, and exit.
        --dns          string     Custom DNS server for dynamic fetching (e.g. 1.1.1.1:53, tls://1.1.1.1, https://1.1.1.1/dns-query).
    -C, --no-cache                Bypass CDN/Proxy caching for custom URLs (ignored for defaults).

Network Options:
        --mark        string      Set Linux socket fwmark for outbound packets. Supports decimal and hex.
        --xmark       string      Alias for --mark.
        --interface   string      Bind outbound packets to an interface name, interface index, or local source IP.

Delay Test (DT) Options:
    -m, --dt-thread    int        Number of concurrent DT workers. Default: 20.
    -t, --dt-timeout   int        Timeout for a single DT attempt in ms. Default: 2000 (TLS/SSL) or 5000 (HTTPS).
    -c, --dt-count     int        Number of DT attempts per candidate. Default: 4.
        --dt-via       string     DT protocol: "https", "tls", or "ssl". Default: https.
        --dt-via-https            Deprecated alias for --dt-via https.
        --dt-url       string     URL to use for HTTPS-based DT. Default: https://speed.cloudflare.com/__down?bytes=0
        --hostname     string     SNI hostname for TLS/SSL DT. Default: speed.cloudflare.com
        --dt-expect-code int      Expected HTTP status code for DT. Default: 200.
        --ev-dt                   Enable DT evaluation using all attempts. Default: off.
    -k, --ev-dt-delay  int        Maximum allowed average DT delay in ms. Default: 600.
        --ev-dt-dtpr   float      Minimum required DT pass rate percentage. Default: 100.0.
        --ev-dt-std    float      Maximum allowed DT standard deviation. Default: 30.0 (if enabled).

Download Test (DLT) Options:
    -n, --dlt-thread   int        Number of concurrent DLT workers. Default: 1.
    -d, --dlt-period   int        Maximum duration for one DLT attempt in seconds. Default: 10.
    -b, --dlt-count    int        Number of DLT attempts per candidate. Default: 1.
    -u, --dlt-url      string     URL to use for DLT. Default: https://speed.cloudflare.com/__down?bytes=99999999
        --dlt-timeout  int        HTTP response timeout for DLT in ms. Default: 5000.
    -l, --speed        float      Minimum required download speed in KB/s. Default: 6000.
    -I, --interval     int        Interval between test attempts in ms. Default: 500.

Mode Options:
        --dt-only                 Perform Delay Test only.
        --disable-download        Deprecated alias for --dt-only.
        --dlt-only                Perform Download Test only.
        --loop         int        Retest qualified candidates for N confirmation cycles; refill from the original pool
                                  if fewer than --result remain.
        --loop-interval int       Seconds to wait between loop cycles. Default: 60.
        --test-timeout int        Total test timeout in minutes. Default: 30.
        --supplement              Enable IP source supplementation/fallback when target result count is not met.

Fingerprinting Options:
        --hello-firefox           Simulate Firefox TLS fingerprint.
        --hello-chrome            Simulate Chrome TLS fingerprint (default).
        --hello-edge              Simulate Edge TLS fingerprint.
        --hello-safari            Simulate Safari TLS fingerprint.

Output & Storage Options:
    -w, --to-file                 Save results to a CSV file.
    -o, --out-file     string     Path for the output CSV file.
    -e, --to-db                   Save results to a SQLite3 database.
    -f, --db-file      string     Path for the SQLite3 database file.
    -g, --label        string     Label for output files and database records.
        --resolve-loc             Attempt to resolve and display Cloudflare location.
        --local-asn               Retrieve and store local ASN/city info.

Alias Options:
        --source                  Alias for --ip.
        --source-file             Alias for --in.
        --result-count            Alias for --result.
        --dt-workers              Alias for --dt-thread.
        --dt-timeout-ms           Alias for --dt-timeout.
        --dt-attempts             Alias for --dt-count.
        --dt-protocol             Alias for --dt-via.
        --sni-hostname            Alias for --hostname.
        --dt-status-code          Alias for --dt-expect-code.
        --dt-evaluate             Alias for --ev-dt.
        --dt-max-delay            Alias for --ev-dt-delay.
        --dt-min-pass-rate        Alias for --ev-dt-dtpr.
        --dt-max-stddev           Alias for --ev-dt-std.
        --dlt-workers             Alias for --dlt-thread.
        --dlt-duration            Alias for --dlt-period.
        --dlt-attempts            Alias for --dlt-count.
        --dlt-timeout-ms          Alias for --dlt-timeout.
        --test-interval-ms        Alias for --interval.
        --min-speed               Alias for --speed.
        --to-csv                  Alias for --to-file.
        --csv-file                Alias for --out-file.
        --to-sqlite               Alias for --to-db.
        --sqlite-file             Alias for --db-file.
        --record-label            Alias for --label.
        --resolve-location        Alias for --resolve-loc.
        --quiet                   Alias for --silence.

General Options:
    -S, --silence                 Enable silence mode with minimal output.
    -V, --debug                   Print detailed debug logs.
    -v, --version                 Show version information and exit.
    -h, --help                    Show this help message.
```

## Backward Compatibility

Deprecated flags are still accepted for existing scripts:

- `--disable-download` maps to `--dt-only`.
- `--dt-via-https` maps to `--dt-via https`.

Prefer the canonical names in new commands and documentation.

## Outbound Socket Controls

`--mark` and `--xmark` set the same outbound mark value. Values may be decimal (`123`) or hex (`0x7b`). Socket marks are Linux-only and may require root or `CAP_NET_ADMIN`.

`--interface` accepts a local source IP, interface name, or numeric interface index. Source-IP binding is portable. Interface name/index binding uses OS-specific socket support: Linux uses `SO_BINDTODEVICE`; Windows uses `IP_UNICAST_IF` / `IPV6_UNICAST_IF`; macOS and Solaris use `IP_BOUND_IF` / `IPV6_BOUND_IF`; FreeBSD, OpenBSD, NetBSD, and DragonFly BSD fall back to binding a source address from the selected interface.

## Database Schema (Table: `CFTD`)

SQLite output stores one row per qualified candidate with timing, pass-rate, speed, source ASN/city, label, and Cloudflare location fields. The CSV output uses the same result fields.

## Acknowledgments

This project is inspired by and built upon work from:

- [CloudflareScanner](https://github.com/Spedoske/CloudflareScanner)
- [CloudflareSpeedTest](https://github.com/XIU2/CloudflareSpeedTest)
- [utls](https://github.com/refraction-networking/utls)
