# cftestor:  Find and verify the best IPs for Cloudflare CDN

In some regions, we can't get the best access from Cloudflare CDN due to the ISP disturbed. So, this util could find better IPs for you to access Cloudflare CDN with lower legacy and greater speed.

## How does it work?
1. Our goal is find better IPs to access Cloudflare CDN with lower legacy and greater speed. So we will perform Delay Test (DT) and Download Test (DLT) sequentially for an IP as assessment moves.
2. We do DT by SSL/TLS shakhand or HTTPS get with the target IP. If it secceed to perform SSL/TLS or HTTPS get, we take the times escapted as delay. We will try several times to perfrom DT for a single IP and count the success tries. If it's success rate ( equal to <count of success>/<tries> ) is not less than what we expected, and the average delay ( <sum of every delay>/<count of success> ) is not bigger than we expected, it pass the DT.
3. If an IP passed DT, then we perform DLT with it. We download some test data with it, caculate the average download speed. If the average speed is not less than we expected, it's a successful DLT. We can perform several DLT sequentially to evaluate download speed accurately.
4. If an IP passsed DLT, it's what we need exactly.

## Quick Start

### Get the pre-build binary

Download pre-build binary form [release](https://github.com/zhfreal/cftestor/releases) , or build yourself from source code.

```bash
$ git clone https://github.com/zhfreal/cftestor.git
$ cd cftestor
$ go build .
```
### Help
```bash
$ ./cftestor -h

cftestor v1.5.1
    CF IP sanner, evaluation througth delay and download speed, find your best IPs Cloudfare CDN applications.
    https://github.com/zhfreal/cftestor

    Usage: cftestor [options]
    options:
        -s, --ip            string  Specific IP or CIDR for test. E.g.: "-s 1.0.0.1", "-s 1.0.0.1/32",
                                    "-s 1.0.0.1/24".
        -i, --in            string  Specific file for test, which contains multiple lines. Each line
                                    represent one IP or CIDR.
        -m, --dt-thread     int     Number of concurrent threads for Delay Test(DT). How many IPs can
                                    be perform DT at the same time. Default 20 threads.
        -t, --dt-timeout    int     Timeout for single DT, unit ms, default 1000ms. A single SSL/TLS
                                    or HTTPS request and response should be finished before timeout.
                                    It should not be less than "-k|--delay-limit", It should be
                                    longger when we perform https connections test by "-dt-via-https"
                                    than when we perform SSL/TLS test by default.
        -c, --dt-count      int     Tries of DT for a IP, default 4.
        -p, --port          int     Port to test, default 443. It's valid when "--only-dt" and "--dt-via-https".
            --hostname      string  Hostname for DT test. It's valid when "--dt-only" is no and "--dt-via-https"
                                    is not provoided.
            --dt-via-https          DT via https other than SSL/TLS shakehand. It's disabled by default,
                                    we do DT via SSL/TLS.
        -n, --dlt-thread    int     Number of concurrent Threads for Download Test(DLT), default 1.
                                    How many IPs can be perform DLT at the same time.
        -d, --dlt-period    int     The total times escaped for single DLT, default 10s.
        -b, --dlt-count     int     Tries of DLT for a IP, default 1.
        -u, --url           string  Customize test URL for DLT.
        -I  --interval      int     Interval between two tests, unit ms, default 500ms.
        -k, --delay-limit   int     Delay filter for DT, unit ms, default 600ms. If A ip's average delay
                                    bigger than this value after DT, it is not qualified and won't do
                                    DLT if DLT required.
        -S, --dtpr-limit    float   The DT pass rate filter, default 100%. It means do 4 times DTs by
                                    default for a IP, it's passed just when no single DT failed.
        -l, --speed         float   Download speed filter, Unit KB/s, default 6000KB/s. After DLT, it's
                                    qualified when its speed is not lower than this value.
        -r, --result        int     The total IPs qualified limitation, default 10. The Process will stop
                                    after it got equal or more than this indicated. It would be invalid if
                                    "--test-all" was set.
            --dt-only               Do DT only, we do DT & DLT at the same time by default.
            --dlt-only              Do DLT only, we do DT & DLT at the same time by default.
        -4, --ipv4                  Just test IPv4. When we don't specify IPs to test by "-s" or "-i",
                                    then it will do IPv4 test from build-in IPs from CloudFlare by default.
        -6, --ipv6                  Just test IPv6. When we don't specify IPs to test by "-s" or "-i",
                                    then it will do IPv6 test from build-in IPs from CloudFlare by using
                                    this flag.
        -a  --test-all              Test all IPs until no more IP left. It's disabled by default.
        -w, --store-to-file         Write result to csv file, disabled by default. If it is provoided and
                                    "-o|--result-file" is not provoided, the result file will be named
                                    as "Result_<YYYYMMDDHHMISS>-<HOSTNAME>.csv" and be stored in current DIR.
        -o, --result-file   string  File name of result. If it don't provoided and "-w|--store-to-file"
                                    is provoided, the result file will be named as
                                    "Result_<YYYYMMDDHHMISS>-<HOSTNAME>.csv" and be stroed in current DIR.
        -e, --store-to-db           Write result to sqlite3 db file, disabled by default. If it's provoided
                                    and "-f|--db-file" is not provoided, it will be named "ip.db" and
                                    store in current directory.
        -f, --db-file       string  Sqlite3 db file name. If it's not provoided and "-e|--store-to-db" is
                                    provoided, it will be named "ip.db" and store in current directory.
        -g, --label         string  Lable for a part of the result file's name and sqlite3 record. It's
                                    hostname from "--hostname" or "-u|--url" by default.
            --no-tcell      bool    Don't use tcell to display the running procedure, disabled by default.
        -V, --debug                 Print debug message.
        -v, --version               Show version.
    pflag: help requested
$
```
### Runing test
```bash
$./cftestor
```
tcell screen during running:

![alt text](Result.png "running")</br>
Result:
```bash
$./cftestor

All Results:

TestTime IP              Speed(KB/s) DelayAvg(ms) Stability(%)
14:26:26 172.67.122.133  11912.66    373          100.00
14:27:50 172.67.34.139   9078.23     389          100.00
14:29:22 172.67.99.232   8162.36     410          100.00
14:27:15 172.67.2.55     7905.95     402          100.00
14:37:31 172.67.124.78   7144.92     384          100.00
14:34:49 172.67.162.221  6715.65     370          100.00
14:35:00 172.67.93.113   6466.62     378          100.00
14:29:34 172.67.243.14   6446.14     471          100.00
14:36:33 172.64.154.19   6387.97     383          100.00
14:34:25 172.67.172.221  6220.86     366          100.00

```

```
> Speed(KB/s): Download speed in KB/s
> DelayAvg(ms): Average delay for DT in ms
> Stability(%): DT pass rate(%)
```

### Data stored in sqlite3 DB
#### table - CFTD have these columns：
```
    TestTime      datetime     when the test happened
    ASN           int          ASN of your local network
    CITY          text         city of your local network
    IP            text         valid IP for CloudFare CDN access
    LABEL         text         label while stand for your CloudFare CDN resources
    DTS           text         the method for DT (SSL or HTTPS)
    DTC           int          tries for DT
    DTPC          int          success count of DT
    DTPR          float        success rate of DT
    DA            float        average delay of DT
    DMI           float        minimal delay of DT
    DMX           float        maximum delay of DT
    DLTC          int          tries for DLT
    DLTPC         int          success count of DLT
    DLTPR         float        success rate of DLT
    DLSA          float        average download speed (KB/s)
    DLDS          int          total bytes downloaded
    DLTD          float        total times escapted during download (in second)
```
## References:
> 
> <a href="https://github.com/Spedoske/CloudflareScanner">github.com/Spedoske/CloudflareScanner</a>
> 
> <a href="https://github.com/XIU2/CloudflareSpeedTest">github.com/XIU2/CloudflareSpeedTest</a>
> 
> <a href="https://github.com/gdamore/tcell">github.com/gdamore/tcell</a>
>
>   