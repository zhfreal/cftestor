package main

import (
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

// type ResultHttp struct {
// 	dnsStartAt      time.Time
// 	dnsEndAt        time.Time
// 	tcpStartAt      time.Time
// 	tcpEndAt        time.Time
// 	tlsStartAt      time.Time
// 	tlsEndAt        time.Time
// 	httpReqAt       time.Time
// 	httpRspAt       time.Time
// 	bodyReadStartAt time.Time
// 	bodyReadEndAt   time.Time
// }

// func WithHTTPStat(ctx context.Context, r *ResultHttp) context.Context {
// 	return httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
// 		DNSStart: func(i httptrace.DNSStartInfo) {
// 			r.dnsStartAt = time.Now()
// 		},
// 		DNSDone: func(i httptrace.DNSDoneInfo) {
// 			r.dnsEndAt = time.Now()
// 		},
// 		ConnectStart: func(_, _ string) {
// 			r.tcpStartAt = time.Now()
// 			// When connecting to IP (When no DNS lookup)
// 			if r.dnsStartAt.IsZero() {
// 				r.dnsStartAt = r.tcpStartAt
// 				r.dnsEndAt = r.tcpStartAt
// 			}
// 		},
// 		ConnectDone: func(network, addr string, err error) {
// 			r.tcpEndAt = time.Now()
// 		},
// 		TLSHandshakeStart: func() {
// 			r.tlsStartAt = time.Now()
// 		},
// 		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
// 			r.tlsEndAt = time.Now()
// 		},
// 		WroteRequest: func(info httptrace.WroteRequestInfo) {
// 			r.httpReqAt = time.Now()
// 			if r.dnsStartAt.IsZero() && r.tcpStartAt.IsZero() {
// 				now := r.httpReqAt
// 				r.dnsStartAt = now
// 				r.dnsEndAt = now
// 				r.tcpStartAt = now
// 				r.tcpEndAt = now
// 			}
// 		},
// 		GotFirstResponseByte: func() {
// 			r.httpRspAt = time.Now()
// 			r.bodyReadStartAt = r.httpRspAt
// 		},
// 	})
// }

func downloadHandlerNew(host, tUrl *string, httpRspTimeoutDur time.Duration,
	round int, doDTOnly bool) []singleResult {
	var allResult = make([]singleResult, 0)
	_, port, err := net.SplitHostPort(*host)
	// invalid host
	if err != nil {
		return allResult
	}
	new_url := newUrl(*tUrl, port)
	// loop for test
	for i := 0; i < round; i++ {
		tReq, err := http.NewRequest("GET", new_url, nil)
		if err != nil {
			log.Fatal(err)
		}
		// var tResultHttp ResultHttp
		// tCtx := WithHTTPStat(tReq.Context(), &tResultHttp)
		// tReq = tReq.WithContext(tCtx)
		// set user agent
		tReq.Header.Set("User-Agent", userAgent)
		t_timeout := httpRspTimeoutDur
		client, tr := newHttpClient(tlsClientID, *host, t_timeout)
		if !doDTOnly && dltDurationInTotal > httpRspTimeoutDur {
			t_timeout = dltDurationInTotal
		}
		client.Timeout = t_timeout
		ctx, cancel := context.WithTimeout(context.Background(), t_timeout)
		tReq = tReq.WithContext(ctx)
		var currentResult = singleResult{false, 0, 0, false, false, 0, 0}
		response, err := client.Do(tReq)
		defer func() {
			cancel()
			if response != nil && response.Body != nil {
				response.Body.Close()
			}
			tr.CloseIdleConnections()
		}()
		// connection is failed(network error), won't continue
		if err != nil {
			allResult = append(allResult, currentResult)
			time.Sleep(time.Duration(interval) * time.Millisecond)
			continue
		}

		// connection test only, won't do download test
		if doDTOnly {
			if response.StatusCode == dtHttpRspReturnCodeExpected {
				currentResult.dTPassed = true
				currentResult.dTDuration, currentResult.httpReqRspDur = tr.Stat()
			}
			allResult = append(allResult, currentResult)
			time.Sleep(time.Duration(interval) * time.Millisecond)
			// if we need evaluate DT, we'll try DT as many as possible
			// if we don't, we'll stop after the first successfull try
			if enableDTEvaluation {
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
		currentResult.dTPassed = true
		currentResult.dTDuration, currentResult.httpReqRspDur = tr.Stat()
		// start timing for download test
		readAt := time.Now()
		timeEndExpected := readAt.Add(dltDurationInTotal)
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
			contentRead += int64(bufferRead)
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
				downloadSuccess = true
				break
			}
			tTimer += 1
			//myLogger.Debug(fmt.Sprintf("FullAddress: %s, Round %d, success for %3d", fullAddress, i, tTimer))
			downloadSuccess = true
		}
		currentResult.dLTPassed = downloadSuccess
		readEndAt := time.Now()
		currentResult.dLTDuration = readEndAt.Sub(readAt)
		currentResult.dLTDataSize = contentRead
		allResult = append(allResult, currentResult)
		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
	// just get the last record in allResult while enable dtOnly and disable enableDTEvaluation
	if doDTOnly && !enableDTEvaluation {
		allResult = allResult[len(allResult)-1:]
	}
	return allResult
}

func downloadWorkerNew(chanIn chan *string, chanOut chan singleVerifyResult, wg *sync.WaitGroup, tUrl *string,
	httpRspTimeoutDur time.Duration, round int, doDTOnly bool) {
	defer (*wg).Done()
LOOP:
	for {
		host, ok := <-chanIn
		if !ok {
			break LOOP
		}
		tResultSlice := downloadHandlerNew(host, tUrl, httpRspTimeoutDur, round, doDTOnly)
		tVerifyResult := singleVerifyResult{time.Now(), *host, tResultSlice}
		chanOut <- tVerifyResult
		// narrowed the gap between two different task by controllerInterval
		time.Sleep(time.Duration(controllerInterval) * time.Millisecond)
	}
}

func sslDTHandlerNew(host *string) []singleResult {
	var allResult = make([]singleResult, 0)
	// loop for test
	for i := 0; i < dtCount; i++ {
		var currentResult = singleResult{false, 0, 0, false, false, 0, 0}
		// connection time duration begin:
		var timeStart = time.Now()
		// conn, tErr := net.DialTimeout("tcp", fullAddress, dtTimeoutDuration)
		ok := performUtlsDial(*host, hostName, dtTimeoutDuration, tlsClientID)
		tDur := time.Since(timeStart)
		if !ok {
			allResult = append(allResult, currentResult)
			time.Sleep(time.Duration(interval) * time.Millisecond)
			continue
		}
		currentResult.dTPassed = true
		currentResult.dTDuration = tDur
		allResult = append(allResult, currentResult)
		// if we don't evaluate DT, we'll stop DT after first successful DT finished.
		if !enableDTEvaluation {
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

func sslDTWorkerNew(chanIn chan *string, chanOut chan singleVerifyResult, wg *sync.WaitGroup) {
	defer (*wg).Done()
LOOP:
	for {
		host, ok := <-chanIn
		if !ok {
			break LOOP
		}
		tResultSlice := sslDTHandlerNew(host)
		tVerifyResult := singleVerifyResult{time.Now(), *host, tResultSlice}
		chanOut <- tVerifyResult
		// narrowed the gap between two different task by controllerInterval
		time.Sleep(time.Duration(controllerInterval) * time.Millisecond)
	}
}
