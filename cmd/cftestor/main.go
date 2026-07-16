package main

import (
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"cftestor/internal/config"
	"cftestor/internal/db"
	"cftestor/internal/fetcher"
	"cftestor/internal/logger"
	"cftestor/internal/outbound"
	"cftestor/internal/ping"
	"cftestor/internal/utils"
)

func print_version() {
	config.PrintVersionInfo()
}

func validDTResult(tVerifyResult *config.VerifyResults) bool {
	if tVerifyResult.Da > 0.0 &&
		tVerifyResult.Da <= float64(config.Config.DTEvaluationDelay) &&
		tVerifyResult.Dtpr*100.0 >= float64(config.Config.DTEvaluationDTPR) &&
		(!config.Config.EnableStdEv || (config.Config.EnableStdEv && tVerifyResult.DaStd <= config.Config.DTStdExp)) {
		return true
	}
	return false
}

func validDLTResult(tVerifyResult *config.VerifyResults) bool {
	if tVerifyResult.Dls >= config.Config.DLTEvaluationSpeed && tVerifyResult.Dlds > config.DownloadSizeMin {
		return true
	}
	return false
}

var (
	dtTaskChan    chan *config.Task
	dtResultChan  chan config.SingleVerifyResult
	dltTaskChan   chan *config.Task
	dltResultChan chan config.SingleVerifyResult
	workerWG      sync.WaitGroup
)

func initWorkers() {
	if !config.Config.DLTOnly {
		dtTaskChan = make(chan *config.Task, config.Config.DTWorkerThread)
		dtResultChan = make(chan config.SingleVerifyResult, config.Config.DTWorkerThread)
		for range config.Config.DTWorkerThread {
			workerWG.Add(1)
			if config.Config.DTHttps {
				go ping.DownloadWorkerNew(dtTaskChan, dtResultChan, &workerWG, &config.Config.DTUrl, config.Config.DTTimeoutDuration, config.Config.DTCount, true)
			} else {
				go ping.SslDTWorkerNew(dtTaskChan, dtResultChan, &workerWG)
			}
		}
	}
	if !config.Config.DTOnly {
		dltTaskChan = make(chan *config.Task, config.Config.DLTWorkerThread)
		dltResultChan = make(chan config.SingleVerifyResult, config.Config.DLTWorkerThread)
		for range config.Config.DLTWorkerThread {
			workerWG.Add(1)
			go ping.DownloadWorkerNew(dltTaskChan, dltResultChan, &workerWG, &config.Config.DLTUrl, config.Config.HttpRspTimeoutDuration, config.Config.DLTCount, false)
		}
	}
}

func runDTSingleRound(ips []*string, handler func(config.SingleVerifyResult)) {
	size := len(ips)
	if size == 0 {
		return
	}
	max_failure := ping.GetMaxFailure(true)
	go func() {
		for _, ip := range ips {
			dtTaskChan <- config.NewTask(ip, max_failure)
		}
	}()

	for range size {
		handler(<-dtResultChan)
	}
}

func runDLTSingleRound(ips []*string, handler func(config.SingleVerifyResult)) {
	size := len(ips)
	if size == 0 {
		return
	}
	max_failure := ping.GetMaxFailure(false)
	go func() {
		for _, ip := range ips {
			dltTaskChan <- config.NewTask(ip, max_failure)
		}
	}()

	for range size {
		handler(<-dltResultChan)
	}
}

