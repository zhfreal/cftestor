# cftestor:  Cloudflare可用节点测试工具

本项目可以测试Cloudflare节点，快速挑选出速度快的IP
## 特点
1、延时测试默认采用SSL握手方式非TCP建立连接方式，目测这样可以剔除Cloudflare专用的IP，初步筛选更加准确；另外可选择HTTP(s) Request方式测试实际连接的延迟。<br>
2、可通过命令行-s传入待测试的IP(段)，重复-s可传入多个IP(段);也可通过"-i 文件"方式传入待测试IP。另外内置CF的IP段，不指定-s或者-s时使用内置IP段。<br>
3、测试结果实时滚动显示，而非测试完后才能看到。<br>
4、默认测试结果自动按时间生成的新文件存储，避免之前的测试结果文件被覆盖。<br>
5、测试结果可存入sqlite数据库，数据库查询更方便日后重复利用。<br>
6、更多使用方法参见"-h"帮助。<br>

## 下载、编译和运行

### 下载、编译

请在[release](https://github.com/XIU2/CloudflareSpeedTest/releases)
中下载最新的预编译文件或自行编译。需要注意的是，由于使用[go-sqlite3](https://github.com/mattn/go-sqlite3)
编译时需要打开CGO，交叉编译时请使用CC和CXX变量传入gcc和g++工具链。
```bash
$ git clone https://github.com/zhfreal/cftestor.git
$ cd cftestor
$ CGO_ENABLED=1 go build .
$ ./cftestor -h

     cftestor v1.2.3
    测试Cloudflare IP的延迟和速度，获取最快的IP！
    https://github.com/zhfreal/cftestor

    参数:
        -s, --ip                    string  待测试IP(段)(默认为空)
                                            例如1.0.0.1或者1.0.0.0/32，可重复使用测试多个IP或者IP段。
        -i, --in                    string  IP(段) 数据文件
                                            文本文件，每一行为一个IP或者IP段。
        -m, --ping-thread           int     延时测试线程数量(默认 50)
        -t, --ping-timeout          int     延时超时时间(ms)(默认 1000ms)
                                            当使用"--ping-via-http"时，应适当加大。
        -c, --ping-try              int     延时测试次数(默认 4)
        -p, --port                  int     测速端口(默认 443)
                                            当使用SSL握手方式测试延时且不进行下载测试时，需要根据此参数测试；其余
                                            情况则是使用"--url"提供的参数进行测试。
            --hostname              string  SSL握手时使用的hostname(默认: "cf.zhfreal.nl")
                                            当使用SSL握手方式测试延时且不进行下载测试时，需要根据此参数测试；其余
                                            情况则是使用"--url"提供的参数进行测试。

            --ping-via-http                 使用HTTP请求方式进行延时测试开关(默认关闭，即使用SSL握手方式测试延时)
                                            当使用此模式时，"--ping-timeout"应适当加大；另外请根据自身服务器的情
                                            况，以及CF对实际访问量的限制，降低--ping-thread值，避免访问量过大，
                                            造成测试结果偏低。
        -n, --download-thread       int     下测试线程数(默认 1)
        -d, --download-max-duration int     单次下载测速最长时间(s)(默认 10s)
        -b, --download-try          int     尝试下载次数(默认 1)
        -u, --url                   string  下载测速地址(默认 "https://cf.zhfreal.nl/500mb.dat")。
                                            自定义下载文件建议使用压缩文件，避免CF或者HTTP容器设置压缩时使测试速度
                                            异常大；另外请在CF上关闭对此文件的缓存或者在服务器上将此文件加上用户名
                                            和密码实现访问控制，这样可以测试经过CF后到实际服务器整个链路的速度。当
                                            在服务器上对下载文件加上用户名和密码的访问控制时，可以如下格式传入url:
                                            "https://<用户名>:<密码>@cf.zhfreal.nl/500mb.dat", "<用户名>"
                                            和"<密码>"请用实际值替换。
            --interval              int     测试间隔时间(ms)(默认 100ms)
        -k, --time-limit            int     平均延时上限(ms)(默认 800ms)
                                            平均延时超过此值不计入结果集，不再进行下载测试。
        -l, --speed                 float   下载平均速度下限(KB/s)(默认 2000KB/s)
                                            下载平均速度低于此值时不计入结果集。
        -r, --result                int     测速结果集数量(默认 20)
                                            当符合条件的IP数量超过此值时，结束测试。但是如果开启"--testall"，此值
                                            不再生效。
            --disable-download              禁用下载测速开关(默认关闭，即需进行下载测试)
        -6, --ipv6                          测试IPv6开关(默认关闭，即进行IPv4测试，仅不携带-i且不携带-s时有效)
        -a  --test-all                      测试全部IP开关(默认关闭，仅不携带-s且不携带-i时有效)
        -w, --store-to-file                 是否将测试结果写入文件开关(默认关闭)
                                            当携带此参数且不携带-o参数时，输出文件名称自动生成。
        -o, --result-file           string  输出结果文件
                                            携带此参数将结果输出至本参数对应的文件。
        -e, --store-to-db                   是否将结果存入sqlite3数据库开关（默认关闭）
                                            此参数打开且不携带"--db-file"参数时，数据库文件默认为"ip.db"。
        -f, --db-file               string  sqlite3数据库文件名称。
                                            携带此参数将结果输出至本参数对应的数据库文件。
        -g, --label                 string  输出结果文件后缀或者数据库中数据记录的标签
                                            用于区分测试目标服务器。携带此参数时，在自动存储文件名模式下，文件名自
                                            动附加此值，数据库中Lable字段为此值。但如果携带"--result-file"时，
                                            此参数对文件名无效。当不携带此参数时，自动结果文件名后缀和数据库记录的
                                            标签为"--hostname"或者"--url"对应的域名。
        -V, --debug                         调试模式
        -v, --version                       打印程序版本
    pflag: help requested

$
```
### 运行
```bash
$./cftestor
2021-04-23 11:27:13    INFO     Start Ping and Speed Test —— Ping-Via-SSL  PingRTTMax(ms):800  SpeedMin(kB/s):2000  ResultLimit:20  PingTestThread:100  SpeedTestThread:1

2021-04-23 11:28:55    INFO     IP                 Speed(KB/s)    PingRTT(ms)    PingSR(%)
2021-04-23 11:28:55    INFO     172.67.38.50       2198.12        409            100.00
2021-04-23 11:29:26    INFO     104.20.12.104      4691.24        406            100.00
2021-04-23 11:30:28    INFO     104.18.134.68      4474.97        414            100.00
2021-04-23 11:30:59    INFO     104.16.243.79      6078.97        410            100.00
2021-04-23 11:31:29    INFO     104.17.0.216       8005.18        415            100.00
2021-04-23 11:32:01    INFO     104.18.90.63       7179.97        417            100.00
2021-04-23 11:32:32    INFO     104.21.117.130     6560.38        417            100.00
2021-04-23 11:33:02    INFO     104.16.218.176     5252.12        421            100.00
2021-04-23 11:33:33    INFO     104.22.14.5        6197.48        418            100.00
2021-04-23 11:34:04    INFO     104.17.14.145      4192.62        425            100.00
2021-04-23 11:34:35    INFO     104.18.144.22      4297.48        422            100.00
2021-04-23 11:35:06    INFO     104.17.45.101      4602.77        421            100.00
2021-04-23 11:35:37    INFO     104.16.223.193     3321.68        426            100.00
2021-04-23 11:36:08    INFO     104.22.65.79       6658.96        425            100.00
2021-04-23 11:36:39    INFO     104.19.212.63      6942.95        427            100.00
2021-04-23 11:37:10    INFO     104.20.117.215     7249.88        427            100.00
2021-04-23 11:37:41    INFO     104.20.80.217      3699.74        429            100.00
2021-04-23 11:38:12    INFO     104.20.240.41      6599.43        427            100.00
2021-04-23 11:39:16    INFO     104.17.84.120      4130.91        431            100.00
2021-04-23 11:39:47    INFO     104.19.115.6       4559.62        443            100.00
2021-04-23 11:40:46    INFO     TotalQualified:20       TotalforPingTest:1518043    TotalPingTested:7207     TotalforSpeedTest:2833     TotalSpeedTested:24


2021-04-23 11:40:46    INFO     All Result(Reverse Order):

TestTime               IP                 Speed(KB/s)    PingRTT(ms)    PingSR(%)
2021-04-23 11:31:29    104.17.0.216       8005.18        415            100.00
2021-04-23 11:37:10    104.20.117.215     7249.88        427            100.00
2021-04-23 11:32:01    104.18.90.63       7179.97        417            100.00
2021-04-23 11:36:39    104.19.212.63      6942.95        427            100.00
2021-04-23 11:36:08    104.22.65.79       6658.96        425            100.00
2021-04-23 11:38:12    104.20.240.41      6599.43        427            100.00
2021-04-23 11:32:32    104.21.117.130     6560.38        417            100.00
2021-04-23 11:33:33    104.22.14.5        6197.48        418            100.00
2021-04-23 11:30:59    104.16.243.79      6078.97        410            100.00
2021-04-23 11:33:02    104.16.218.176     5252.12        421            100.00
2021-04-23 11:29:26    104.20.12.104      4691.24        406            100.00
2021-04-23 11:35:06    104.17.45.101      4602.77        421            100.00
2021-04-23 11:39:47    104.19.115.6       4559.62        443            100.00
2021-04-23 11:30:28    104.18.134.68      4474.97        414            100.00
2021-04-23 11:34:35    104.18.144.22      4297.48        422            100.00
2021-04-23 11:34:04    104.17.14.145      4192.62        425            100.00
2021-04-23 11:39:16    104.17.84.120      4130.91        431            100.00
2021-04-23 11:37:41    104.20.80.217      3699.74        429            100.00
2021-04-23 11:35:37    104.16.223.193     3321.68        426            100.00
2021-04-23 11:28:55    172.67.38.50       2198.12        409            100.00
$
```
因对输出进行了格式化对齐，中英文混合很难动态计算长度、对齐，故测试过程和结果没有使用中文。好在测试结果简单，只需关注测试数据即可。
## 注意事项
### 自定义测试地址，找到符合自己的最佳IP
默认测试中--url参数中对应的文件已经在CF上缓存，测试结果未必适用于您的服务器。若需找到最佳IP，建议使用自己的测试地址。 <br>
1、自行在落地服务器上创建测试文件，测试文件最好压缩。避免CF或者HTTP容器(nginx/apache/caddy等)使用压缩方式传输数据时测试速度异常大。<br>
2、在CF上通过设置页面规则关闭对此测试文件的缓存，或者在HTTP容器中对此文件进行用户名和密码的访问控制(此时可按此传入: "-u https://<用户名>:<密码>@cf.zhfreal.nl/500mb.dat", "<用户名>"和"<密码>"请用实际值替换)。<br>
3、因CF没有缓存，请注意落地服务器的流量消耗。<br>

### 使用"--ping-via-http"参数测试延时
#### "--ping-timeout"参数增大
此时测试的是HTTP(s)请求和响应的延迟，建议"--ping-timeout"参数值在2000(ms)以上。因为HTTP(s)请求一般会使用HTTPS在SSL握手的基础上再进行HTTP请求和响应交互。其时间远长于SSL握手时间。否则影响测试结果。
#### ”--ping-thread“参数和"--download-thread"参数减小
这种方式下因ping的过程实际上是在做HTTP(s)请求和响应交互，请根据落地服务器的实际性能和CF的限制，降低ping和Download的线程数。建议"--ping-thread 5", ”--download-thread 1“。否则影响测试结果。


### Sqlite数据库存储测试结果
#### 表：CFTestDetails，字段如下：
    TestTime                  datetime     测试时间                         
    ASN                       int          测试所使用本地网络的ASN          
    CITY                      text         测试所在地                       
    IP                        text         目标CF的IP地址                   
    LABEL                     text         落地服务器标识                   
    ConnCount                 int          连接尝试次数                     
    ConnSuccessCount          int          连接成功次数                     
    ConnSuccessRate           float        连接成功率                       
    ConnDurationAvg           float        连接平均延迟                     
    ConnDurationMin           float        连接最小延迟                     
    ConnDurationMax           float        连接最大延迟                     
    DownloadCount             int          下载尝试次数                     
    DownloadSuccessCount      int          下载成功次数                     
    DownloadSuccessRatio      float        下载成功率                       
    DownloadSpeedAvg          float        下载平均速度(KB/s)               
    DownloadSize              int          总下载数据大小(byte)             
    DownloadDurationSec       float        总下载时间(秒) 
#### 数据库目前存储结果，可自行通过sqlite命令行等方式查询使用结果，方便测试结果的追踪和重复使用。
## 参考
> https://github.com/Spedoske/CloudflareScanner
> <br>
> https://github.com/XIU2/CloudflareSpeedTest
