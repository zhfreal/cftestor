package db

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
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

	for _, tD := range data {
		asnStr, city := "", ""
		if tD.Asn > 0 {
			asnStr = fmt.Sprintf("AS%v", tD.Asn)
			city = tD.City
		}
		if err = w.Write([]string{
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
			asnStr,
			tD.Loc,
		}); err != nil {
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
	if len(verifyResultsSlice) > 0 {
		dbRecords = make([]DBRecord, 0)
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
			record.IP = *v.IP
			record.Loc = *v.Loc
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
			line := fmt.Sprintf("%s\t%s", v[i].TestTime.Format("15:04:05"), *v[i].IP)
			if len(*v[i].Loc) > 0 {
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
			t_str := *v[i].IP
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
	if err = AddCFDTRecords(db, dbRecords); err != nil {
		return fmt.Errorf("failed to add CFTD records to SQLite database %q: %w", dbFilePath, err)
	}
	return nil
}
