package main

import (
	"fmt"
	"os"
)

const (
	logLevelDebug   = 1<<5 - 1
	logLevelInfo    = 1<<4 - 1
	logLevelWarning = 1<<3 - 1
	logLevelError   = 1<<2 - 1
	logLevelFatal   = 1<<1 - 1
	myIndent        = " "
)

type LogLevel int

type MyLogger struct {
	loggerLevel LogLevel
	indent      string
}

func (myLogger *MyLogger) getLogLevelString(lv LogLevel) string {
	switch lv {
	case logLevelDebug:
		return "DEBUG"
	case logLevelInfo:
		return "INFO"
	case logLevelWarning:
		return "WARNING"
	case logLevelError:
		return "ERROR"
	case logLevelFatal:
		return "FATAL"
	default:
	}
	switch myLogger.loggerLevel {
	case logLevelDebug:
		return "DEBUG"
	case logLevelInfo:
		return "INFO"
	case logLevelWarning:
		return "WARNING"
	case logLevelError:
		return "ERROR"
	case logLevelFatal:
		return "FATAL"
	default:
	}
	return "INFO"
}

func (myLogger *MyLogger) newLogger(lv LogLevel) MyLogger {
	return MyLogger{lv, myIndent}
}

func (myLogger *MyLogger) log_newline(lv LogLevel, newline bool, info ...any) {
	fmt.Print(getTimeNowStr())
	fmt.Print(myLogger.indent)
	t_log_type_str := myLogger.getLogLevelString(lv)
	fmt.Printf("%v", t_log_type_str)
	fmt.Print(myLogger.indent)
	myLogger.print(newline, info...)
}

func (myLogger *MyLogger) log_newlinef(lv LogLevel, format string, info ...any) {
	fmt.Print(getTimeNowStr())
	fmt.Print(myLogger.indent)
	t_log_type_str := myLogger.getLogLevelString(lv)
	fmt.Printf("%v", t_log_type_str)
	fmt.Print(myLogger.indent)
	myLogger.printf(format, info...)
}

func (myLogger *MyLogger) print(newline bool, info ...any) {
	if len(info) >= 1 {
		fmt.Printf("%v", info[0])
		if len(info) > 1 {
			for _, t := range info[1:] {
				fmt.Printf("%s%v", myLogger.indent, t)
			}
		}
	}
	if newline {
		fmt.Println()
	}
}

func (myLogger *MyLogger) printf(format string, info ...any) {
	fmt.Printf(format, info...)
}

func (myLogger *MyLogger) debug(newline bool, info ...any) {
	myLogger.log_newline(logLevelDebug, newline, info...)
}

func (myLogger *MyLogger) debugf(format string, info ...any) {
	myLogger.log_newlinef(logLevelDebug, format, info...)
}

func (myLogger *MyLogger) info(newline bool, info ...any) {
	myLogger.log_newline(logLevelInfo, newline, info...)
}

func (myLogger *MyLogger) infof(format string, info ...any) {
	myLogger.log_newlinef(logLevelInfo, format, info...)
}

func (myLogger *MyLogger) warning(newline bool, info ...any) {
	myLogger.log_newline(logLevelWarning, newline, info...)
}

func (myLogger *MyLogger) warningf(format string, info ...any) {
	myLogger.log_newlinef(logLevelWarning, format, info...)
}

func (myLogger *MyLogger) error(newline bool, info ...any) {
	myLogger.log_newline(logLevelError, newline, info...)
}

func (myLogger *MyLogger) errorf(format string, info ...any) {
	myLogger.log_newlinef(logLevelError, format, info...)
}

func (myLogger *MyLogger) fatal(newline bool, info ...any) {
	myLogger.log_newline(logLevelFatal, newline, info...)
	os.Exit(1)
}

func (myLogger *MyLogger) fatalf(format string, info ...any) {
	myLogger.log_newlinef(logLevelFatal, format, info...)
	os.Exit(1)
}

func (myLogger *MyLogger) log(loglvl LogLevel, newline bool, info ...any) {
	switch loglvl {
	case logLevelDebug:
		myLogger.debug(newline, info...)
	case logLevelInfo:
		myLogger.info(newline, info...)
	case logLevelWarning:
		myLogger.warning(newline, info...)
	case logLevelError:
		myLogger.error(newline, info...)
	case logLevelFatal:
		myLogger.fatal(newline, info...)
	default:
	}
}

func (myLogger *MyLogger) logf(loglvl LogLevel, format string, info ...any) {
	switch loglvl {
	case logLevelDebug:
		myLogger.debugf(format, info...)
	case logLevelInfo:
		myLogger.infof(format, info...)
	case logLevelWarning:
		myLogger.warningf(format, info...)
	case logLevelError:
		myLogger.errorf(format, info...)
	case logLevelFatal:
		myLogger.fatalf(format, info...)
	default:
	}
}

func (myLogger *MyLogger) Debug(info ...any) {
	myLogger.debug(false, info...)
}

func (myLogger *MyLogger) Debugf(format string, info ...any) {
	myLogger.debugf(format, info...)
}

