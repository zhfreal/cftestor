package main

import (
	"context"
	"io"
	"math"
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
	round int, doDTOnly bool, max_failure int) ([]singleResult, string) {
	var loc = ""
	var allResult = make([]singleResult, 0, round)
	_, port, err := net.SplitHostPort(*host)
	if err != nil {
		return allResult, ""
	}
	new_url, err := newUrl(*tUrl, port)
	if err != nil {
		myLogger.Errorf("failed to build test URL for %s: %v\n", *host, err)
		return allResult, ""
	}
	applyNoCache := shouldApplyNoCache(*tUrl)
	t_failure_counter := 0

	for i := 0; i < round; i++ {
		currentResult, rLoc := performDownloadRound(*host, new_url, httpRspTimeoutDur, doDTOnly, applyNoCache)

		if !currentResult.dTPassed || (!doDTOnly && currentResult.dLTWasDone && !currentResult.dLTPassed) {
			t_failure_counter++
		}
		if rLoc != "" && loc == "" {
			loc = rLoc
		}
		allResult = append(allResult, currentResult)

		// Early exit logic
		if doDTOnly && !Config.EnableDTEvaluation && currentResult.dTPassed {
			break
		}

		if (Config.EnableDTEvaluation || !doDTOnly) && t_failure_counter > max_failure {
			break
		}

		if i < round-1 {
			time.Sleep(time.Duration(Config.Interval) * time.Millisecond)
		}
	}

	if doDTOnly && !Config.EnableDTEvaluation && len(allResult) > 0 {
		allResult = allResult[len(allResult)-1:]
	}
	return allResult, loc
}

func performDownloadRound(host, targetUrl string, httpRspTimeoutDur time.Duration, doDTOnly, applyNoCache bool) (singleResult, string) {
	var currentResult = singleResult{false, 0, 0, false, false, 0, 0}
	var loc = ""

	tReq, err := http.NewRequest("GET", targetUrl, nil)
	if err != nil {
		return currentResult, ""
	}
	tReq.Header.Set("User-Agent", Config.UserAgent)
	if applyNoCache {
		tReq.Header.Set("Cache-Control", "no-cache")
		tReq.Header.Set("Pragma", "no-cache")
	}

	t_timeout := httpRspTimeoutDur
	if !doDTOnly && Config.DLTDurationInTotal > httpRspTimeoutDur {
		t_timeout = Config.DLTDurationInTotal
	}

	client, tr := newHttpClient(Config.TLSClientID, host, t_timeout)
	defer tr.CloseIdleConnections()

	ctx, cancel := context.WithTimeout(context.Background(), t_timeout)
	defer cancel()
	tReq = tReq.WithContext(ctx)

	response, err := client.Do(tReq)
	if err != nil || response == nil {
		return currentResult, ""
	}
	defer response.Body.Close()

	// Resolve location if it's a trace request
	if response.Request.URL.Path == "/cdn-cgi/trace" && response.StatusCode == 200 {
		loc, _ = get_loc_from_cf_resp(response.Body)
	}

	if doDTOnly {
		if response.StatusCode == Config.DTHttpRspReturnCodeExpected {
			currentResult.dTPassed = true
			currentResult.dTDuration, currentResult.httpReqRspDur = tr.Stat()
		}
		return currentResult, loc
	}

	currentResult.dLTWasDone = true
	if response.StatusCode != 200 {
		return currentResult, loc
	}

	currentResult.dTPassed = true
	currentResult.dTDuration, currentResult.httpReqRspDur = tr.Stat()

	// Download test phase
	readAt := time.Now()
	timeEndExpected := readAt.Add(Config.DLTDurationInTotal)
	contentLength := response.ContentLength
	if contentLength <= 0 {
		contentLength = fileDefaultSize
	}

	// Use a larger buffer (128KB) for download to maximize throughput
	buffer := make([]byte, 128*1024)
	var contentRead int64
	for contentRead < contentLength && time.Now().Before(timeEndExpected) {
		n, tErr := response.Body.Read(buffer)
		contentRead += int64(n)
		if n > 0 {
			currentResult.dLTPassed = true
		}
		if tErr != nil {
			if tErr == io.EOF {
				currentResult.dLTPassed = true
			} else if nErr, ok := tErr.(net.Error); ok && nErr.Timeout() {
				if contentRead > 0 {
					currentResult.dLTPassed = true
				}
			} else {
				currentResult.dLTPassed = false
			}
			break
		}
	}

	currentResult.dLTDuration = time.Since(readAt)
	currentResult.dLTDataSize = contentRead
	return currentResult, loc
}

