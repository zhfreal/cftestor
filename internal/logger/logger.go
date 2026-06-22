package logger

import (
	"fmt"
	"os"
	"time"
)

const (
	LogLevelDebug   = 1<<5 - 1
	LogLevelInfo    = 1<<4 - 1
	LogLevelWarning = 1<<3 - 1
	LogLevelError   = 1<<2 - 1
	LogLevelFatal   = 1<<1 - 1
	myIndent        = " "
)

type LogLevel int

type MyLogger struct {
	LoggerLevel LogLevel
	Indent      string
}

var Log MyLogger = NewLogger(LogLevelInfo)

func (myLogger *MyLogger) getLogLevelString(lv LogLevel) string {
	switch lv {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarning:
		return "WARNING"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
	}
	switch myLogger.LoggerLevel {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarning:
		return "WARNING"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
	}
	return "INFO"
}

func NewLogger(lv LogLevel) MyLogger {
	return MyLogger{lv, myIndent}
}

func getTimeNowStr() string {
	return time.Now().Format("15:04:05")
}

func (myLogger *MyLogger) matchLogLevel(lv LogLevel) bool {
	return myLogger.LoggerLevel&lv == lv
}

func (myLogger *MyLogger) log_newline(lv LogLevel, newline bool, info ...any) {
	if !myLogger.matchLogLevel(lv) {
		return
	}
	fmt.Print(getTimeNowStr())
	fmt.Print(myLogger.Indent)
	t_log_type_str := myLogger.getLogLevelString(lv)
	fmt.Printf("%v", t_log_type_str)
	fmt.Print(myLogger.Indent)
	myLogger.print(newline, info...)
}

func (myLogger *MyLogger) log_newlinef(lv LogLevel, format string, info ...any) {
	if !myLogger.matchLogLevel(lv) {
		return
	}
	fmt.Print(getTimeNowStr())
	fmt.Print(myLogger.Indent)
	t_log_type_str := myLogger.getLogLevelString(lv)
	fmt.Printf("%v", t_log_type_str)
	fmt.Print(myLogger.Indent)
	myLogger.printf(format, info...)
}

func (myLogger *MyLogger) print(newline bool, info ...any) {
	if len(info) >= 1 {
		fmt.Printf("%v", info[0])
		if len(info) > 1 {
			for _, t := range info[1:] {
				fmt.Printf("%s%v", myLogger.Indent, t)
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
	myLogger.log_newline(LogLevelDebug, newline, info...)
}

func (myLogger *MyLogger) debugf(format string, info ...any) {
	myLogger.log_newlinef(LogLevelDebug, format, info...)
}

func (myLogger *MyLogger) info(newline bool, info ...any) {
	myLogger.log_newline(LogLevelInfo, newline, info...)
}

func (myLogger *MyLogger) infof(format string, info ...any) {
	myLogger.log_newlinef(LogLevelInfo, format, info...)
}

func (myLogger *MyLogger) warning(newline bool, info ...any) {
	myLogger.log_newline(LogLevelWarning, newline, info...)
}

func (myLogger *MyLogger) warningf(format string, info ...any) {
	myLogger.log_newlinef(LogLevelWarning, format, info...)
}

func (myLogger *MyLogger) error(newline bool, info ...any) {
	myLogger.log_newline(LogLevelError, newline, info...)
}

func (myLogger *MyLogger) errorf(format string, info ...any) {
	myLogger.log_newlinef(LogLevelError, format, info...)
}

func (myLogger *MyLogger) fatal(newline bool, info ...any) {
	myLogger.log_newline(LogLevelFatal, newline, info...)
	os.Exit(1)
}

func (myLogger *MyLogger) fatalf(format string, info ...any) {
	myLogger.log_newlinef(LogLevelFatal, format, info...)
	os.Exit(1)
}

func (myLogger *MyLogger) log(loglvl LogLevel, newline bool, info ...any) {
	switch loglvl {
	case LogLevelDebug:
		myLogger.debug(newline, info...)
	case LogLevelInfo:
		myLogger.info(newline, info...)
	case LogLevelWarning:
		myLogger.warning(newline, info...)
	case LogLevelError:
		myLogger.error(newline, info...)
	case LogLevelFatal:
		myLogger.fatal(newline, info...)
	default:
	}
}

func (myLogger *MyLogger) logf(loglvl LogLevel, format string, info ...any) {
	switch loglvl {
	case LogLevelDebug:
		myLogger.debugf(format, info...)
	case LogLevelInfo:
		myLogger.infof(format, info...)
	case LogLevelWarning:
		myLogger.warningf(format, info...)
	case LogLevelError:
		myLogger.errorf(format, info...)
	case LogLevelFatal:
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