func calcResult(out config.SingleVerifyResult, statDownload bool) config.VerifyResults {
	var tVerifyResult = config.VerifyResults{}
	tVerifyResult.DtDList = make([]float64, 0)
	tVerifyResult.TestTime = out.TestTime
	tIP := out.Host
	tVerifyResult.IP = &tIP
	tVerifyResult.Loc = &out.Loc
	if len(out.ResultSlice) == 0 {
		return tVerifyResult
	}
	tVerifyResult.Dtc = len(out.ResultSlice)
	var tDurationsAll = 0.0
	for _, v := range out.ResultSlice {
		if v.DTPassed {
			tVerifyResult.Dtpc += 1
			tDuration := float64(v.DTDuration) / float64(time.Millisecond)
			if config.Config.DTHttps {
				tDuration += float64(v.HttpReqRspDur) / float64(time.Millisecond)
			}
			tVerifyResult.DtDList = append(tVerifyResult.DtDList, tDuration)
			tDurationsAll += tDuration
			if tDuration > tVerifyResult.Dmx {
				tVerifyResult.Dmx = tDuration
			}
			if tVerifyResult.Dmi <= 0.0 || tDuration < tVerifyResult.Dmi {
				tVerifyResult.Dmi = tDuration
			}
			if statDownload {
				tVerifyResult.Dltc += 1
				if v.DLTWasDone && v.DLTPassed {
					tVerifyResult.Dltpc += 1
					tVerifyResult.Dltd += float64(v.DLTDuration) / float64(time.Second)
					tVerifyResult.Dlds += v.DLTDataSize
				}
			}
		}
	}
	if tVerifyResult.Dtpc > 0 {
		tVerifyResult.Da = tDurationsAll / float64(tVerifyResult.Dtpc)
		tVerifyResult.Dtpr = float64(tVerifyResult.Dtpc) / float64(tVerifyResult.Dtc)
		if config.Config.EnableStdEv {
			tVerifyResult.DaVar = utils.Variance(tVerifyResult.DtDList)
			tVerifyResult.DaStd = utils.Std(tVerifyResult.DtDList)
		}
	}
	if statDownload {
		if tVerifyResult.Dltpc > 0 && tVerifyResult.Dlds > config.DownloadSizeMin {
			tVerifyResult.Dltpr = float64(tVerifyResult.Dltpc) / float64(tVerifyResult.Dltc)
			tVerifyResult.Dls = float64(tVerifyResult.Dlds) / tVerifyResult.Dltd / 1000
		}
	}
	return tVerifyResult
}

func printDetails(logLvl logger.LogLevel, v []config.VerifyResults, showSpeed bool) {
	if len(v) == 0 {
		return
	}
	if logger.Log.LoggerLevel&logLvl != logLvl {
		return
	}
	indent := logger.Log.Indent
	if len(indent) == 0 {
		indent = " "
	}
	for i := 0; i < len(v); i++ {
		t_ip := *v[i].IP
		if len(*v[i].Loc) > 0 {
			t_ip = fmt.Sprintf("%s#%s", t_ip, *v[i].Loc)
		}
		msg := fmt.Sprintf("IP:%v%s", t_ip, indent)
		if showSpeed {
			msg += fmt.Sprintf("Spd:%.2f%s", v[i].Dls, indent)
		}
		msg += fmt.Sprintf("Dly:%.0f", v[i].Da)
		msg += fmt.Sprintf("%sStb:%.2f", indent, v[i].Dtpr*100)
		if config.Config.EnableStdEv {
			msg += fmt.Sprintf("%sStd:%.2f", indent, v[i].DaStd)
		}
		logger.Log.Logf(logLvl, "%s", msg)
	}
}

func displayDetails(showSpeed, loopEnabled bool, v []config.VerifyResults) {
	if config.Config.Debug {
		printDetails(logger.LogLevelDebug, v, showSpeed)
	} else {
		if config.Config.SilenceMode {
			if !loopEnabled {
				for _, t_v := range v {
					tStr := *t_v.IP
					if t_v.Loc != nil && len(*t_v.Loc) > 0 {
						tStr = fmt.Sprintf("%s#%s", tStr, *t_v.Loc)
					}
					logger.Log.Println(tStr)
				}
			}
		} else {
			printDetails(logger.LogLevelInfo, v, showSpeed)
		}
	}
}

func displayStat(ov config.OverAllStat) {
	if logger.Log.LoggerLevel&logger.LogLevelDebug != logger.LogLevelDebug {
		return
	}
	logger.Log.Printf("==== Res: %d ==== ", ov.ResultCount)
	srcCount := ov.Remain
	if !config.Config.DLTOnly {
		dtTotal := ov.DtCached + ov.DtTasksDone + ov.DtOnGoing
		if config.Config.DTOnly {
			dtTotal += srcCount
		}
		logger.Log.Printf(" DT:%d/%d ", ov.DtTasksDone, dtTotal)
	}
	if !config.Config.DTOnly {
		dltTotal := ov.DltCached + ov.DltTasksDone + ov.DltOnGoing
		if config.Config.DLTOnly {
			dltTotal += srcCount
		}
		logger.Log.Printf(" DLT:%d/%d ", ov.DltTasksDone, dltTotal)
	}
	logger.Log.Println("")
}