func (myLogger *MyLogger) Debugln(info ...any) {
	myLogger.debug(true, info...)
}

func (myLogger *MyLogger) Info(info ...any) {
	myLogger.info(false, info...)
}

func (myLogger *MyLogger) Infof(format string, info ...any) {
	myLogger.infof(format, info...)
}

func (myLogger *MyLogger) Infoln(info ...any) {
	myLogger.info(true, info...)
}

func (myLogger *MyLogger) Warning(info ...any) {
	myLogger.warning(false, info...)
}

func (myLogger *MyLogger) Warningf(format string, info ...any) {
	myLogger.warningf(format, info...)
}

func (myLogger *MyLogger) Warningln(info ...any) {
	myLogger.warning(true, info...)
}

func (myLogger *MyLogger) Error(info ...any) {
	myLogger.error(false, info...)
}

func (myLogger *MyLogger) Errorf(format string, info ...any) {
	myLogger.errorf(format, info...)
}

func (myLogger *MyLogger) Errorln(info ...any) {
	myLogger.error(true, info...)
}

func (myLogger *MyLogger) Fatal(info ...any) {
	myLogger.fatal(false, info...)
}

func (myLogger *MyLogger) Fatalf(format string, info ...any) {
	myLogger.fatalf(format, info...)
}

func (myLogger *MyLogger) Fatalln(info ...any) {
	myLogger.fatal(true, info...)
}

func (myLogger *MyLogger) Log(loglvl LogLevel, a ...any) {
	myLogger.log(loglvl, false, a...)
}

func (myLogger *MyLogger) Logf(loglvl LogLevel, format string, a ...any) {
	myLogger.logf(loglvl, format, a...)
}

func (myLogger *MyLogger) Logln(loglvl LogLevel, a ...any) {
	myLogger.log(loglvl, true, a...)
}

func (myLogger *MyLogger) Print(info ...any) {
	myLogger.print(false, info...)
}

func (myLogger *MyLogger) Printf(format string, info ...any) {
	myLogger.printf(format, info...)
}

func (myLogger *MyLogger) Println(info ...any) {
	myLogger.print(true, info...)
}

func (myLogger *MyLogger) PrintSingleStat(logLvl LogLevel, v []VerifyResults, ov overAllStat, showSpeed bool) {
	myLogger.PrintDetails(logLvl, v, showSpeed)
	myLogger.PrintOverAllStat(logLvl, ov)
}

// log when debug or info
func (myLogger *MyLogger) PrintDetails(logLvl LogLevel, v []VerifyResults, showSpeed bool) {
	// no data for print
	if len(v) == 0 {
		return
	}

	// print only when logLvl is permitted in myLogger
	if myLogger.loggerLevel&logLvl != logLvl {
		return
	}
	// fix indent
	if len(myLogger.indent) == 0 {
		myLogger.indent = myIndent
	}
	lc := v
	for i := 0; i < len(lc); i++ {
		t_ip := *lc[i].ip
		if len(*lc[i].loc) > 0 {
			t_ip = fmt.Sprintf("%s#%s", t_ip, *lc[i].loc)
		}
		myLogger.Logf(logLvl, "IP:%v%s", t_ip, myLogger.indent)
		if showSpeed {
			myLogger.Printf("Speed(KB/s):%.2f%s", lc[i].dls, myLogger.indent)
		}
		myLogger.Printf("Delay(ms):%.0f", lc[i].da)
		myLogger.Printf("%sStab.(%%):%.2f", myLogger.indent, lc[i].dtpr*100)
		if enableStdEv {
			myLogger.Printf("%sVar.:%.2f", myLogger.indent, lc[i].daVar)
			myLogger.Printf("%sStd.:%.2f", myLogger.indent, lc[i].daStd)
		}
	}
	myLogger.Println()
}

// print just IPs
func (myLogger *MyLogger) PrintClearIPs(v []VerifyResults) {
	// no data for print
	if len(v) == 0 {
		return
	}
	lc := v
	for i := 0; i < len(lc); i++ {
		myLogger.Println(*lc[i].ip)
	}
}

// print OverAll statistic
func (myLogger *MyLogger) PrintOverAllStat(logLvl LogLevel, ov overAllStat) {
	// print only when logLvl is permitted in myLogger
	if myLogger.loggerLevel&logLvl != logLvl {
		return
	}
	// fix space
	if len(myLogger.indent) == 0 {
		myLogger.indent = myIndent
	}
	myLogger.Logf(logLvl, "Result: %d    ", ov.resultCount)
	srcCount := len(srcHosts) + len(srcIPRsRaw) + len(srcIPRsExtracted)
	if !dltOnly {
		myLogger.Printf("DT - Tested: %d  ", ov.dtTasksDone)
		dtCached := ov.dtCached + srcCount
		myLogger.Printf("Cached: %d\t", dtCached)
	}
	if !dtOnly {
		myLogger.Printf("DLT - Tested: %d  ", ov.dltTasksDone)
		dltCached := ov.dltCached
		if dltOnly {
			dltCached += srcCount
		}
		myLogger.Printf("Cached: %d", dltCached)
	}
	myLogger.Println("")
}
