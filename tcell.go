package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

func printOneRow(x, y int, style tcell.Style, str string) {
	for _, c := range str {
		var comb []rune
		w := runewidth.RuneWidth(c)
		if w == 0 {
			comb = []rune{c}
			c = ' '
			w = 1
		}
		(*termAll).SetContent(x, y, c, comb, style)
		x += w
	}
	tx, _ := (*termAll).Size()
	if tx > len(str) {
		c := ' '
		for i := 0; i < tx-len(str); i++ {
			(*termAll).SetContent(x, y, c, nil, tcell.StyleDefault)
			x += 1
		}
	}
}

func initScreen() {
	defer func() { (*termAll).Sync() }()
	(*termAll).Clear()
	printRuntimeWithoutSync()
	printTitlePreWithoutSync()
	printCancelWithoutSync()
	updateScreen()
}

func printRuntimeWithoutSync() {
	printOneRow(0, titleRuntimeRow, contentStyle, *titleRuntime)
}

func printTitlePreWithoutSync() {
	var len0, len1, len2 int
	len0 = MaxInt(len(titlePre[0][0]), len(titlePre[1][0]))
	len1 = MaxInt(len(titlePre[0][1]), len(titlePre[1][1]))
	len2 = MaxInt(len(titlePre[0][2]), len(titlePre[1][2]))
	rowStr0 := fmt.Sprintf("%*v%-*v  %*v%v", len0, titlePre[0][0], len1, titlePre[0][1], len2, titlePre[0][2], titlePre[0][3])
	printOneRow(0, titlePreRow, contentStyle, rowStr0)
	if len(titlePre[1][0]) > 0 && len(titlePre[1][1]) > 0 {
		rowStr1 := fmt.Sprintf("%*v%-*v", len0, titlePre[1][0], len1, titlePre[1][1])
		if len(titlePre[1][2]) > 0 && len(titlePre[1][3]) > 0 {
			rowStr1 += fmt.Sprintf("  %*v%v", len2, titlePre[1][2], titlePre[1][3])
		}
		printOneRow(0, titlePreRow+1, contentStyle, rowStr1)
	}
}

func printCancelWithoutSync() {
	printOneRow(0, titleCancelRow, titleStyleCancel, titleCancel)
}

func printCancelConfirmWithoutSync() {
	printOneRow(0, titleCancelRow, titleStyleCancel, titleCancelConfirm)
}

func printQuitWaitingWithoutSync() {
	printOneRow(0, titleCancelRow, titleStyleCancel, titleWaitQuit)
}

func printCancel() {
	printCancelWithoutSync()
	(*termAll).Show()
}

func printCancelConfirm() {
	printCancelConfirmWithoutSync()
	(*termAll).Show()
}

func printQuitWaiting() {
	printQuitWaitingWithoutSync()
	(*termAll).Show()
}

func printQuittingCountDown(sec int) {
	for i := sec; i > 0; i-- {
		printOneRow(0, titleCancelRow, titleStyleCancel, fmt.Sprintf("Exit in %ds...", i))
		(*termAll).Show()
		time.Sleep(time.Second)
	}
}

func printTitleResultHintWithoutSync() {
	printOneRow(0, titleResultHintRow, titleStyle, fmt.Sprintf("%s%v", titleResultHint, len(verifyResultsMap)))
}

func printTitleDebugHintWithoutSync() {
	if !debug {
		return
	}
	printOneRow(0, titleDebugHintRow, contentStyle, titleDebugHint)
}

func printTaskStatWithoutSync() {
	if !dltOnly {
		printOneRow(0, titleTasksStatRow, contentStyle, *titleTasksStat[0])
		if !dtOnly {
			// start from row 23
			printOneRow(resultStatIndent, titleTasksStatRow+1, contentStyle, *titleTasksStat[1])
		}
	} else {
		printOneRow(0, titleTasksStatRow, contentStyle, *titleTasksStat[0])
	}
}

