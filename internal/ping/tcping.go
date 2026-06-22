package ping

import (
	"bufio"
	"context"
	"io"
	"math"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"cftestor/internal/config"
	"cftestor/internal/logger"
	"cftestor/internal/utils"
)

func downloadHandlerNew(host, tUrl *string, httpRspTimeoutDur time.Duration,
	round int, doDTOnly bool, max_failure int) ([]config.SingleResult, string) {
	var loc = ""
	var allResult = make([]config.SingleResult, 0, round)
	_, port, err := net.SplitHostPort(*host)
	if err != nil {
		return allResult, ""
	}
	new_url, err := utils.NewUrl(*tUrl, port, config.DefaultDLTUrl)
	if err != nil {
		logger.Log.Errorf("failed to build test URL for %s: %v\n", *host, err)
		return allResult, ""
	}
	applyNoCache := config.ShouldApplyNoCache(*tUrl)
	t_failure_counter := 0

	for i := 0; i < round; i++ {
		currentResult, rLoc := performDownloadRound(*host, new_url, httpRspTimeoutDur, doDTOnly, applyNoCache)

		if !currentResult.DTPassed || (!doDTOnly && currentResult.DLTWasDone && !currentResult.DLTPassed) {
			t_failure_counter++
		}
		if rLoc != "" && loc == "" {
			loc = rLoc
		}
		allResult = append(allResult, currentResult)

		if doDTOnly && !config.Config.EnableDTEvaluation && currentResult.DTPassed {
			break
		}

		if (config.Config.EnableDTEvaluation || !doDTOnly) && t_failure_counter > max_failure {
			break
		}

		if i < round-1 {
			time.Sleep(time.Duration(config.Config.Interval) * time.Millisecond)
		}
	}

	if doDTOnly && !config.Config.EnableDTEvaluation && len(allResult) > 0 {
		allResult = allResult[len(allResult)-1:]
	}
	return allResult, loc
}

func performDownloadRound(host, targetUrl string, httpRspTimeoutDur time.Duration, doDTOnly, applyNoCache bool) (config.SingleResult, string) {
	var currentResult = config.SingleResult{
		DTPassed:      false,
		DTDuration:    0,
		HttpReqRspDur: 0,
		DLTWasDone:    false,
		DLTPassed:     false,
		DLTDuration:   0,
		DLTDataSize:   0,
	}
	var loc = ""

	tReq, err := http.NewRequest("GET", targetUrl, nil)
	if err != nil {
		return currentResult, ""
	}
	tReq.Header.Set("User-Agent", config.Config.UserAgent)
	if applyNoCache {
		tReq.Header.Set("Cache-Control", "no-cache")
		tReq.Header.Set("Pragma", "no-cache")
	}

	t_timeout := httpRspTimeoutDur
	if !doDTOnly && config.Config.DLTDurationInTotal > httpRspTimeoutDur {
		t_timeout = config.Config.DLTDurationInTotal
	}

	client, tr := NewHttpClient(config.Config.TLSClientID, host, t_timeout)
	defer tr.CloseIdleConnections()

	ctx, cancel := context.WithTimeout(context.Background(), t_timeout)
	defer cancel()
	tReq = tReq.WithContext(ctx)

	response, err := client.Do(tReq)
	if err != nil || response == nil {
		return currentResult, ""
	}
	defer response.Body.Close()

	if response.Request.URL.Path == "/cdn-cgi/trace" && response.StatusCode == 200 {
		loc, _ = getLocFromCFResp(response.Body)
	}

	if doDTOnly {
		if response.StatusCode == config.Config.DTHttpRspReturnCodeExpected {
			currentResult.DTPassed = true
			currentResult.DTDuration, currentResult.HttpReqRspDur = tr.Stat()
		}
		return currentResult, loc
	}

	currentResult.DLTWasDone = true
	if response.StatusCode != 200 {
		return currentResult, loc
	}

	currentResult.DTPassed = true
	currentResult.DTDuration, currentResult.HttpReqRspDur = tr.Stat()

	readAt := time.Now()
	timeEndExpected := readAt.Add(config.Config.DLTDurationInTotal)
	contentLength := response.ContentLength
	if contentLength <= 0 {
		contentLength = config.FileDefaultSize
	}

	buffer := make([]byte, 128*1024)
	var contentRead int64
	for contentRead < contentLength && time.Now().Before(timeEndExpected) {
		n, tErr := response.Body.Read(buffer)
		contentRead += int64(n)
		if n > 0 {
			currentResult.DLTPassed = true
		}
		if tErr != nil {
			if tErr == io.EOF {
				currentResult.DLTPassed = true
			} else if nErr, ok := tErr.(net.Error); ok && nErr.Timeout() {
				if contentRead > 0 {
					currentResult.DLTPassed = true
				}
			} else {
				currentResult.DLTPassed = false
			}
			break
		}
	}

	currentResult.DLTDuration = time.Since(readAt)
	currentResult.DLTDataSize = contentRead
	return currentResult, loc
}

