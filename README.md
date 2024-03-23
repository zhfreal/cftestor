# cftestor:  Find and verify the best IPs for Cloudflare CDN

In some regions, we can't get the best access from Cloudflare CDN due to the ISP disturbed. So, this util could find better IPs for you to access Cloudflare CDN with lower legacy and greater speed.

## How does it work?
1. Our goal is find better IPs to access Cloudflare CDN with lower legacy and greater speed. So we will perform Delay Test (DT) and Download Test (DLT) sequentially for an IP as assessment moves. <br>
2. We do DT by SSL/TLS shakhand or HTTPS get with the target IP. If it succeed to perform SSL/TLS or HTTPS get, we take the times escapted as delay. We will try several times to perfrom DT for a single IP and count the success tries. If it's success rate ( equal to "count of success" / "tries" ) is not less than what we expected, and the average delay ( "sum of every delay" / "count of success" ) is not bigger than we expected, it pass the DT. <br>
3. If an IP passed DT, then we perform DLT with it. We download some test data with it, caculate the average download speed. If the average speed is not less than we expected, it's a successful DLT. We can perform several DLT sequentially to evaluate download speed accurately.<br>
4. If an IP passsed DLT, it's what we need exactly. <br>

## Quick Start

### Get the pre-build binary

Download pre-build binary form [release](https://github.com/zhfreal/cftestor/releases) , or build yourself from source code.

```bash
$ git clone https://github.com/zhfreal/cftestor.git
$ cd cftestor
$ go build .
```
### basic
```bash
$ cftestor
  ░█▀▀░█▀▀░▀█▀░█▀▀░█▀▀░▀█▀░█▀█░█▀▄
  ░█░░░█▀▀░░█░░█▀▀░▀▀█░░█░░█░█░█▀▄
  ░▀▀▀░▀░░░░▀░░▀▀▀░▀▀▀░░▀░░▀▀▀░▀░▀

  CF CDN IP scanner, find best IPs for your Cloudflare CDN applications.
  https://github.com/zhfreal/cftestor

Version: v1.9.0
BuildOn: 2024-03-23 13:47:50 +0800
BuildTag: release, by zhfreal
BuildFrom: 9538c569cf28ec357dcd61244da5f9d5bcba9f5c

14:24:50 INFO IP:104.18.59.34:443 Speed(KB/s):10125.14 Delay(ms):193 Stab.(%):100.00
14:24:53 INFO IP:104.19.194.192:443 Speed(KB/s):18474.78 Delay(ms):518 Stab.(%):100.00
```

### lower latency and more faster
```bash
$ cftestor -k 150 -l 200000
```

```
> Speed(KB/s): Download speed in KB/s
> DelayAvg(ms): Average delay for DT in ms
> Stability(%): DT pass rate(%)
```

### more usage
```bash
$ ./cftestor -h

  ░█▀▀░█▀▀░▀█▀░█▀▀░█▀▀░▀█▀░█▀█░█▀▄
  ░█░░░█▀▀░░█░░█▀▀░▀▀█░░█░░█░█░█▀▄
  ░▀▀▀░▀░░░░▀░░▀▀▀░▀▀▀░░▀░░▀▀▀░▀░▀

  CF CDN IP scanner, find best IPs for your Cloudflare CDN applications.
  https://github.com/zhfreal/cftestor

Version: v1.9.0
BuildOn: 2024-03-23 13:47:50 +0800
BuildTag: release, by zhfreal
BuildFrom: 9538c569cf28ec357dcd61244da5f9d5bcba9f5c

    Usage: cftestor [options]
    options:
    -s, --ip           string  Specific IP, CIDR or host for test. E.g.: "-s 1.0.0.1", "-s 1.0.0.1/32",
                               "-s 1.0.0.1/24", "-s 1.1.1.1:443".
    -i, --in           string  Specify file for test, which contains multiple lines. Each line
                               represent one IP or CIDR.
    -p, --port         int     Port to test, could be specific one or more ports at same time,
                               The port should be working via SSL/TLS/HTTPS protocol,  default 443.
    -m, --dt-thread    int     Number of concurrent threads for Delay Test(DT). How many IPs can 
                               be perform DT at the same time. Default 20 threads.
    -t, --dt-timeout   int     Timeout for single DT, unit ms, default 1000ms. A single SSL/TLS 
                               or HTTPS request and response should be finished before timeout. 
                               It should not be less than "-k|--evaluate-dt-delay", It should be 
                               longer when we perform https connections test by "-dt-via-https" 
                               than when we perform SSL/TLS test by default.
    -c, --dt-count     int     Tries of DT for a IP, default 4.
        --hostname     string  Hostname for DT test. It's valid when "--dt-only" is no and "--dt-via-https" 
                               is not provided.
        --dt-via https|tls|ssl DT via https or SSL/TLS shaking hands, "--dt-via <https|tls|ssl>"
                               default https.
        --dt-url       string  Specify test URL for DT.
        --ev-dt                Evaluate DT, we'll try "-c|--dt-count value" to evaluate delay;
                               if we don't turn this on, we'll stop DT after we got the first
                               successfull DT; if we turn this on, we'll evaluate the test result 
                               through average delay of singe DT and statistic of all successfull
                               DT by these two thresholds  "-k|--evaluate-dt-delay value" and 
                               "-S|--evaluate-dt-dtpr value". default turn off.
    -k, --ev-dt-delay  int     single DT's delay should not bigger than this, unit ms, default 600ms.
    -S, --ev-dt-dtpr   float   The DT pass rate should not lower than this, default 100, means 100%, all
                               DT must be below "-k|--evaluate-dt-delay value".
    -n, --dlt-thread   int     Number of concurrent Threads for Download Test(DLT), default 1. 
                               How many IPs can be perform DLT at the same time.
    -d, --dlt-period   int     The total times escaped for single DLT, default 10s.
    -b, --dlt-count    int     Tries of DLT for a IP, default 1.
    -u, --dlt-url      string  Specify test URL for DLT.
        --dlt-timeout  int     Specify the timeout for http response when do DLT. In ms, default as 5000 ms.
    -I  --interval     int     Interval between two tests, unit ms, default 500ms.

    -l, --speed        float   Download speed filter, Unit KB/s, default 6000KB/s. After DLT, it's 
                               qualified when its speed is not lower than this value.
    -r, --result       int     The total IPs qualified limitation, default 10. The Process will stop 
                               after it got equal or more than this indicated. It would be invalid if
                               "--test-all" was set.
        --dt-only              Do DT only, we do DT & DLT at the same time by default.
        --dlt-only             Do DLT only, we do DT & DLT at the same time by default.
        --fast                 Fast mode, use inner IPs for fast detection. Just when neither"-s|--ip value"
                               nor "-i|--in value" is provided, and this flag is provided. It will be working
                               Disabled by default.
    -4, --ipv4                 Just test IPv4. When we don't specify IPs by "-s|--ip value" and "-i|--in value",
                               then it will do IPv4 test from build-in IPs from CloudFlare by default.
    -6, --ipv6                 Just test IPv6. When we don't specify IPs by "-s|--ip value" and "-i|--in value",
                               then it will do IPv6 test from build-in IPs from CloudFlare by using
                               this flag.
    -a  --test-all             Test all IPs until no more IP left. It's disabled by default. 
    -w, --to-file              Write result to csv file, disabled by default. If it is provided and 
                               "-o|--result-file" is not provided, the result file will be named
                               as "Result_<YYYYMMDDHHMISS>-<HOSTNAME>.csv" and be stored in current DIR.
    -o, --outfile      string  File name of result. If it don't provided and "-w|--store-to-file"
                               is provided, the result file will be named as 
                               "Result_<YYYYMMDDHHMISS>-<HOSTNAME>.csv" and be stored in current DIR.
    -e, --to-db                Write result to sqlite3 db file, disabled by default. If it's provided
                               and "-f|--db-file" is not provided, it will be named "ip.db" and
                               store in current directory.
    -f, --dbfile       string  Sqlite3 db file name. If it's not provided and "-e|--store-to-db" is
                               provided, it will be named "ip.db" and store in current directory.
    -g, --label        string  the label for a part of the result file's name and sqlite3 record. It's 
                               hostname from "--hostname" or "-u|--url" by default.
    -V, --debug                Print debug message.
        --tcell                Use tcell to display the running procedure when in debug mode.
                               Turn this on will activate "--debug".
    -v, --version              Show version.
pflag: help requested
$
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