func printDetailsListWithoutSync(details [][]*string, startRow int, maxRowsDisplayed int) {
	if len(details) == 0 {
		return
	}
	t_len := len(details)
	t_lowest := 0
	if t_len > maxRowsDisplayed {
		t_lowest = t_len - maxRowsDisplayed
	}
	// scan for indent
	t_indent_slice := make([]int, 0)
	for _, v := range detailTitleSlice {
		t_indent_slice = append(t_indent_slice, len(v))
	}
	for i := t_lowest; i < t_len; i++ {
		for j := 0; j < len(t_indent_slice); j++ {
			t_indent_slice[j] = MaxInt(t_indent_slice[j], len(*details[i][j]))
		}
	}
	// print title
	t_sb := strings.Builder{}
	for j := 0; j < len(t_indent_slice)-1; j++ {
		t_sb.WriteString(fmt.Sprintf("%-*s%s", t_indent_slice[j], detailTitleSlice[j], myIndent))
	}
	t_sb.WriteString(fmt.Sprintf("%v", detailTitleSlice[len(t_indent_slice)-1]))
	printOneRow(0, startRow, contentStyle, t_sb.String())
	// print list
	for i := t_len - 1; i >= t_lowest; i-- {
		t_sb.Reset()
		for j := 0; j < len(t_indent_slice)-1; j++ {
			t_sb.WriteString(fmt.Sprintf("%-*s%s", t_indent_slice[j], *details[i][j], myIndent))
		}
		t_sb.WriteString(*details[i][len(t_indent_slice)-1])
		printOneRow(0, startRow+t_len-i, contentStyle, t_sb.String())
	}
}

func printResultListWithoutSync() {
	printTitleResultHintWithoutSync()
	printDetailsListWithoutSync(resultStrSlice, titleResultRow, maxResultsDisplay)
}

func printDebugListWithoutSync() {
	if !debug {
		return
	}
	printTitleDebugHintWithoutSync()
	printDetailsListWithoutSync(debugStrSlice, titleDebugRow, maxDebugDisplay)
}

func updateScreen() {
	defer func() { (*termAll).Show() }()
	printTaskStatWithoutSync()
	printResultListWithoutSync()
	printDebugListWithoutSync()
}

func initTitleStr() {
	var tMsgRuntime string
	tMsgRuntime = fmt.Sprintf("%v %v - ", runTime, version)

	if !dltOnly {
		tMsgRuntime += fmt.Sprintf("Start Delay (%v) Test (DT)", dtSource)
		if !dtOnly {
			tMsgRuntime += " and "
		}
	}
	if !dtOnly {
		tMsgRuntime += "Speed Test (DLT)"
	}
	titleRuntime = &tMsgRuntime
	titlePre[0][0] = "Result Exp.:"
	// we just control the display "resultMin" in main.init()
	titlePre[0][1] = " " + strconv.Itoa(resultMin)
	// if !testAll {
	// 	titlePre[0][1] = " " + strconv.Itoa(resultMin)
	// } else {
	// 	titlePre[0][1] = " ~"
	// }
	if dtOnly {
		titlePre[0][2] = "Max Delay:"
		titlePre[0][3] = fmt.Sprintf(" %vms", dtEvaluationDelay)
		titlePre[1][0] = "Min Stab.:"
		titlePre[1][1] = fmt.Sprintf(" %v", dtEvaluationDTPR) + "%"
	} else if dltOnly {
		titlePre[0][2] = "Min Speed:"
		titlePre[0][3] = fmt.Sprintf(" %vKB/s", dltEvaluationSpeed)
	} else {
		titlePre[0][2] = "Min Speed:"
		titlePre[0][3] = fmt.Sprintf(" %vKB/s", dltEvaluationSpeed)
		titlePre[1][0] = "Max Delay:"
		titlePre[1][1] = fmt.Sprintf(" %vms", dtEvaluationDelay)
		titlePre[1][2] = "Min Stab.:"
		titlePre[1][3] = fmt.Sprintf(" %v", dtEvaluationDTPR) + "%"
	}
	detailTitleSlice = append(detailTitleSlice, "IP")
	if !dtOnly {
		detailTitleSlice = append(detailTitleSlice, "Speed(KB/s)")
	}
	detailTitleSlice = append(detailTitleSlice, "Delay(ms)")
	if !dltOnly {
		detailTitleSlice = append(detailTitleSlice, "Stab.(%)")
	}
	updateTaskStatStr(overAllStat{0, 0, 0, 0, 0, 0, 0})
}

