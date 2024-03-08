package main

import (
	"context"
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"
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
func downloadHandler(ip net.IP, tUrl *string, httpRspTimeoutDuration time.Duration, dltTimeDurationMax time.Duration,
	dltCount int, interval int, dtOnly, evaluationDT bool) []singleResult {
	_, port := ParseUrl(*tUrl)
	fullAddress := getConnPeerAddress(ip, port)
	var allResult = make([]singleResult, 0)
	// loop for test
	for i := 0; i < dltCount; i++ {
		tReq, err := http.NewRequest("GET", *tUrl, nil)
		if err != nil {
			log.Fatal(err)
		}
		var tResultHttp ResultHttp
		tCtx := WithHTTPStat(tReq.Context(), &tResultHttp)
		tReq = tReq.WithContext(tCtx)
		// set user agent
		tReq.Header.Set("User-Agent", userAgent)
		var client = http.Client{
			Transport: &http.Transport{
				DialContext: GetDialContextByAddr(fullAddress),
				//ResponseHeaderTimeout: HttpRspTimeoutDuration,
			},
			CheckRedirect: nil,
			Jar:           nil,
			Timeout:       httpRspTimeoutDuration,
		}
		if !dtOnly && dltTimeDurationMax > httpRspTimeoutDuration {
			client.Timeout = dltTimeDurationMax
		}
		var currentResult = singleResult{false, 0, 0, false, false, 0, 0}
		response, err := client.Do(tReq)
		// connection is failed(network error), won't continue
		if err != nil {
			allResult = append(allResult, currentResult)
			time.Sleep(time.Duration(interval) * time.Millisecond)
			continue
		}
		currentResult.dTPassed = true
		currentResult.dTDuration = tResultHttp.tlsEndAt.Sub(tResultHttp.tcpStartAt)
		currentResult.httpReqRspDur = tResultHttp.httpRspAt.Sub(tResultHttp.httpReqAt)
		// connection test only, won't do download test
		if dtOnly {
			allResult = append(allResult, currentResult)
			time.Sleep(time.Duration(interval) * time.Millisecond)
			// if we need evaluate DT, we'll try DT as many as possible
			// if we don't, we'll stop after the first successfull try
			response.Body.Close()
			if evaluationDT {
				continue
			} else {
				break
			}
		}
		// if download test permitted, set DownloadPerformed to true
		currentResult.dLTWasDone = true
		// connection is not make(uri error or server error), won't do download test
		if response.StatusCode != 200 {
			allResult = append(allResult, currentResult)
			time.Sleep(time.Duration(interval) * time.Millisecond)
			continue
		}
		// start timing for download test
		tResultHttp.bodyReadStartAt = time.Now()
		timeEndExpected := tResultHttp.bodyReadStartAt.Add(dltTimeDurationMax)
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
			// there is an error shown and it's not io.EOF(read ended)
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
	// just get the last record in allResult while enable dtOnly and disable enableDTEvaluation
	if dtOnly && !enableDTEvaluation {
		allResult = allResult[len(allResult)-1:]
	}
	return allResult
}

func downloadWorker(chanIn chan *string, chanOut chan singleVerifyResult, chanOnGoing chan int, wg *sync.WaitGroup,
	tUrl *string, httpRspTimeoutDuration time.Duration, dltTimeDurationMax time.Duration,
	dltCount int, interval int, dtOnly, evaluationDT bool) {
	defer (*wg).Done()
LOOP:
	for {
		ip, ok := <-chanIn
		if !ok {
			break LOOP
		}
		if *ip == workerStopSignal {
			//log.Println("Download task finished!")
			break LOOP
		}
		// push an element to chanOnGoing, means that there is a test ongoing.
		// push it immediately once we confirm it's a normal task
		chanOnGoing <- workOnGoing
		Ip := net.ParseIP(*ip)
		tResultSlice := downloadHandler(Ip, tUrl, httpRspTimeoutDuration, dltTimeDurationMax, dltCount, interval, dtOnly, evaluationDT)
		tVerifyResult := singleVerifyResult{time.Now(), Ip, tResultSlice}
		chanOut <- tVerifyResult
		// pull out an element from chanOnGoing, means that a test work is finished.
		<-chanOnGoing
		// narrowed the gap between two different task by controllerInterval
		time.Sleep(time.Duration(controllerInterval) * time.Millisecond)
	}
}

func sslDTHandler(ip net.IP, hostName *string, port int, dtTimeoutDuration time.Duration,
	totalRound int, interval int, evaluateDT bool) []singleResult {
	conf := &tls.Config{
		ServerName: *hostName,
	}
	dialer := net.Dialer{Timeout: dtTimeoutDuration}
	fullAddress := getConnPeerAddress(ip, port)
	var allResult = make([]singleResult, 0)
	// loop for test
	for i := 0; i < totalRound; i++ {
		var currentResult = singleResult{false, 0, 0, false, false, 0, 0}
		// connection time duration begin:
		var timeStart = time.Now()
		// conn, tErr := net.DialTimeout("tcp", fullAddress, dtTimeoutDuration)
		conn, tErr := tls.DialWithDialer(&dialer, "tcp", fullAddress, conf)
		tDur := time.Since(timeStart)
		if tErr != nil {
			allResult = append(allResult, currentResult)
			time.Sleep(time.Duration(interval) * time.Millisecond)
			continue
		}
		currentResult.dTPassed = true
		currentResult.dTDuration = tDur
		allResult = append(allResult, currentResult)
		_ = conn.Close()
		// if we don't evaluate DT, we'll stop DT after first successful DT finished.
		if !evaluateDT {
			break
		}
		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
	// we just get the last record in all allResult while we diable enableDTEvaluation
	if !enableDTEvaluation {
		allResult = allResult[len(allResult)-1:]
	}
	return allResult
}

func sslDTWorker(chanIn chan *string, chanOut chan singleVerifyResult, chanOnGoing chan int, wg *sync.WaitGroup,
	hostName *string, port int, dtTimeoutDuration time.Duration, totalRound int, interval int, evaluateDT bool) {
	defer (*wg).Done()
LOOP:
	for {
		ip, ok := <-chanIn
		if !ok {
			break LOOP
		}
		if *ip == workerStopSignal {
			break LOOP
		}
		// push an element to chanOnGoing, means that there is a test ongoing.
		// push it immediately once we confirm it's a normal task
		chanOnGoing <- workOnGoing
		Ip := net.ParseIP(*ip)
		tResultSlice := sslDTHandler(Ip, hostName, port, dtTimeoutDuration, totalRound, interval, evaluateDT)
		tVerifyResult := singleVerifyResult{time.Now(), Ip, tResultSlice}
		chanOut <- tVerifyResult
		// pull out an element from chanOnGoing, means that a test work is finished.
		<-chanOnGoing
		// narrowed the gap between two different task by controllerInterval
		time.Sleep(time.Duration(controllerInterval) * time.Millisecond)
	}
}
