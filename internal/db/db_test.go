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
			ECH:         true,
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
	if fetched[0].IP != "1.1.1.1:443" || fetched[0].Loc != "SFO" || fetched[0].Asn != 13335 || !fetched[0].ECH {
		t.Errorf("record 0 mismatch: %+v", fetched[0])
	}
	if fetched[1].IP != "1.0.0.1:443" || fetched[1].Loc != "LAX" || fetched[1].ECH {
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
			ECH:         true,
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
	if rows[0][16] != "ECH" {
		t.Errorf("expected header column 16 to be ECH, got %s", rows[0][16])
	}
	if rows[1][16] != "Y" {
		t.Errorf("expected data column 16 to be Y, got %s", rows[1][16])
	}
}

func TestGenDBRecords(t *testing.T) {
	ip1 := "1.1.1.1:443"
	loc1 := "HKG"
	now := time.Now()

	results := []config.VerifyResults{
		{
			IP:           &ip1,
			Loc:          &loc1,
			ECHSupported: true,
			TestTime:     now,
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
	if rec.IP != ip1 || rec.Loc != loc1 || rec.DTC != 4 || rec.DTPC != 4 || rec.DLS != 50000.0 || !rec.ECH {
		t.Errorf("DBRecord mismatch: %+v", rec)
	}
}

func TestNilIPAndLocHandling(t *testing.T) {
	results := []config.VerifyResults{
		{
			IP:       nil,
			Loc:      nil,
			TestTime: time.Now(),
		},
	}

	records := GenDBRecords(results, false)
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].IP != "" || records[0].Loc != "" {
		t.Errorf("expected empty IP and Loc, got IP=%q Loc=%q", records[0].IP, records[0].Loc)
	}

	PrintFinalStat(results, false, false)
	PrintFinalStat(results, false, true)
}

func TestSqliteMigrationFromLegacyTable(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "legacy_ip.db")

	database, err := OpenSqlite(dbPath)
	if err != nil {
		t.Fatalf("OpenSqlite failed: %v", err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatalf("database.DB() failed: %v", err)
	}
	defer sqlDB.Close()

	// Create legacy table without ECH column
	_, err = sqlDB.Exec(`CREATE TABLE CFTD (
		TestTime TEXT,
		ASN INTEGER,
		CITY TEXT,
		LOC TEXT,
		IP TEXT,
		LABEL TEXT,
		DS TEXT,
		DTC INTEGER,
		DTPC INTEGER,
		DTPR REAL,
		DA REAL,
		DMI REAL,
		DMX REAL,
		DLTC INTEGER,
		DLTPC INTEGER,
		DLTPR REAL,
		DLS REAL,
		DLDS INTEGER,
		DLTD REAL
	);`)
	if err != nil {
		t.Fatalf("create legacy table failed: %v", err)
	}

	// Insert into legacy table using AddCFDTRecords which runs AutoMigrate
	records := []DBRecord{
		{
			IP:  "1.2.3.4:443",
			ECH: true,
		},
	}
	if err := AddCFDTRecords(database, records); err != nil {
		t.Fatalf("AddCFDTRecords on legacy table failed: %v", err)
	}

	var fetched []DBRecord
	if err := database.Find(&fetched).Error; err != nil {
		t.Fatalf("database.Find failed: %v", err)
	}
	if len(fetched) != 1 || !fetched[0].ECH {
		t.Errorf("expected ECH column migrated and set to true, got: %+v", fetched)
	}
}