func updateTaskStatStr(ov overAllStat) {
	var t = strings.Builder{}
	var t1 = strings.Builder{}
	t.WriteString(getTimeNowStr())
	t.WriteString(myIndent)
	// t.WriteString(fmt.Sprintf("Result:%-*d%s", resultNumLen, resultCount, myIndent))
	t_dtCachedS := ov.dtCached
	t_dltCachedS := ov.dltCached
	t_dtCachedSNumLen := len(strconv.Itoa(t_dtCachedS))
	t_dltCachedSNumLen := len(strconv.Itoa(t_dltCachedS))
	t_dtDoneNumLen := len(strconv.Itoa(ov.dtTasksDone))
	t_dltDoneNumLen := len(strconv.Itoa(ov.dltTasksDone))
	var t_indent = 0
	if !dltOnly {
		t_indent = MaxInt(dtThreadsNumLen, t_dtCachedSNumLen, t_dltCachedSNumLen, t_dtDoneNumLen, t_dltDoneNumLen)
	}
	if !dtOnly {
		t_indent = MaxInt(t_indent, dltThreadsNumLen, t_dtCachedSNumLen, t_dltCachedSNumLen, t_dtDoneNumLen, t_dltDoneNumLen)
	}
	if !dltOnly {
		if dtOnly {
			t.WriteString(fmt.Sprintf("DT - Tested:%-*d%s", t_indent, ov.dtTasksDone, myIndent))

		} else {
			t.WriteString(fmt.Sprintf("DT  - Tested:%-*d%s", t_indent, ov.dtTasksDone, myIndent))
			t1.WriteString(fmt.Sprintf("DLT - Tested:%-*d%s", t_indent, ov.dltTasksDone, myIndent))
			t1.WriteString(fmt.Sprintf("OnGoing:%-*d%s", t_indent, ov.dltOnGoing, myIndent))
			t1.WriteString(fmt.Sprintf("Cached:%-*d%s", t_indent, t_dltCachedS, myIndent))
			ts1 := t1.String()
			titleTasksStat[1] = &ts1
		}
		t.WriteString(fmt.Sprintf("OnGoing:%-*d%s", t_indent, ov.dtOnGoing, myIndent))
		t.WriteString(fmt.Sprintf("Cached:%-*d%s", t_indent, t_dtCachedS, myIndent))
		ts := t.String()
		titleTasksStat[0] = &ts
	} else {
		t.WriteString(fmt.Sprintf("DLT - Tested:%-*d%s", t_indent, ov.dltTasksDone, myIndent))
		t.WriteString(fmt.Sprintf("OnGoing:%-*d%s", t_indent, ov.dltOnGoing, myIndent))
		t.WriteString(fmt.Sprintf("Cached:%-*d%s", t_indent, t_dltCachedS, myIndent))
		ts := t.String()
		titleTasksStat[0] = &ts
	}
}

func updateDetailList(showSpeed bool, src [][]*string, v []VerifyResults, limit int) (dst [][]*string) {
	dst = src
	for _, tv := range v {
		t_str_list := make([]*string, 0)
		// t_v1 := fmt.Sprintf("%v", *tv.ip)
		// t_str_list = append(t_str_list, &t_v1)
		tStr := fmt.Sprintf("%s#%s", *tv.ip, *tv.loc)
		t_str_list = append(t_str_list, &tStr)
		// show speed only when it performed DLT
		if !dtOnly {
			t_v2 := " "
			if showSpeed {
				t_v2 = fmt.Sprintf("%.2f", tv.dls)
			}
			t_str_list = append(t_str_list, &t_v2)
		}
		t_v3 := fmt.Sprintf("%.0f", tv.da)
		t_str_list = append(t_str_list, &t_v3)
		// show DTPR only when it performed DT
		if !dltOnly {
			t_v4 := fmt.Sprintf("%.2f", tv.dtpr*100)
			t_str_list = append(t_str_list, &t_v4)
		}
		dst = append(dst, t_str_list)
	}
	if len(dst) > limit {
		dst = dst[(len(dst) - limit):]
	}
	return
}

func updateResultStrList(showSpeed bool, v []VerifyResults) {
	resultStrSlice = updateDetailList(showSpeed, resultStrSlice, v, maxResultsDisplay)
}

func updateDebugStrList(showSpeed bool, v []VerifyResults) {
	if !debug {
		return
	}
	debugStrSlice = updateDetailList(showSpeed, debugStrSlice, v, maxDebugDisplay)
}

func updateResult(showSpeed bool, v []VerifyResults) {
	defer (*termAll).Show()
	updateResultStrList(showSpeed, v)
	printResultListWithoutSync()
}

func updateDebug(showSpeed bool, v []VerifyResults) {
	if !debug {
		return
	}
	defer (*termAll).Show()
	updateDebugStrList(showSpeed, v)
	printDebugListWithoutSync()
}

func updateTaskStat(ov overAllStat) {
	defer (*termAll).Show()
	updateTaskStatStr(ov)
	printTaskStatWithoutSync()
}

func updateTcellDetails(isResult, showSpeed bool, v []VerifyResults) {
	// prevent display debug msg when in not-debug mode
	if !debug {
		return
	}
	if isResult { // result
		updateResult(showSpeed, v)
	} else { // non-debug
		updateDebug(showSpeed, v)
	}
}
