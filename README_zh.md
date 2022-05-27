# cftestor:  Cloudflare CDN IP测试工具

本项目可以测试Cloudflare CDN IP地址，根据延迟和速度挑选出最佳地址。

## 原理
1. 根据延迟和速度评估CF CDN地址。<br>
2. 延迟测试(DT)采用SSL/TLS握手或HTTPS连接方式非TCP握手方式，通过该IP，进行SSL/TLS握手或HTTPS连接，如果成功，统计花费时间。测试几次后统计成功率和平均延迟，并且根据成功率和平均延迟筛选合格的IP。<br>
3. DT测试合格的IP进入下载测试(DLT)。DLT则是通过使用该IP下载CDN网络上的资源，计算平均下载速度。可尝试多次下载，合并统计平均速度。当其平均速度符合条件时，此IP为合格的IP。<br>
4. DT可并行测试，同时测试多个IP。DLT一般串行，一次测试一个IP，避免相互占有带宽。可设置结果集数量，通过测试的IP数量达到结果集限制，则结束测试。<br>

## 下载. 编译和运行

### 下载. 编译

请在[release](https://github.com/zhfreal/cftestor/releases)
中下载最新的预编译文件或自行编译。
```bash
$ git clone https://github.com/zhfreal/cftestor.git
$ cd cftestor
$ go build .
$ ./cftestor -h

    cftestor v1.5.1
    根据延迟. 速度优选CF CDN IP
    https://github.com/zhfreal/cftestor

    参数:
        -s, --ip            string  待测试IP(段)。例如: "-s 1.0.0.1", "-s 1.0.0.1/32",
                                    "-s 1.0.0.1/24"。可重复使用, 传递多个IP或者IP段。
        -i, --in            string  IP(段) 数据文件, 文本文件。 其每一行为一个IP或者IP段。
        -m, --dt-thread     int     延时测试线程数量, 默认20。
        -t, --dt-timeout    int     延时超时时间(ms), 默认1000ms。此值不能小于
                                    "-k|--delay-limit"。当使用"--dt-via-https"时, 应适
                                    当加大此值。
        -c, --dt-count      int     延时测试次数, 默认4。
        -p, --port          int     测速端口, 默认443。当使用SSL握手方式测试延时且不进行下载测
                                    试时, 需要根据此参数测试；其余情况则是使用"--url"提供的参
                                    数进行测试。
            --hostname      string  SSL握手时使用的hostname, 默认"cf.9999876.xyz"。仅当
                                    "--dt-only"且不携带"-dt-via-https"时有效。
            --dt-via-https          使用HTTPS请求相应方式进行延时测试开关。
                                    默认关闭, 即使用SSL握手方式测试延时。
        -n, --dlt-thread    int     下测试线程数, 默认1。
        -d, --dlt-period    int     单次下载测速最长时间(s), 默认10s。
        -b, --dlt-count     int     尝试下载次数, 默认1。
        -u, --url           string  下载测速地址, 默认 "https://cf.9999876.xyz/500mb.dat"。
        -I  --interval      int     测试间隔时间(ms), 默认500ms。
        -k, --delay-limit   int     平均延时上限(ms), 默认600ms。 平均延时超过此值不计入结果集,
                                    不进行下载测试。
        -S, --dtpr-limit    float   延迟测试成功率下限(%), 默认100%。
                                    当低于此值时不计入结果集, 不进行下载测试。默认100, 即不低于
                                    100%。此值低于100%的IP会发生断流或者偶尔无法连接的情况。
        -l, --speed         float   下载平均速度下限(KB/s), 默认6000KB/s。下载平均速度低于此值
                                    时不计入结果集。
        -r, --result        int     测速结果集数量, 默认10。
                                    当符合条件的IP数量超过此值时, 结束测试。但是如果开启
                                    "--test-all", 此值不生效。
            --dt-only               只进行延迟测试, 不进行下载测速开关, 默认关闭。
            --dlt-only              不单独使用延迟测试, 直接使用下载测试, 默认关闭。
        -4, --ipv4                  测试IPv4开关, 表示测试IPv4地址。仅当不携带"-s"和"-i"时有效。
                                    默认打开。与"-6|--ipv6"不能同时使用。
        -6, --ipv6                  测试IPv6开关, 表示测试IPv6地址。仅当不携带"-s"和"-i"时有效。
                                    默认关闭。与"-4|--ipv4"不能同时使用。
        -a  --test-all              测试全部IP开关。默认关闭。
        -w, --store-to-file         是否将测试结果写入文件开关, 默认关闭。
        -o, --result-file   string  输出结果文件。携带此参数将结果输出至本参数对应的文件。
        -e, --store-to-db           是否将结果存入sqlite3数据库开关。默认关闭。
        -f, --db-file       string  sqlite3数据库文件名称。携带此参数将结果输出至本参数对应的数
                                    据库文件。
        -g, --label         string  输出结果文件后缀或者数据库中数据记录的标签, 用于区分测试目标
                                    服务器。默认为"--url"地址的hostname或者"--hostname"。
            --no-tcell      bool    不使用TCell显示。
        -V, --debug                 调试模式。
        -v, --version               打印版本。
    pflag: help requested
$
```
### 运行
```bash
$./cftestor

运行画面：
```
![alt text](Result.png "运行画面")</br>
结果：
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
> Speed(KB/s): 下载速度， 单位KB/s
> DelayAvg(ms): TCP同步包发送到SSL握手成功(或者收到http响应时)的平均时间，单位毫秒ms
> Stability(%): 延迟测试（SSL握手或者HTTPS连接）的成功率，大致反映网络的丢包率。
```

## 进阶应用
### 1. 自定义测试地址，找到符合自己的最佳IP
默认测试结果只能反映当前位置到CloudFlare服务器的网络情况，不能反映CloudFlare到落地服务器的网络状况。若要测试当前位置经过CloudFlare服务器到落地服务器整条链路的状况，需使用自己的测试地址。 <br>
(1). 自行在Cloudflare创建站点。<br>
(2). 自行在落地服务器上创建测试文件，测试文件最好压缩。避免CF或者HTTP容器(nginx/apache/caddy等)使用压缩方式传输数据，干扰测试结果。<br>
(3). 在CF上通过设置页面规则关闭对此测试文件的缓存，或者在HTTP容器中对此文件进行用户名和密码的简单访问控制(此时可按此格式传入: "-u https://<用户名>:<密码>@cf.example.com/test.dat")。<br>
(4). 请注意落地服务器的流量消耗。<br>

### 2. 优化参数
(1). **"-m|--dt-thread"**: 延迟测试(DT)并发任务数量，根据ISP的Qos策略情况，其可能影响测试结果。<br>
(2). "-I|--interval": 测试间隔，DT和DLT所有测试的间隔，根据ISP的Qos策略情况，其可能影响各个测试的结果，对测试进程也有不小的影响。<br>
(3). **"-t|--dt-timeout"**: 延迟测试(DT)的SSL/TLS握手或者HTTPS请求超时时间，未在该时间内完成SSL/TLS握手或者收到HTTPS Response则失败。此参数影响测试进度。<br>
(4). **"-c|--dt-count"** DT测试次数，增加此参数值，增加测试总时长，可能增加延迟测试准确性。<br>
(5). **"-d|--dlt-period"**, **"-b|--dlt-count"**, 增加这两参数值，增加DLT测试时常，可能增加下载速度准确性。<br>
(6). **"-k|--delay-limit"**, **"-S|--dtpr-limit"**, **"-l|--speed"** 三个参数直接影响筛选结果。<br>
(7). TCell模式下，依次通过 **"ESC" 和 "Enter" 键提前中断测试**。<br>
(8). 如果长时间没有符合条件的IP，建议打开调试模式 **"-V|--debug"** ，查看调试信息，调整优选参数。<br>

### 3. Sqlite数据库存储测试结果
#### 表：CFTD，字段如下：
```
    TestTime      datetime     测试时间                         
    ASN           int          测试所使用本地网络的ASN          
    CITY          text         测试所在地                       
    IP            text         目标CF的IP地址                   
    LABEL         text         落地服务器标识                   
    DTS           text         延迟类型(SSL or HTTPS)
    DTC           int          延迟测试次数                     
    DTPC          int          延迟测试通过次数                     
    DTPR          float        延迟测试成功率                       
    DA            float        平均延迟                     
    DMI           float        最小延迟                     
    DMX           float        最大延迟                     
    DLTC          int          下载尝试次数                     
    DLTPC         int          下载成功次数                     
    DLTPR         float        下载成功率                       
    DLSA          float        下载平均速度(KB/s)               
    DLDS          int          总下载数据大小(byte)
    DLTD          float        总下载时间(秒) 
```
数据库目前仅存储测试结果，方便测试结果的追踪和重复使用。可自行通过sqlite命令行等方式查询使用。
## 感谢:
> 
> <a href="https://github.com/Spedoske/CloudflareScanner">github.com/Spedoske/CloudflareScanner</a>
> 
> <a href="https://github.com/XIU2/CloudflareSpeedTest">github.com/XIU2/CloudflareSpeedTest</a>
> 
> <a href="https://github.com/gdamore/tcell">github.com/gdamore/tcell</a>
>
>   