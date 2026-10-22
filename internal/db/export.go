package db

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"text/tabwriter"

	"cftestor/internal/config"
	"cftestor/internal/utils"
)

func WriteCSVResult(data []DBRecord, filePath string) error {
	var fp *os.File
	var err error
	var w *csv.Writer
	if !utils.FileExists(filePath) {
		fp, err = os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to create CSV file %q: %w", filePath, err)
		}
		wn, wErr := fp.Write(config.UTF8BomBytes)
		if wErr != nil {
			return fmt.Errorf("failed to write UTF-8 BOM to CSV file %q: %w", filePath, wErr)
		}
		if wn != len(config.UTF8BomBytes) {
			return fmt.Errorf("failed to write UTF-8 BOM to CSV file %q: %w", filePath, io.ErrShortWrite)
		}
		w = csv.NewWriter(fp)
		if err = w.Write(config.ResultCsvHeader); err != nil {
			return fmt.Errorf("failed to write CSV header to %q: %w", filePath, err)
		}
	} else {
		fp, err = os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, os.FileMode(0644))
		if err != nil {
			return fmt.Errorf("failed to open CSV file %q: %w", filePath, err)
		}
		w = csv.NewWriter(fp)
	}
	defer func() { _ = fp.Close() }()

	row := make([]string, 16)
	for _, tD := range data {
		asnStr, city := "", ""
		if tD.Asn > 0 {
			asnStr = "AS" + strconv.Itoa(tD.Asn)
			city = tD.City
		}
		row[0] = tD.TestTimeStr
		row[1] = tD.IP
		row[2] = strconv.FormatFloat(tD.DLS, 'f', 2, 64)
		row[3] = strconv.FormatFloat(tD.DA, 'f', 0, 64)
		row[4] = tD.DS
		row[5] = strconv.FormatFloat(tD.DTPR*100, 'f', 2, 64)
		row[6] = strconv.Itoa(tD.DTC)
		row[7] = strconv.Itoa(tD.DTPC)
		row[8] = strconv.FormatFloat(tD.DMI, 'f', 0, 64)
		row[9] = strconv.FormatFloat(tD.DMX, 'f', 0, 64)
		row[10] = strconv.Itoa(tD.DLTC)
		row[11] = strconv.Itoa(tD.DLTPC)
		row[12] = strconv.FormatFloat(tD.DLTPR*100, 'f', 2, 64)
		row[13] = city
		row[14] = asnStr
		row[15] = tD.Loc
		if err = w.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV record to %q: %w", filePath, err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return fmt.Errorf("failed to flush CSV file %q: %w", filePath, err)
	}
	return nil
}

func GenDBRecords(verifyResultsSlice []config.VerifyResults, getLocalAsnAndCity bool) (dbRecords []DBRecord) {
	n := len(verifyResultsSlice)
	if n > 0 {
		dbRecords = make([]DBRecord, 0, n)
		ASN, city := 0, ""
		if getLocalAsnAndCity {
			ASN, city, _ = utils.GetGeoInfoFromIncolumitas("", config.Config.Interval)
		}
		for _, v := range verifyResultsSlice {
			record := DBRecord{}
			record.Asn = ASN
			record.City = city
			record.Label = config.Config.SuffixLabel
			record.DS = config.Config.DTSource
			record.TestTimeStr = v.TestTime.Format("2006-01-02 15:04:05")
			if v.IP != nil {
				record.IP = *v.IP
			}
			if v.Loc != nil {
				record.Loc = *v.Loc
			}
			record.DTC = v.Dtc
			record.DTPC = v.Dtpc
			record.DTPR = v.Dtpr
			record.DA = v.Da
			record.DMI = v.Dmi
			record.DMX = v.Dmx
			record.DLTC = v.Dltc
			record.DLTPC = v.Dltpc
			record.DLTPR = v.Dltpr
			record.DLS = v.Dls
			record.DLDS = v.Dlds
			record.DLTD = v.Dltd
			dbRecords = append(dbRecords, record)
		}
	}
	return
}

func PrintFinalStat(v []config.VerifyResults, isDtOnly, inSilence bool) {
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
		if !config.Config.DLTOnly {
			header += "\tDly-Min(ms)\tDly-Max(ms)\tDT-T\tDT-P(%)"
			if config.Config.EnableDTEvaluation {
				header += "\tStd"
			}
		}
		header += "\t"
		fmt.Fprintln(w, header)
		for i := 0; i < len(v); i++ {
			ipStr := ""
			if v[i].IP != nil {
				ipStr = *v[i].IP
			}
			line := fmt.Sprintf("%s\t%s", v[i].TestTime.Format("15:04:05"), ipStr)
			if v[i].Loc != nil && len(*v[i].Loc) > 0 {
				line = fmt.Sprintf("%s#%s", line, *v[i].Loc)
			}
			if !isDtOnly {
				line += fmt.Sprintf("\t%.0f\t%d\t%.2f", v[i].Dls, v[i].Dltc, v[i].Dltpr*100)
			}
			line += fmt.Sprintf("\t%.0f", v[i].Da)
			if !config.Config.DLTOnly {
				line += fmt.Sprintf("\t%.0f\t%.0f\t%d\t%.2f", v[i].Dmi, v[i].Dmx, v[i].Dtc, v[i].Dtpr*100)
				if config.Config.EnableDTEvaluation {
					line += fmt.Sprintf("\t%.2f", v[i].DaStd)
				}
			}
			line += "\t"
			fmt.Fprintln(w, line)
		}
		fmt.Fprintln(w, "")
		w.Flush()
	} else {
		for i := 0; i < len(v); i++ {
			t_str := ""
			if v[i].IP != nil {
				t_str = *v[i].IP
			}
			if v[i].Loc != nil && len(*v[i].Loc) > 0 {
				t_str += fmt.Sprintf("#%s", *v[i].Loc)
			}
			fmt.Println(t_str)
		}
	}
}

func SaveDBRecords(dbRecords []DBRecord, dbFilePath string) error {
	if len(dbRecords) == 0 {
		return nil
	}
	db, err := OpenSqlite(dbFilePath)
	if err != nil {
		return fmt.Errorf("failed to open SQLite database %q: %w", dbFilePath, err)
	}
	if sqlDB, err := db.DB(); err == nil {
		defer sqlDB.Close()
	}
	if err = AddCFDTRecords(db, dbRecords); err != nil {
		return fmt.Errorf("failed to add CFTD records to SQLite database %q: %w", dbFilePath, err)
	}
	return nil
}
