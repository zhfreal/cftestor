package main

import (
	"context"
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"
	"strconv"
	"sync"
	"time"
)

type ResultHttp struct {
	dnsStartAt      time.Time
	dnsEndAt        time.Time
	tcpStartAt      time.Time
	tcpEndAt        time.Time
	tlsStartAt      time.Time
	tlsEndAt        time.Time
	httpReqAt       time.Time
	httpRspAt       time.Time
	bodyReadStartAt time.Time
	bodyReadEndAt   time.Time
}

func WithHTTPStat(ctx context.Context, r *ResultHttp) context.Context {
	return httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		DNSStart: func(i httptrace.DNSStartInfo) {
			r.dnsStartAt = time.Now()
		},
		DNSDone: func(i httptrace.DNSDoneInfo) {
			r.dnsEndAt = time.Now()
		},
		ConnectStart: func(_, _ string) {
			r.tcpStartAt = time.Now()
			// When connecting to IP (When no DNS lookup)
			if r.dnsStartAt.IsZero() {
				r.dnsStartAt = r.tcpStartAt
				r.dnsEndAt = r.tcpStartAt
			}
		},
		ConnectDone: func(network, addr string, err error) {
			r.tcpEndAt = time.Now()
		},
		TLSHandshakeStart: func() {
			r.tlsStartAt = time.Now()
		},
		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			r.tlsEndAt = time.Now()
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			r.httpReqAt = time.Now()
			if r.dnsStartAt.IsZero() && r.tcpStartAt.IsZero() {
				now := r.httpReqAt
				r.dnsStartAt = now
				r.dnsEndAt = now
				r.tcpStartAt = now
				r.tcpEndAt = now
			}
		},
		GotFirstResponseByte: func() {
			r.httpRspAt = time.Now()
			r.bodyReadStartAt = r.httpRspAt
		},
	})
}