func getLocFromCFResp(body io.Reader) (string, error) {
	loc := ""
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		t_slice := strings.Split(line, "=")
		if len(t_slice) != 2 {
			continue
		}
		if strings.ToLower(t_slice[0]) == "colo" {
			loc = strings.ToUpper(t_slice[1])
			if len(loc) > 3 {
				loc = loc[:3]
			}
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	t, ok := utils.IataMap[loc]
	if ok {
		return t, nil
	} else {
		return loc, nil
	}
}

func DownloadWorkerNew(chanIn chan *config.Task, chanOut chan config.SingleVerifyResult, wg *sync.WaitGroup, tUrl *string,
	httpRspTimeoutDur time.Duration, round int, doDTOnly bool) {
	defer wg.Done()
LOOP:
	for {
		t, ok := <-chanIn
		if !ok {
			break LOOP
		}
		host := t.GetHost()
		max_failure := t.GetMaxFailure()
		tResultSlice, tLoc := downloadHandlerNew(host, tUrl, httpRspTimeoutDur, round, doDTOnly, max_failure)
		tVerifyResult := config.SingleVerifyResult{
			TestTime:    time.Now(),
			Host:        *host,
			Loc:         tLoc,
			ResultSlice: tResultSlice,
		}
		chanOut <- tVerifyResult
	}
}

func sslDTHandlerNew(host *string, max_failure int) []config.SingleResult {
	var allResult = make([]config.SingleResult, 0)
	t_failure_counter := 0
	for i := 0; i < config.Config.DTCount; i++ {
		var currentResult = config.SingleResult{
			DTPassed:      false,
			DTDuration:    0,
			HttpReqRspDur: 0,
			DLTWasDone:    false,
			DLTPassed:     false,
			DLTDuration:   0,
			DLTDataSize:   0,
		}
		var timeStart = time.Now()
		ok := PerformUtlsDial(*host, config.Config.HostName, config.Config.DTTimeoutDuration, config.Config.TLSClientID)
		tDur := time.Since(timeStart)
		if !ok {
			allResult = append(allResult, currentResult)
			t_failure_counter += 1
		} else {
			currentResult.DTPassed = true
			currentResult.DTDuration = tDur
			allResult = append(allResult, currentResult)
		}
		if !config.Config.EnableDTEvaluation || t_failure_counter > max_failure {
			break
		}
		time.Sleep(time.Duration(config.Config.Interval) * time.Millisecond)
	}
	if !config.Config.EnableDTEvaluation {
		allResult = allResult[len(allResult)-1:]
	}
	return allResult
}

func SslDTWorkerNew(chanIn chan *config.Task, chanOut chan config.SingleVerifyResult, wg *sync.WaitGroup) {
	defer wg.Done()
LOOP:
	for {
		t, ok := <-chanIn
		if !ok {
			break LOOP
		}
		host := t.GetHost()
		max_failure := t.GetMaxFailure()
		tResultSlice := sslDTHandlerNew(host, max_failure)
		tVerifyResult := config.SingleVerifyResult{
			TestTime:    time.Now(),
			Host:        *host,
			Loc:         "",
			ResultSlice: tResultSlice,
		}
		chanOut <- tVerifyResult
	}
}

func GetMaxEvDTFailure() int {
	return int(math.Round(float64(config.Config.DTCount) * (1 - config.Config.DTEvaluationDTPR/100)))
}

func GetMaxFailure(isDT bool) int {
	if isDT {
		if config.Config.EnableDTEvaluation {
			return GetMaxEvDTFailure()
		}
		return config.Config.DTCount
	}
	return config.Config.DLTCount
}
