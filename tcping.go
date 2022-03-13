package main

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Mine
func downloadHandler(ip net.IP, tcpport int, tUrl string, HttpRspTimeoutDuration time.Duration, downloadTimeMaxDuration time.Duration,
	DownloadTry int, interval int, testPingOnly bool) []SingleResultSlice {
	var fullAddress string
	if ip.To4() != nil { //IPv4
		fullAddress = ip.String() + ":" + strconv.Itoa(tcpport)
	} else { //
		fullAddress = "[" + ip.String() + "]:" + strconv.Itoa(tcpport)
	}

	var allResult = make([]SingleResultSlice, 0)
	// loop for test
	for i := 0; i < DownloadTry; i++ {
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
		if !testPingOnly {
			client.Timeout += downloadTimeMaxDuration
		}
		client.Transport = &http.Transport{
			DialContext: GetDialContextByAddr(fullAddress),
			//ResponseHeaderTimeout: HttpRspTimeoutDuration,
		}
		var currentResult = SingleResultSlice{false, 0, false, false, 0, 0}
		response, err := client.Do(tReq)
		// pingect is failed(network error), won't continue
		if err != nil {
			allResult = append(allResult, currentResult)
			time.Sleep(time.Duration(interval) * time.Millisecond)
			continue
		}
		currentResult.PingSuccess = true
		currentResult.PingTimeDuration = tResultHttp.httpRspAt.Sub(tResultHttp.tcpStartAt)
		// pingection test only, won't do download test
		if testPingOnly {
			allResult = append(allResult, currentResult)
			time.Sleep(time.Duration(interval) * time.Millisecond)
			continue
		}
		// if download test permitted, set DownloadPerformed to true
		currentResult.DownloadPerformed = true
		// pingect is not make(uri error or server error), won't do download test
		if response.StatusCode != 200 {
			allResult = append(allResult, currentResult)
			time.Sleep(time.Duration(interval) * time.Millisecond)
			continue
		}
		// start timing for download test
		tResultHttp.bodyReadStartAt = time.Now()
		timeEndExpected := tResultHttp.bodyReadStartAt.Add(downloadTimeMaxDuration)
		contentLength := response.ContentLength
		if contentLength == -1 {
			contentLength = FileDefaultSize
		}
		buffer := make([]byte, DownloadBufferSize)
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
		currentResult.DownloadSuccess = downloadSuccess
		tResultHttp.bodyReadEndAt = time.Now()
		currentResult.DownloadDuration = tResultHttp.bodyReadEndAt.Sub(tResultHttp.bodyReadStartAt)
		currentResult.DownloadSize = contentRead
		allResult = append(allResult, currentResult)
		_ = response.Body.Close()
		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
	return allResult
}

func DownloadWorker(chanIn chan string, chanOut chan SingleVerifyResult, wg *sync.WaitGroup,
	tUrl string, tcpport int, HttpRspTimeoutDuration time.Duration, downloadTimeMaxDuration time.Duration,
	DownloadTry int, interval int, testPingOnly bool) {
	defer wg.Done()
LOOP:
	for {
		select {
		case ip := <-chanIn:
			if ip == WorkerStopSignal {
				//log.Println("Download task finished!")
				break LOOP
			}
			Ip := net.ParseIP(ip)
			tResultSlice := downloadHandler(Ip, tcpport, tUrl, HttpRspTimeoutDuration, downloadTimeMaxDuration, DownloadTry, interval, testPingOnly)
			tVerifyResult := SingleVerifyResult{time.Now(), Ip, tResultSlice}
			chanOut <- tVerifyResult
			break
		}
		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
}

func tcppingHandler(ip net.IP, hostName string, tcpPort int, pingTimeoutDuration time.Duration,
	totalRound int, interval int) []SingleResultSlice {
	conf := &tls.Config{
		ServerName: hostName,
	}
	fullAddress := ip.String()
	if ip.To4() == nil && ip.To16() != nil {
		fullAddress = "[" + fullAddress + "]"
	}
	fullAddress += ":" + strconv.Itoa(tcpPort)
	var allResult = make([]SingleResultSlice, 0)
	// loop for test
	for i := 0; i < totalRound; i++ {
		var currentResult = SingleResultSlice{false, 0, false, false, 0, 0}
		// pingection time duration begin:
		var timeStart = time.Now()
		conn, tErr := net.DialTimeout("tcp", fullAddress, pingTimeoutDuration)
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
		currentResult.PingSuccess = true
		currentResult.PingTimeDuration = time.Since(timeStart)
		allResult = append(allResult, currentResult)
		_ = pingTLS.Close()
		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
	return allResult
}

func TcppingWorker(chanIn chan string, chanOut chan SingleVerifyResult, wg *sync.WaitGroup,
	hostName string, tcpport int, pingTimeoutDuration time.Duration, totalRound int, interval int) {
	defer wg.Done()
LOOP:
	for {
		select {
		case ip := <-chanIn:
			if ip == WorkerStopSignal {
				break LOOP
			}
			Ip := net.ParseIP(ip)
			tResultSlice := tcppingHandler(Ip, hostName, tcpport, pingTimeoutDuration, totalRound, interval)
			tVerifyResult := SingleVerifyResult{time.Now(), Ip, tResultSlice}
			chanOut <- tVerifyResult
			break
		}
		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
}
