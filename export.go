package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"text/tabwriter"
)

func writeCSVResult(data []DBRecord, filePath string) {
	var fp = &os.File{}
	var err error
	var w = &csv.Writer{}
	if !fileExists(filePath) {
		fp, err = os.Create(filePath)
		if err != nil {
			log.Fatalf("Create File %v failed with: %v", filePath, err)
		}
		wn, wErr := fp.Write(utf8BomBytes)
		if wn != len(utf8BomBytes) && wErr != nil {
			log.Fatalf("Write csv File %v failed with: %v", filePath, err)
		}
		w = csv.NewWriter(fp)
		err = w.Write(resultCsvHeader)
		if err != nil {
			log.Fatalf("Write csv File %v failed with: %v", filePath, err)
		}
	} else {
		fp, err = os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, os.FileMode(0644))
		if err != nil {
			log.Fatalf("Open File %v failed with: %v", filePath, err)
		}
		w = csv.NewWriter(fp)
	}
	defer func() { _ = fp.Close() }()

	for _, tD := range data {
		asn_str, city := "", ""
		if tD.Asn > 0 {
			asn_str = fmt.Sprintf("AS%v", tD.Asn)
			city = tD.City
		}
		err = w.Write([]string{
			tD.TestTimeStr,
			tD.IP,
			fmt.Sprintf("%.2f", tD.DLS),
			fmt.Sprintf("%.0f", tD.DA),
			tD.DS,
			fmt.Sprintf("%.2f", tD.DTPR*100),
			fmt.Sprintf("%d", tD.DTC),
			fmt.Sprintf("%d", tD.DTPC),
			fmt.Sprintf("%.0f", tD.DMI),
			fmt.Sprintf("%.0f", tD.DMX),
			fmt.Sprintf("%d", tD.DLTC),
			fmt.Sprintf("%d", tD.DLTPC),
			fmt.Sprintf("%.2f", tD.DLTPR*100),
			city,
			asn_str,
			tD.Loc,
		})
		if err != nil {
			log.Fatalf("Write csv File %v failed with: %v", filePath, err)
		}
	}
	w.Flush()
}

func genDBRecords(verifyResultsSlice []VerifyResults, getLocalAsnAndCity bool) (dbRecords []DBRecord) {
	if len(verifyResultsSlice) > 0 {
		dbRecords = make([]DBRecord, 0)
		ASN, city := 0, ""
		if getLocalAsnAndCity {
			ASN, city, _ = getGeoInfoFromIncolumitas("")
		}
		for _, v := range verifyResultsSlice {
			record := DBRecord{}
			record.Asn = ASN
			record.City = city
			record.Label = Config.SuffixLabel
			record.DS = Config.DTSource
			record.TestTimeStr = v.testTime.Format("2006-01-02 15:04:05")
			record.IP = *v.ip
			record.Loc = *v.loc
			record.DTC = v.dtc
			record.DTPC = v.dtpc
			record.DTPR = v.dtpr
			record.DA = v.da
			record.DMI = v.dmi
			record.DMX = v.dmx
			record.DLTC = v.dltc
			record.DLTPC = v.dltpc
			record.DLTPR = v.dltpr
			record.DLS = v.dls
			record.DLDS = v.dlds
			record.DLTD = v.dltd
			dbRecords = append(dbRecords, record)
		}
	}
	return
}

func printFinalStat(v []VerifyResults, isDtOnly, inSilence bool) {
	// no data for print
	if len(v) == 0 {
		return
	}
	if !inSilence {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight|tabwriter.Debug)
		header := "Time\tIP"
		if !isDtOnly {
			header += "\tSpd(KB/s)\tDLT-T\tDLT-P(%)"
		}
		header += "\tDly-Avg(ms)"
		if !Config.DLTOnly {
			header += "\tDly-Min(ms)\tDly-Max(ms)\tDT-T\tDT-P(%)"
			if Config.EnableDTEvaluation {
				header += "\tStd"
			}
		}
		header += "\t"
		fmt.Fprintln(w, header)
		for i := 0; i < len(v); i++ {
			line := fmt.Sprintf("%s\t%s", v[i].testTime.Format("15:04:05"), *v[i].ip)
			if len(*v[i].loc) > 0 {
				line = fmt.Sprintf("%s#%s", line, *v[i].loc)
			}
			if !isDtOnly {
				line += fmt.Sprintf("\t%.0f\t%d\t%.2f", v[i].dls, v[i].dltc, v[i].dltpr*100)
			}
			line += fmt.Sprintf("\t%.0f", v[i].da)
			if !Config.DLTOnly {
				line += fmt.Sprintf("\t%.0f\t%.0f\t%d\t%.2f", v[i].dmi, v[i].dmx, v[i].dtc, v[i].dtpr*100)
				if Config.EnableDTEvaluation {
					line += fmt.Sprintf("\t%.2f", v[i].daStd)
				}
			}
			line += "\t"
			fmt.Fprintln(w, line)
		}
		fmt.Fprintln(w, "")
		w.Flush()
	} else {
		for i := 0; i < len(v); i++ {
			t_str := *v[i].ip
			if v[i].loc != nil && len(*v[i].loc) > 0 {
				t_str += fmt.Sprintf("#%s", *v[i].loc)
			}
			fmt.Println(t_str)
		}
	}
}

func saveDBRecords(dbRecords []DBRecord, dbFilePath string) {
	if len(dbRecords) > 0 {
		db, err := OpenSqlite(dbFilePath)
		if err != nil {
			myLogger.Errorln("<saveDBRecords> open sqlite error ", err)
			return
		}
		err = AddCFDTRecords(db, dbRecords)
		if err != nil {
			myLogger.Errorln("<saveDBRecords> add CFDT records error ", err)
		}
	}
}