func runWorker() {
	initWorkers()

	var thisSourceIPs = config.SrcIPs
	var t_result_min = config.Config.ResultMin
	var start_time = time.Now()

	// Determine starting IP source level
	hasUserSources := len(config.IPStr) > 0 || len(config.Config.IPFile) > 0
	currentSourceLevel := config.SourceLevelFull
	if hasUserSources {
		currentSourceLevel = config.SourceLevelUser
	} else if config.Config.FastMode {
		currentSourceLevel = config.SourceLevelFast
	}

	var tMode int8 = 0
	if config.Config.IPv4Mode {
		tMode |= config.TypeIPv4
	}
	if config.Config.IPv6Mode {
		tMode |= config.TypeIPv6
	}

RETRY_LOOP:
	for {
		tmpResultMap := make(map[string]config.VerifyResults)
		var tmpTestSlice map[string]bool
		looper := config.NewSafeLooperWithInterval(config.Config.Loop, config.Config.LoopInterval*1000)
	LOOP:
		for {
			dtDoneTasks := 0
			dltDoneTasks := 0
			tmpTestSlice = make(map[string]bool)

		SINGLE_ROUND:
			for {
				if time.Since(start_time) >= time.Duration(config.Config.TestTimeout)*time.Minute {
					break SINGLE_ROUND
				}

				if !config.Config.DLTOnly {
					dtBatch := thisSourceIPs.RetrieveSome(config.Config.DTWorkerThread, !config.Config.TestAll)
					if len(dtBatch) == 0 {
						break SINGLE_ROUND
					}

					dltBatch := make([]*string, 0)
					cachedMap := make(map[string]config.VerifyResults)

					runDTSingleRound(dtBatch, func(dtRes config.SingleVerifyResult) {
						dtDoneTasks++
						tVerifyResult := calcResult(dtRes, false)
						t_ip := *tVerifyResult.IP

						if validDTResult(&tVerifyResult) {
							if !config.Config.DTOnly {
								cachedMap[t_ip] = tVerifyResult
								dltBatch = append(dltBatch, &t_ip)
								if config.Config.Debug {
									displayDetails(false, looper.Status() > -1, []config.VerifyResults{tVerifyResult})
								}
							} else {
								if config.Config.ResolveLoc && config.Config.SilenceMode && looper.Status() == -1 && (tVerifyResult.Loc == nil || len(*tVerifyResult.Loc) == 0) {
									loc := outbound.GetGeoInfoFromCF(tVerifyResult.IP)
									tVerifyResult.Loc = &loc
								}
								v, ok := tmpResultMap[t_ip]
								if ok {
									tVerifyResult.Combine(v)
								}
								tmpResultMap[t_ip] = tVerifyResult
								displayDetails(false, looper.Status() > -1, []config.VerifyResults{tVerifyResult})
								tmpTestSlice[t_ip] = true
							}
						} else {
							if looper.InLooping() {
								v, ok := tmpResultMap[t_ip]
								if ok {
									tVerifyResult.Combine(v)
								}
								tmpResultMap[t_ip] = tVerifyResult
							}
							if config.Config.Debug {
								displayDetails(false, looper.Status() > -1, []config.VerifyResults{tVerifyResult})
							}
						}
					})

					if config.Config.Debug && !config.Config.DTOnly {
						displayStat(config.OverAllStat{
							DtTasksDone:  dtDoneTasks,
							DltTasksDone: dltDoneTasks,
							ResultCount:  len(tmpTestSlice),
							Remain:       thisSourceIPs.LenInt(),
						})
					}

					if !config.Config.DTOnly && len(dltBatch) > 0 {
						runDLTSingleRound(dltBatch, func(dltRes config.SingleVerifyResult) {
							dltDoneTasks++
							tVerifyResult := calcResult(dltRes, true)
							t_ip := *tVerifyResult.IP
							v := cachedMap[t_ip]
							tVerifyResult.Combine(v)

							if validDLTResult(&tVerifyResult) && validDTResult(&tVerifyResult) {
								if config.Config.ResolveLoc && config.Config.SilenceMode && looper.Status() == -1 && (tVerifyResult.Loc == nil || len(*tVerifyResult.Loc) == 0) {
									loc := outbound.GetGeoInfoFromCF(&t_ip)
									tVerifyResult.Loc = &loc
								}
								mv, ok := tmpResultMap[t_ip]
								if ok {
									tVerifyResult.Combine(mv)
								}
								tmpResultMap[t_ip] = tVerifyResult
								tmpTestSlice[t_ip] = true
								displayDetails(true, looper.Status() > -1, []config.VerifyResults{tVerifyResult})
							} else {
								if looper.InLooping() {
									mv, ok := tmpResultMap[t_ip]
									if ok {
										tVerifyResult.Combine(mv)
									}
									tmpResultMap[t_ip] = tVerifyResult
								}
								if config.Config.Debug {
									displayDetails(true, looper.Status() > -1, []config.VerifyResults{tVerifyResult})
								}
							}
						})
					}
				} else {
					dltBatch := thisSourceIPs.RetrieveSome(config.Config.DLTWorkerThread, !config.Config.TestAll)
					if len(dltBatch) == 0 {
						break SINGLE_ROUND
					}
					runDLTSingleRound(dltBatch, func(dltRes config.SingleVerifyResult) {
						dltDoneTasks++
						tVerifyResult := calcResult(dltRes, true)
						t_ip := *tVerifyResult.IP
						if validDLTResult(&tVerifyResult) {
							if config.Config.ResolveLoc && config.Config.SilenceMode && looper.Status() == -1 && (tVerifyResult.Loc == nil || len(*tVerifyResult.Loc) == 0) {
								loc := outbound.GetGeoInfoFromCF(&t_ip)
								tVerifyResult.Loc = &loc
							}
							v, ok := tmpResultMap[t_ip]
							if ok {
								tVerifyResult.Combine(v)
							}
							tmpResultMap[t_ip] = tVerifyResult
							tmpTestSlice[t_ip] = true
							displayDetails(true, looper.Status() > -1, []config.VerifyResults{tVerifyResult})
						} else {
							if looper.InLooping() {
								v, ok := tmpResultMap[t_ip]
								if ok {
									tVerifyResult.Combine(v)
								}
								tmpResultMap[t_ip] = tVerifyResult
							}
							if config.Config.Debug {
								displayDetails(true, looper.Status() > -1, []config.VerifyResults{tVerifyResult})
							}
						}
					})
				}

				if config.Config.Debug {
					displayStat(config.OverAllStat{
						DtTasksDone:  dtDoneTasks,
						DltTasksDone: dltDoneTasks,
						ResultCount:  len(tmpTestSlice),
						Remain:       thisSourceIPs.LenInt(),
					})
				}

				if !config.Config.TestAll && len(tmpTestSlice) >= t_result_min {
					break SINGLE_ROUND
				}
			}

			if len(tmpResultMap) == 0 {
				break LOOP
			}
			if !looper.Loop() {
				break LOOP
			} else {
				tmp_slice := make([]string, 0, len(tmpResultMap))
				for k := range tmpResultMap {
					tmp_slice = append(tmp_slice, k)
				}
				newSourceIPs := config.NewSourceIPs()
				if err := newSourceIPs.AddFromSlice(tmp_slice, config.TypeIPv4|config.TypeIPv6); err != nil {
					logger.Log.Errorf("failed to prepare loop candidates: %v\n", err)
					break LOOP
				}
				if err := newSourceIPs.AddPorts(config.Config.PortStrSlice); err != nil {
					logger.Log.Errorf("failed to add configured ports for loop retest: %v\n", err)
					break LOOP
				}
				thisSourceIPs = newSourceIPs
				if !config.Config.TestAll {
					t_result_min = len(tmp_slice)
				}
				looper.Sleep()
			}
		}

		for tIP := range tmpTestSlice {
			tr := tmpResultMap[tIP]
			isValid := true
			if !config.Config.DLTOnly && !validDTResult(&tr) {
				isValid = false
			}
			if !config.Config.DTOnly && !validDLTResult(&tr) {
				isValid = false
			}
			if isValid {
				config.VerifyResultsMap[tIP] = tr
			}
		}

		thisSourceIPs = config.SrcIPs
		
		hasReachedMin := !config.Config.TestAll && len(config.VerifyResultsMap) >= config.Config.ResultMin
		isTimedOut := time.Since(start_time) >= time.Duration(config.Config.TestTimeout)*time.Minute
		
		if hasReachedMin || isTimedOut {
			break RETRY_LOOP
		}
		
		if thisSourceIPs.IsEmpty() {
			supplemented := false
			if config.Config.Supplement {
				for currentSourceLevel < config.SourceLevelFull {
					currentSourceLevel++
					err := config.SupplementSourceIPs(currentSourceLevel, tMode)
					if err != nil {
						logger.Log.Errorf("IP supplementation failed for level %d: %v\n", currentSourceLevel, err)
						continue
					}
					if !config.SrcIPs.IsEmpty() {
						thisSourceIPs = config.SrcIPs
						supplemented = true
						break
					}
				}
			}
			if !supplemented {
				break RETRY_LOOP
			}
		}
		
		t_result_min = config.Config.ResultMin - len(config.VerifyResultsMap)
	}

	if dtTaskChan != nil {
		close(dtTaskChan)
	}
	if dltTaskChan != nil {
		close(dltTaskChan)
	}
	workerWG.Wait()
}

