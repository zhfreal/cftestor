package main

import (
	"fmt"
	"os"
	"sort"
	"sync"
	"time"
)

func print_version() {
	fmt.Println(appArt)
	fmt.Println(`  CF CDN IP scanner, find best IPs for you.
  https://github.com/zhfreal/cftestor`)
	fmt.Println()
	fmt.Printf("Version:    %v\n", version)
	fmt.Printf("BuildDate:  %v\n", buildDate)
	fmt.Printf("BuildTag:   %v\n", buildTag)
	fmt.Printf("BuildHash:  %v\n", buildHash)
	fmt.Println()
}

func validDTResult(tVerifyResult *VerifyResults) bool {
	if tVerifyResult.da > 0.0 &&
		tVerifyResult.da <= float64(Config.DTEvaluationDelay) &&
		tVerifyResult.dtpr*100.0 >= float64(Config.DTEvaluationDTPR) &&
		(!Config.EnableStdEv || (Config.EnableStdEv && tVerifyResult.daStd <= Config.DTStdExp)) {
		return true
	}
	return false
}

func validDLTResult(tVerifyResult *VerifyResults) bool {
	if tVerifyResult.dls >= Config.DLTEvaluationSpeed && tVerifyResult.dlds > downloadSizeMin {
		return true
	}
	return false
}

var (
	dtTaskChan    chan *task
	dtResultChan  chan singleVerifyResult
	dltTaskChan   chan *task
	dltResultChan chan singleVerifyResult
	workerWG      sync.WaitGroup
)

func initWorkers() {
	if !Config.DLTOnly {
		dtTaskChan = make(chan *task, Config.DTWorkerThread)
		dtResultChan = make(chan singleVerifyResult, Config.DTWorkerThread)
		for range Config.DTWorkerThread {
			workerWG.Add(1)
			if Config.DTHttps {
				go downloadWorkerNew(dtTaskChan, dtResultChan, &workerWG, &Config.DTUrl, Config.DTTimeoutDuration, Config.DTCount, true)
			} else {
				go sslDTWorkerNew(dtTaskChan, dtResultChan, &workerWG)
			}
		}
	}
	if !Config.DTOnly {
		dltTaskChan = make(chan *task, Config.DLTWorkerThread)
		dltResultChan = make(chan singleVerifyResult, Config.DLTWorkerThread)
		for range Config.DLTWorkerThread {
			workerWG.Add(1)
			go downloadWorkerNew(dltTaskChan, dltResultChan, &workerWG, &Config.DLTUrl, Config.HttpRspTimeoutDuration, Config.DLTCount, false)
		}
	}
}

func runDTSingleRound(ips []*string, handler func(singleVerifyResult)) {
	size := len(ips)
	if size == 0 {
		return
	}
	max_failure := get_max_failure(true)
	go func() {
		for _, ip := range ips {
			dtTaskChan <- NewTask(ip, max_failure)
		}
	}()

	for range size {
		handler(<-dtResultChan)
	}
}

func runDLTSingleRound(ips []*string, handler func(singleVerifyResult)) {
	size := len(ips)
	if size == 0 {
		return
	}
	max_failure := get_max_failure(false)
	go func() {
		for _, ip := range ips {
			dltTaskChan <- NewTask(ip, max_failure)
		}
	}()

	for range size {
		handler(<-dltResultChan)
	}
}