func downloadWorkerNew(chanIn chan *task, chanOut chan singleVerifyResult, wg *sync.WaitGroup, tUrl *string,
	httpRspTimeoutDur time.Duration, round int, doDTOnly bool) {
	defer (*wg).Done()
	// max_failure := 0
	// if doDTOnly {
	// 	if !Config.EnableDTEvaluation {
	// 		max_failure = Config.DTCount
	// 	} else {
	// 		max_failure = get_max_ev_dt_failure()
	// 	}
	// } else {
	// 	max_failure = Config.DLTCount
	// }
LOOP:
	for {
		t, ok := <-chanIn
		if !ok {
			break LOOP
		}
		host := t.GetHost()
		max_failure := t.GetMaxFailure()
		// if doDTOnly && LoopStatus.Ok() {
		// 	max_failure = Config.DTCount
		// }
		tResultSlice, tLoc := downloadHandlerNew(host, tUrl, httpRspTimeoutDur, round, doDTOnly, max_failure)
		tVerifyResult := singleVerifyResult{time.Now(), *host, tLoc, tResultSlice}
		chanOut <- tVerifyResult
		// narrowed the gap between two different task by controllerInterval
		// time.Sleep(time.Duration(controllerInterval) * time.Millisecond)
	}
}

func sslDTHandlerNew(host *string, max_failure int) []singleResult {
	var allResult = make([]singleResult, 0)
	// Config.Loop for test
	t_failure_counter := 0
	for i := 0; i < Config.DTCount; i++ {
		var currentResult = singleResult{false, 0, 0, false, false, 0, 0}
		// connection time duration begin:
		var timeStart = time.Now()
		// conn, tErr := net.DialTimeout("tcp", fullAddress, Config.DTTimeoutDuration)
		ok := performUtlsDial(*host, Config.HostName, Config.DTTimeoutDuration, Config.TLSClientID)
		tDur := time.Since(timeStart)
		if !ok {
			allResult = append(allResult, currentResult)
			t_failure_counter += 1
		} else {
			currentResult.dTPassed = true
			currentResult.dTDuration = tDur
			allResult = append(allResult, currentResult)
		}
		// if we don't evaluate DT, we'll stop DT after first successful DT finished.
		if !Config.EnableDTEvaluation || t_failure_counter > max_failure {
			break
		}
		time.Sleep(time.Duration(Config.Interval) * time.Millisecond)
	}
	// we just get the last record in all allResult while we disable Config.EnableDTEvaluation
	if !Config.EnableDTEvaluation {
		allResult = allResult[len(allResult)-1:]
	}
	return allResult
}

func sslDTWorkerNew(chanIn chan *task, chanOut chan singleVerifyResult, wg *sync.WaitGroup) {
	defer (*wg).Done()
	// max_failure := Config.DTCount
	// if Config.EnableDTEvaluation {
	// 	max_failure = get_max_ev_dt_failure()
	// }
LOOP:
	for {
		t, ok := <-chanIn
		if !ok {
			break LOOP
		}
		// if LoopStatus.Ok() {
		// 	max_failure = Config.DTCount
		// }
		host := t.GetHost()
		max_failure := t.GetMaxFailure()
		tResultSlice := sslDTHandlerNew(host, max_failure)
		tVerifyResult := singleVerifyResult{time.Now(), *host, "", tResultSlice}
		chanOut <- tVerifyResult
		// narrowed the gap between two different task by controllerInterval
		// time.Sleep(time.Duration(controllerInterval) * time.Millisecond)
	}
}

func get_max_ev_dt_failure() int {
	return int(math.Round(float64(Config.DTCount) * (1 - Config.DTEvaluationDTPR/100)))
}

// isDT: true for DT, false for DLT
func get_max_failure(isDT bool) int {
	if isDT {
		if Config.EnableDTEvaluation {
			return get_max_ev_dt_failure()
		}
		return Config.DTCount
	}
	return Config.DLTCount
}