func main() {
	opts, shouldExit, exitCode, err := config.ConfigureApp(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}
	if opts.PrintVersion {
		if !config.Config.SilenceMode {
			print_version()
		}
		os.Exit(0)
	}
	if shouldExit {
		os.Exit(exitCode)
	}

	if err := outbound.PrepareOutboundOptions(&opts); err != nil {
		logger.Log.Errorf("Outbound preparation failed: %v", err)
		os.Exit(1)
	}

	if len(config.Config.FetchCFDomainsFile) > 0 {
		logger.Log.Infof("Starting dynamic Cloudflare CDN domain fetch...")
		domains, err := fetcher.FetchCloudflareDomains(config.Config.DNSServer, config.Config.TrancoLimit)
		if err != nil {
			logger.Log.Errorf("Domain fetch failed: %v", err)
			os.Exit(1)
		}
		if err := utils.WriteStringsToFile(config.Config.FetchCFDomainsFile, domains); err != nil {
			logger.Log.Errorf("Failed to save fetched domains: %v", err)
			os.Exit(1)
		}
		logger.Log.Infof("Successfully saved %d Cloudflare CDN domains to %s", len(domains), config.Config.FetchCFDomainsFile)
		os.Exit(0)
	}

	if len(config.Config.FetchIPv4File) > 0 {
		logger.Log.Infof("Starting dynamic IPv4 fetch...")
		cidrs, err := fetcher.FetchDynamicIPv4(config.Config.DNSServer, config.Config.TrancoLimit)
		if err != nil {
			logger.Log.Errorf("Fetch failed: %v", err)
			os.Exit(1)
		}
		if err := utils.WriteStringsToFile(config.Config.FetchIPv4File, cidrs); err != nil {
			logger.Log.Errorf("Failed to save fetched IPs: %v", err)
			os.Exit(1)
		}
		logger.Log.Infof("Successfully saved %d IPv4 CIDRs to %s", len(cidrs), config.Config.FetchIPv4File)
		os.Exit(0)
	}

	if len(config.Config.FetchIPv6File) > 0 {
		logger.Log.Infof("Starting dynamic IPv6 fetch...")
		cidrs, err := fetcher.FetchDynamicIPv6(config.Config.DNSServer, config.Config.TrancoLimit)
		if err != nil {
			logger.Log.Errorf("Fetch failed: %v", err)
			os.Exit(1)
		}
		if err := utils.WriteStringsToFile(config.Config.FetchIPv6File, cidrs); err != nil {
			logger.Log.Errorf("Failed to save fetched IPs: %v", err)
			os.Exit(1)
		}
		logger.Log.Infof("Successfully saved %d IPv6 CIDRs to %s", len(cidrs), config.Config.FetchIPv6File)
		os.Exit(0)
	}

	runWorker()

	if len(config.VerifyResultsMap) > 0 {
		verifyResultsSlice := make([]config.VerifyResults, 0)
		for _, v := range config.VerifyResultsMap {
			if config.Config.ResolveLoc && len(*v.Loc) == 0 {
				t_loc := outbound.GetGeoInfoFromCF(v.IP)
				v.Loc = &t_loc
			}
			verifyResultsSlice = append(verifyResultsSlice, v)
		}
		var records []db.DBRecord
		if config.Config.StoreToFile || config.Config.StoreToDB {
			records = db.GenDBRecords(verifyResultsSlice, config.Config.ResolveLocalASNAndCity)
			if config.Config.StoreToFile {
				if !config.Config.SilenceMode {
					logger.Log.Print("Writing CSV results to " + config.Config.ResultFile)
				}
				if err := db.WriteCSVResult(records, config.Config.ResultFile); err != nil {
					fmt.Fprintf(os.Stderr, "error: %v\n", err)
					os.Exit(1)
				}
				if !config.Config.SilenceMode {
					logger.Log.Println("  Done")
				}
			}
			if config.Config.StoreToDB {
				if !config.Config.SilenceMode {
					logger.Log.Print("Writing SQLite results to " + config.Config.DBFile)
				}
				if err := db.SaveDBRecords(records, config.Config.DBFile); err != nil {
					fmt.Fprintf(os.Stderr, "error: %v\n", err)
					os.Exit(1)
				}
				if !config.Config.SilenceMode {
					logger.Log.Println("  Done")
				}
			}
		}
		sort.Sort(sort.Reverse(config.ResultSpeedSorter(verifyResultsSlice)))
		if !config.Config.SilenceMode {
			logger.Log.Println()
			logger.Log.Println("All Results:")
			db.PrintFinalStat(verifyResultsSlice, config.Config.DTOnly, false)
		} else {
			if config.Config.Loop > 0 {
				db.PrintFinalStat(verifyResultsSlice, config.Config.DTOnly, true)
			}
		}
	}
}