func runWorker() {
	initWorkers()

	var thisSourceIPs = srcIPs
	var t_result_min = Config.ResultMin
	var start_time = time.Now()

RETRY_LOOP:
	for {
		tmpResultMap := make(map[string]VerifyResults)
		var tmpTestSlice map[string]bool
		looper := NewSafeLooperWithInterval(Config.Loop, Config.LoopInterval*1000)
	LOOP:
		for {
			dtDoneTasks := 0
			dltDoneTasks := 0
			tmpTestSlice = make(map[string]bool)

		SINGLE_ROUND:
			for {
				if time.Since(start_time) >= time.Duration(Config.TestTimeout)*time.Minute {
					break SINGLE_ROUND
				}

				// DT Stage
				if !Config.DLTOnly {
					dtBatch := thisSourceIPs.RetrieveSome(Config.DTWorkerThread, !Config.TestAll)
					if len(dtBatch) == 0 {
						break SINGLE_ROUND
					}

					dltBatch := make([]*string, 0)
					cachedMap := make(map[string]VerifyResults)

					runDTSingleRound(dtBatch, func(dtRes singleVerifyResult) {
						dtDoneTasks++
						tVerifyResult := calcResult(dtRes, false)
						t_ip := *tVerifyResult.ip

						if validDTResult(&tVerifyResult) {
							if !Config.DTOnly {
								cachedMap[t_ip] = tVerifyResult
								dltBatch = append(dltBatch, &t_ip)
								if Config.Debug {
									displayDetails(false, looper.Status() > -1, []VerifyResults{tVerifyResult})
								}
							} else {
								if Config.ResolveLoc && Config.SilenceMode && looper.Status() == -1 && (tVerifyResult.loc == nil || len(*tVerifyResult.loc) == 0) {
									loc := getGeoInfoFromCF(tVerifyResult.ip)
									tVerifyResult.loc = &loc
								}
								v, ok := tmpResultMap[t_ip]
								if ok {
									tVerifyResult.combine(v)
								}
								tmpResultMap[t_ip] = tVerifyResult
								displayDetails(false, looper.Status() > -1, []VerifyResults{tVerifyResult})
								tmpTestSlice[t_ip] = true
							}
						} else {
							if looper.InLooping() {
								v, ok := tmpResultMap[t_ip]
								if ok {
									tVerifyResult.combine(v)
								}
								tmpResultMap[t_ip] = tVerifyResult
							}
							if Config.Debug {
								displayDetails(false, looper.Status() > -1, []VerifyResults{tVerifyResult})
							}
						}
					})

					// DLT Stage for DT candidates
					if !Config.DTOnly && len(dltBatch) > 0 {
						runDLTSingleRound(dltBatch, func(dltRes singleVerifyResult) {
							dltDoneTasks++
							tVerifyResult := calcResult(dltRes, true)
							t_ip := *tVerifyResult.ip
							v := cachedMap[t_ip]
							tVerifyResult.combine(v)

							if validDLTResult(&tVerifyResult) && validDTResult(&tVerifyResult) {
								if Config.ResolveLoc && Config.SilenceMode && looper.Status() == -1 && (tVerifyResult.loc == nil || len(*tVerifyResult.loc) == 0) {
									loc := getGeoInfoFromCF(&t_ip)
									tVerifyResult.loc = &loc
								}
								mv, ok := tmpResultMap[t_ip]
								if ok {
									tVerifyResult.combine(mv)
								}
								tmpResultMap[t_ip] = tVerifyResult
								tmpTestSlice[t_ip] = true
								displayDetails(true, looper.Status() > -1, []VerifyResults{tVerifyResult})
							} else {
								if looper.InLooping() {
									mv, ok := tmpResultMap[t_ip]
									if ok {
										tVerifyResult.combine(mv)
									}
									tmpResultMap[t_ip] = tVerifyResult
								}
								if Config.Debug {
									displayDetails(true, looper.Status() > -1, []VerifyResults{tVerifyResult})
								}
							}
						})
					}
				} else {
					// DLT Only Stage
					dltBatch := thisSourceIPs.RetrieveSome(Config.DLTWorkerThread, !Config.TestAll)
					if len(dltBatch) == 0 {
						break SINGLE_ROUND
					}
					runDLTSingleRound(dltBatch, func(dltRes singleVerifyResult) {
						dltDoneTasks++
						tVerifyResult := calcResult(dltRes, true)
						t_ip := *tVerifyResult.ip
						if validDLTResult(&tVerifyResult) {
							if Config.ResolveLoc && Config.SilenceMode && looper.Status() == -1 && (tVerifyResult.loc == nil || len(*tVerifyResult.loc) == 0) {
								loc := getGeoInfoFromCF(&t_ip)
								tVerifyResult.loc = &loc
							}
							v, ok := tmpResultMap[t_ip]
							if ok {
								tVerifyResult.combine(v)
							}
							tmpResultMap[t_ip] = tVerifyResult
							tmpTestSlice[t_ip] = true
							displayDetails(true, looper.Status() > -1, []VerifyResults{tVerifyResult})
						} else {
							if looper.InLooping() {
								v, ok := tmpResultMap[t_ip]
								if ok {
									tVerifyResult.combine(v)
								}
								tmpResultMap[t_ip] = tVerifyResult
							}
							if Config.Debug {
								displayDetails(true, looper.Status() > -1, []VerifyResults{tVerifyResult})
							}
						}
					})
				}

				if Config.Debug {
					displayStat(overAllStat{
						dtTasksDone:  dtDoneTasks,
						dltTasksDone: dltDoneTasks,
						resultCount:  len(tmpTestSlice),
						remain:       thisSourceIPs.LenInt(),
					})
				}

				if !Config.TestAll && len(tmpTestSlice) >= t_result_min {
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
				newSourceIPs := NewSourceIPs()
				if err := newSourceIPs.AddFromSlice(tmp_slice, TypeIPv4|TypeIPv6); err != nil {
					myLogger.Errorf("failed to prepare loop candidates: %v\n", err)
					break LOOP
				}
				if err := newSourceIPs.AddPorts(Config.PortStrSlice); err != nil {
					myLogger.Errorf("failed to add configured ports for loop retest: %v\n", err)
					break LOOP
				}
				thisSourceIPs = newSourceIPs
				if !Config.TestAll {
					t_result_min = len(tmp_slice)
				}
				looper.Sleep()
			}
		}

		for tIP := range tmpTestSlice {
			tr := tmpResultMap[tIP]
			isValid := true
			if !Config.DLTOnly && !validDTResult(&tr) {
				isValid = false
			}
			if !Config.DTOnly && !validDLTResult(&tr) {
				isValid = false
			}
			if isValid {
				verifyResultsMap[tIP] = tr
			}
		}

		thisSourceIPs = srcIPs
		if (!Config.TestAll && len(verifyResultsMap) >= Config.ResultMin) || thisSourceIPs.IsEmpty() || time.Since(start_time) >= time.Duration(Config.TestTimeout)*time.Minute {
			break RETRY_LOOP
		} else {
			t_result_min = Config.ResultMin - len(verifyResultsMap)
		}
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
	shouldExit, exitCode, err := configureApp(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}
	if shouldExit {
		os.Exit(exitCode)
	}

	// start controller worker
	runWorker()
	if len(verifyResultsMap) > 0 {
		verifyResultsSlice := make([]VerifyResults, 0)
		for _, v := range verifyResultsMap {
			if Config.ResolveLoc && len(*v.loc) == 0 {
				t_loc := getGeoInfoFromCF(v.ip)
				v.loc = &t_loc
			}
			verifyResultsSlice = append(verifyResultsSlice, v)
		}
		var records []DBRecord
		if Config.StoreToFile || Config.StoreToDB {
			records = genDBRecords(verifyResultsSlice, Config.ResolveLocalASNAndCity)
			// write to csv file
			if Config.StoreToFile {
				if !Config.SilenceMode {
					myLogger.Print("Writing CSV results to " + Config.ResultFile)
				}
				if err := writeCSVResult(records, Config.ResultFile); err != nil {
					fmt.Fprintf(os.Stderr, "error: %v\n", err)
					os.Exit(1)
				}
				if !Config.SilenceMode {
					myLogger.Println("  Done")
				}
			}
			// write to db
			if Config.StoreToDB {
				if !Config.SilenceMode {
					myLogger.Print("Writing SQLite results to " + Config.DBFile)
				}
				if err := saveDBRecords(records, Config.DBFile); err != nil {
					fmt.Fprintf(os.Stderr, "error: %v\n", err)
					os.Exit(1)
				}
				if !Config.SilenceMode {
					myLogger.Println("  Done")
				}
			}
		}
		// sort by speed
		sort.Sort(sort.Reverse(resultSpeedSorter(verifyResultsSlice)))
		if !Config.SilenceMode {
			myLogger.Println()
			myLogger.Println("All Results:")
			printFinalStat(verifyResultsSlice, Config.DTOnly, false)
		} else {
			if Config.Loop > 0 {
				printFinalStat(verifyResultsSlice, Config.DTOnly, true)
			}
		}
	}
}
