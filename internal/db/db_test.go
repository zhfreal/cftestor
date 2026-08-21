package db

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cftestor/internal/config"
)

func TestSqliteDatabaseOperations(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_ip.db")

	database, err := OpenSqlite(dbPath)
	if err != nil {
		t.Fatalf("OpenSqlite failed: %v", err)
	}

	sqlDB, err := database.DB()
	if err != nil {
		t.Fatalf("database.DB() failed: %v", err)
	}
	defer sqlDB.Close()

	if err := AddTableCFDT(database); err != nil {
		t.Fatalf("AddTableCFDT failed: %v", err)
	}

	records := []DBRecord{
		{
			TestTimeStr: "2026-08-21 12:00:00",
			Asn:         13335,
			City:        "San Francisco",
			Loc:         "SFO",
			IP:          "1.1.1.1:443",
			Label:       "test-node",
			DS:          "https",
			DTC:         4,
			DTPC:        4,
			DTPR:        1.0,
			DA:          25.5,
			DMI:         20.0,
			DMX:         30.0,
			DLTC:        1,
			DLTPC:       1,
			DLTPR:       1.0,
			DLS:         15000.0,
			DLDS:        15000000,
			DLTD:        1.0,
		},
		{
			TestTimeStr: "2026-08-21 12:00:01",
			Asn:         13335,
			City:        "Los Angeles",
			Loc:         "LAX",
			IP:          "1.0.0.1:443",
			Label:       "test-node",
			DS:          "https",
			DTC:         4,
			DTPC:        2,
			DTPR:        0.5,
			DA:          45.0,
			DMI:         40.0,
			DMX:         50.0,
		},
	}

	if err := AddCFDTRecords(database, records); err != nil {
		t.Fatalf("AddCFDTRecords failed: %v", err)
	}

	var fetched []DBRecord
	if err := database.Find(&fetched).Error; err != nil {
		t.Fatalf("database.Find failed: %v", err)
	}

	if len(fetched) != 2 {
		t.Fatalf("expected 2 records in sqlite database, got %d", len(fetched))
	}
	if fetched[0].IP != "1.1.1.1:443" || fetched[0].Loc != "SFO" || fetched[0].Asn != 13335 {
		t.Errorf("record 0 mismatch: %+v", fetched[0])
	}
	if fetched[1].IP != "1.0.0.1:443" || fetched[1].Loc != "LAX" {
		t.Errorf("record 1 mismatch: %+v", fetched[1])
	}
}

func TestWriteCSVResultAndReadBack(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "results.csv")

	records := []DBRecord{
		{
			TestTimeStr: "2026-08-21 12:00:00",
			Asn:         13335,
			City:        "Tokyo",
			Loc:         "NRT",
			IP:          "1.1.1.1:443",
			DS:          "https",
			DTC:         4,
			DTPC:        4,
			DTPR:        1.0,
			DA:          35.2,
			DMI:         30.0,
			DMX:         40.0,
			DLTC:        1,
			DLTPC:       1,
			DLTPR:       1.0,
			DLS:         20000.5,
		},
	}

	if err := WriteCSVResult(records, csvPath); err != nil {
		t.Fatalf("WriteCSVResult failed: %v", err)
	}

	// Read file and parse CSV
	file, err := os.Open(csvPath)
	if err != nil {
		t.Fatalf("os.Open failed: %v", err)
	}
	defer file.Close()

	// Skip BOM if present
	bom := make([]byte, len(config.UTF8BomBytes))
	if _, err := file.Read(bom); err != nil {
		t.Fatalf("reading BOM failed: %v", err)
	}

	reader := csv.NewReader(file)
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("csv.ReadAll failed: %v", err)
	}

	// 1 header row + 1 data row
	if len(rows) != 2 {
		t.Fatalf("expected 2 CSV rows, got %d", len(rows))
	}
	if rows[0][1] != "IP" {
		t.Errorf("expected header column 1 to be IP, got %s", rows[0][1])
	}
	if rows[1][1] != "1.1.1.1:443" {
		t.Errorf("expected data column 1 to be 1.1.1.1:443, got %s", rows[1][1])
	}
}

func TestGenDBRecords(t *testing.T) {
	ip1 := "1.1.1.1:443"
	loc1 := "HKG"
	now := time.Now()

	results := []config.VerifyResults{
		{
			IP:       &ip1,
			Loc:      &loc1,
			TestTime: now,
			Dtc:      4,
			Dtpc:     4,
			Dtpr:     1.0,
			Da:       20.0,
			Dmi:      18.0,
			Dmx:      22.0,
			Dltc:     1,
			Dltpc:    1,
			Dltpr:    1.0,
			Dls:      50000.0,
			Dltd:     1.5,
			Dlds:     75000000,
		},
	}

	records := GenDBRecords(results, false)
	if len(records) != 1 {
		t.Fatalf("expected 1 DBRecord, got %d", len(records))
	}
	rec := records[0]
	if rec.IP != ip1 || rec.Loc != loc1 || rec.DTC != 4 || rec.DTPC != 4 || rec.DLS != 50000.0 {
		t.Errorf("DBRecord mismatch: %+v", rec)
	}
}