// download test core
func downloadHandler(ip net.IP, tcpport int, tUrl string, HttpRspTimeoutDuration time.Duration, dltMaxDuration time.Duration,
	dltCount int, interval int, dtOnly bool) []singleResult {
	var fullAddress string
	if ip.To4() != nil { //IPv4
		fullAddress = ip.String() + ":" + strconv.Itoa(tcpport)
	} else { //
		fullAddress = "[" + ip.String() + "]:" + strconv.Itoa(tcpport)
	}

	var allResult = make([]singleResult, 0)
	// loop for test
	for i := 0; i < dltCount; i++ {
		tReq, err := http.NewRequest("GET", tUrl, nil)
		if err != nil {
			log.Fatal(err)
		}
		var tResultHttp ResultHttp
		tCtx := WithHTTPStat(tReq.Context(), &tResultHttp)
		tReq = tReq.WithContext(tCtx)
		var client = http.Client{
			Transport:     nil,
			CheckRedirect: nil,
			Jar:           nil,
			Timeout:       HttpRspTimeoutDuration,
		}
		if !dtOnly {
			client.Timeout += dltMaxDuration
		}
		client.Transport = &http.Transport{
			DialContext: GetDialContextByAddr(fullAddress),
			//ResponseHeaderTimeout: HttpRspTimeoutDuration,
		}
		var currentResult = singleResult{false, 0, 0, false, false, 0, 0}
		response, err := client.Do(tReq)
		// pingect is failed(network error), won't continue
		if err != nil {
			allResult = append(allResult, currentResult)
			time.Sleep(time.Duration(interval) * time.Millisecond)
			continue
		}
		currentResult.dTPassedCount = true
		currentResult.dTDuration = tResultHttp.tlsEndAt.Sub(tResultHttp.tcpStartAt)
		currentResult.httpRRDuration = tResultHttp.httpRspAt.Sub(tResultHttp.httpReqAt)
		// pingection test only, won't do download test
		if dtOnly {
			allResult = append(allResult, currentResult)
			time.Sleep(time.Duration(interval) * time.Millisecond)
			continue
		}
		// if download test permitted, set DownloadPerformed to true
		currentResult.dLTWasDone = true
		// pingect is not make(uri error or server error), won't do download test
		if response.StatusCode != 200 {
			allResult = append(allResult, currentResult)
			time.Sleep(time.Duration(interval) * time.Millisecond)
			continue
		}
		// start timing for download test
		tResultHttp.bodyReadStartAt = time.Now()
		timeEndExpected := tResultHttp.bodyReadStartAt.Add(dltMaxDuration)
		contentLength := response.ContentLength
		if contentLength == -1 {
			contentLength = fileDefaultSize
		}
		buffer := make([]byte, downloadBufferSize)
		var contentRead int64 = 0
		var downloadSuccess = false
		// just read  the length of content which indicated in response and read before time expire
		var tTimer = 0
		for contentRead < contentLength && time.Now().Before(timeEndExpected) {
			bufferRead, tErr := response.Body.Read(buffer)
			// there is an error occured and it's not io.EOF(read ended)
			// don't download anymore
			if tErr != nil {
				if tErr2, ok := tErr.(net.Error); ok && tErr2.Timeout() {
					//myLogger.Debug(fmt.Sprintf("FullAddress: %s, Round %d, deadline exceeded", fullAddress, i))
				} else if tErr == io.EOF {
					//myLogger.Debug(fmt.Sprintf("FullAddress: %s, Round %d, read end!", fullAddress, i))
					// other error occur
				} else {
					/*myLogger.Debug(fmt.Sprintf("FullAddress: %s, Round %d, error: %v!, %5.2f", fullAddress, i, err,
					  float64(time.Now().Sub(timeStart))/float64(time.Millisecond)))*/
					downloadSuccess = false
					break
				}
				contentRead += int64(bufferRead)
				downloadSuccess = true
				break
			}
			tTimer += 1
			//myLogger.Debug(fmt.Sprintf("FullAddress: %s, Round %d, success for %3d", fullAddress, i, tTimer))
			contentRead += int64(bufferRead)
			downloadSuccess = true
		}
		currentResult.dLTPassed = downloadSuccess
		tResultHttp.bodyReadEndAt = time.Now()
		currentResult.dLTDuration = tResultHttp.bodyReadEndAt.Sub(tResultHttp.bodyReadStartAt)
		currentResult.dLTDataSize = contentRead
		allResult = append(allResult, currentResult)
		_ = response.Body.Close()
		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
	return allResult
}

func downloadWorker(chanIn chan string, chanOut chan singleVerifyResult, chanOnGoing chan int, wg *sync.WaitGroup,
	tUrl string, tcpport int, HttpRspTimeoutDuration time.Duration, dltMaxDuration time.Duration,
	dltCount int, interval int, dtOnly bool) {
	defer (*wg).Done()
LOOP:
	for {
		ip, ok := <-chanIn
		if !ok {
			break LOOP
		}
		if ip == workerStopSignal {
			//log.Println("Download task finished!")
			break LOOP
		}
		Ip := net.ParseIP(ip)
		// push an element to chanOnGoing, means that there is a test ongoing.
		chanOnGoing <- workOnGoing
		tResultSlice := downloadHandler(Ip, tcpport, tUrl, HttpRspTimeoutDuration, dltMaxDuration, dltCount, interval, dtOnly)
		tVerifyResult := singleVerifyResult{time.Now(), Ip, tResultSlice}
		chanOut <- tVerifyResult
		// pull out an element from chanOnGoing, means that a test work is finished.
		<-chanOnGoing
		// nanrrowed the gap between two different task by controlerInterval
		time.Sleep(time.Duration(controlerInterval) * time.Millisecond)
	}
}

func sslDTHandler(ip net.IP, hostName string, tcpPort int, dtTimeoutDuration time.Duration,
	totalRound int, interval int) []singleResult {
	conf := &tls.Config{
		ServerName: hostName,
	}
	fullAddress := ip.String()
	if ip.To4() == nil && ip.To16() != nil {
		fullAddress = "[" + fullAddress + "]"
	}
	fullAddress += ":" + strconv.Itoa(tcpPort)
	var allResult = make([]singleResult, 0)
	// loop for test
	for i := 0; i < totalRound; i++ {
		var currentResult = singleResult{false, 0, 0, false, false, 0, 0}
		// pingection time duration begin:
		var timeStart = time.Now()
		conn, tErr := net.DialTimeout("tcp", fullAddress, dtTimeoutDuration)
		if tErr != nil {
			allResult = append(allResult, currentResult)
			time.Sleep(time.Duration(interval) * time.Millisecond)
			continue
		}
		pingTLS := tls.Client(conn, conf)
		tErr = pingTLS.Handshake()
		if tErr != nil {
			allResult = append(allResult, currentResult)
			time.Sleep(time.Duration(interval) * time.Millisecond)
			continue
		}
		currentResult.dTPassedCount = true
		currentResult.dTDuration = time.Since(timeStart)
		allResult = append(allResult, currentResult)
		_ = pingTLS.Close()
		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
	return allResult
}

func sslDTWorker(chanIn chan string, chanOut chan singleVerifyResult, chanOnGoing chan int, wg *sync.WaitGroup,
	hostName string, tcpport int, dtTimeoutDuration time.Duration, totalRound int, interval int) {
	defer (*wg).Done()
LOOP:
	for {
		ip, ok := <-chanIn
		if !ok {
			break LOOP
		}
		if ip == workerStopSignal {
			break LOOP
		}
		Ip := net.ParseIP(ip)
		// push an element to chanOnGoing, means that there is a test ongoing.
		chanOnGoing <- workOnGoing
		tResultSlice := sslDTHandler(Ip, hostName, tcpport, dtTimeoutDuration, totalRound, interval)
		tVerifyResult := singleVerifyResult{time.Now(), Ip, tResultSlice}
		chanOut <- tVerifyResult
		// pull out an element from chanOnGoing, means that a test work is finished.
		<-chanOnGoing
		// nanrrowed the gap between two different task by controlerInterval
		time.Sleep(time.Duration(controlerInterval) * time.Millisecond)
	}
}
