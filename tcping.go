package main

import (
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

// download test core
func downloadHandler(host, tUrl *string, httpRspTimeoutDuration time.Duration, dltTimeDurationMax time.Duration,
	dltCount int, interval int, dtOnly, evaluationDT bool) []singleResult {
	var allResult = make([]singleResult, 0)
	_, port, err := net.SplitHostPort(*host)
	// invalid host
	if err != nil {
		return allResult
	}
	new_url := NewUrl(*tUrl, port)
	// loop for test
	for i := 0; i < dltCount; i++ {
		tReq, err := http.NewRequest("GET", new_url, nil)
		if err != nil {
			log.Fatal(err)
		}
		// var tResultHttp ResultHttp
		// tCtx := WithHTTPStat(tReq.Context(), &tResultHttp)
		// tReq = tReq.WithContext(tCtx)
		// set user agent
		tReq.Header.Set("User-Agent", userAgent)
		client, tr := newHttpClient(tlsClientID, *host, httpRspTimeoutDuration)
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
		currentResult.dTDuration, currentResult.httpReqRspDur = tr.Stat()
		// connection test only, won't do download test
		if dtOnly {
			allResult = append(allResult, currentResult)
			time.Sleep(time.Duration(interval) * time.Millisecond)
			// if we need evaluate DT, we'll try DT as many as possible
			// if we don't, we'll stop after the first successfull try
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
		readAt := time.Now()
		timeEndExpected := readAt.Add(dltTimeDurationMax)
		contentLength := response.ContentLength
		if contentLength == -1 {
			contentLength = fileDefaultSize
		}
		buffer := make([]byte, downloadBufferSize)
		var contentRead int64 = 0
		var downloadSuccess = false
		// just read  the length of content which indicated in response and read before time expire
		var tTimer = 0
		defer response.Body.Close()
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
	if dtOnly && !enableDTEvaluation {
		allResult = allResult[len(allResult)-1:]
	}
	return allResult
}

func downloadWorker(chanIn chan *string, chanOut chan singleVerifyResult, chanOnGoing chan int, wg *sync.WaitGroup,
	tUrl *string, httpRspTimeoutDuration time.Duration, dltTimeDurationMax time.Duration,
	dltCount int, dtOnly, evaluationDT bool) {
	defer (*wg).Done()
LOOP:
	for {
		host, ok := <-chanIn
		if !ok {
			break LOOP
		}
		if *host == workerStopSignal {
			//log.Println("Download task finished!")
			break LOOP
		}
		// push an element to chanOnGoing, means that there is a test ongoing.
		// push it immediately once we confirm it's a normal task
		chanOnGoing <- workOnGoing
		tResultSlice := downloadHandler(host, tUrl, httpRspTimeoutDuration, dltTimeDurationMax, dltCount, interval, dtOnly, evaluationDT)
		tVerifyResult := singleVerifyResult{time.Now(), *host, tResultSlice}
		chanOut <- tVerifyResult
		// pull out an element from chanOnGoing, means that a test work is finished.
		<-chanOnGoing
		// narrowed the gap between two different task by controllerInterval
		time.Sleep(time.Duration(controllerInterval) * time.Millisecond)
	}
}

func sslDTHandler(host *string, hostName *string, dtTimeoutDuration time.Duration,
	totalRound int, interval int, evaluateDT bool) []singleResult {
	var allResult = make([]singleResult, 0)
	// loop for test
	for i := 0; i < totalRound; i++ {
		var currentResult = singleResult{false, 0, 0, false, false, 0, 0}
		// connection time duration begin:
		var timeStart = time.Now()
		// conn, tErr := net.DialTimeout("tcp", fullAddress, dtTimeoutDuration)
		ok := performUtlsDial(*host, *hostName, dtTimeoutDuration, tlsClientID)
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

func sslDTWorker(chanIn chan *string, chanOut chan singleVerifyResult, chanOnGoing chan int, wg *sync.WaitGroup, evaluateDT bool) {
	defer (*wg).Done()
LOOP:
	for {
		host, ok := <-chanIn
		if !ok {
			break LOOP
		}
		if *host == workerStopSignal {
			break LOOP
		}
		// push an element to chanOnGoing, means that there is a test ongoing.
		// push it immediately once we confirm it's a normal task
		chanOnGoing <- workOnGoing
		tResultSlice := sslDTHandler(host, &hostName, dtTimeoutDuration, dtCount, interval, evaluateDT)
		tVerifyResult := singleVerifyResult{time.Now(), *host, tResultSlice}
		chanOut <- tVerifyResult
		// pull out an element from chanOnGoing, means that a test work is finished.
		<-chanOnGoing
		// narrowed the gap between two different task by controllerInterval
		time.Sleep(time.Duration(controllerInterval) * time.Millisecond)
	}
}
