package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestTime      datetime     when the test happened
// ASN           int          ASN of your local network
// CITY          text         city of your local network
// IP            text         valid IP for CloudFare CDN access
// LABEL         text         label while stand for your CloudFare CDN resources
// DTS           text         the method for DT (SSL or HTTPS)
// DTC           int          tries for DT
// DTPC          int          success count of DT
// DTPR          float        success rate of DT
// DA            float        average delay of DT
// DMI           float        minimal delay of DT
// DMX           float        maximum delay of DT
// DLTC          int          tries for DLT
// DLTPC         int          success count of DLT
// DLTPR         float        success rate of DLT
// DLSA          float        average download speed (KB/s)
// DLDS          int          total bytes downloaded
// DLTD          float        total times escapted during download (in second)
const (
	DBFile         = "ip.db"
	TableName      = "CFTD"
	CreateTableSql = `create table IF NOT EXISTS CFTD (
    TestTime    datetime, 
    ASN         int, 
    CITY        text,
	LOC			text,
    IP          text, 
    LABEL       text,
    DS          text,
    DTC         int,
    DTPC        int,
    DTPR        float,
    DA          float,
    DMI         float,
    DMX         float,
    DLTC        int,
    DLTPC       int,
    DLTPR       float,
    DLS         float,
    DLDS        int,
    DLTD        float)`
	InsertDataSqlExp = `insert into CFTD (
    TestTime    ,
    ASN         ,
    CITY        ,
	LOC			,
    IP          ,
    LABEL       ,
    DS          ,
    DTC         ,
    DTPC        ,
    DTPR        ,
    DA          ,
    DMI         ,
    DMX         ,
    DLTC        ,
    DLTPC       ,
    DLTPR       ,
    DLS         ,
    DLDS        ,
    DLTD        )
    values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
)

type dBRecord struct {
	testTimeStr string  `gorm:"column:TestTime"`
	asn         int     `gorm:"column:ASN"`
	city        string  `gorm:"column:CITY"`
	loc         string  `gorm:"column:LOC"`
	ip          string  `gorm:"column:IP"`
	label       string  `gorm:"column:LABEL"`
	ds          string  `gorm:"column:DS"`
	dtc         int     `gorm:"column:DTC"`
	dtpc        int     `gorm:"column:DTPC"`
	dtpr        float64 `gorm:"column:DTPR"`
	da          float64 `gorm:"column:DA"`
	dmi         float64 `gorm:"column:DMI"`
	dmx         float64 `gorm:"column:DMX"`
	dltc        int     `gorm:"column:DLTC"`
	dltpc       int     `gorm:"column:DLTPC"`
	dltpr       float64 `gorm:"column:DLTPR"`
	dls         float64 `gorm:"column:DLS"`
	dlds        int64   `gorm:"column:DLDS"`
	dltd        float64 `gorm:"column:DLTD"`
}

func (a *dBRecord) TableName() string {
	return TableName
}

func OpenSqlite(dbfile string) (*gorm.DB, error) {
	dial := sqlite.Open(dbfile)
	return gorm.Open(dial, &gorm.Config{
		NowFunc: func() time.Time {
			return time.Now()
		},
		// NamingStrategy: schema.NamingStrategy{
		//     TablePrefix: config.Table_Prefix,
		// },
	})

}

func AddTableCFDT(db *gorm.DB) error {
	return db.AutoMigrate(&dBRecord{})
}

func AddCFDTRecords(db *gorm.DB, records []dBRecord) error {
	err := AddTableCFDT(db)
	if err != nil {
		return err
	}
	return db.Save(&records).Error
}

type cfTestDetail struct {
	testTimeStr string
	asn         int
	city        string
	label       string
	loc         string // location
	VerifyResults
}

func openDB(dbFile string) *sql.DB {
	if len(dbFile) == 0 {
		dbFile = DBFile
	}
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		log.Fatalf("%v\n", err)
	}
	return db
}

func dbExec(db *sql.DB, sql string, closeDB bool) *sql.Result {
	if closeDB {
		defer func() { _ = db.Close() }()
	}
	r, err := db.Exec(sql)
	if err != nil {
		log.Fatalf("%v\n", err)
	}
	return &r
}

func openTable(dbFile string) *sql.DB {
	db := openDB(dbFile)
	_ = dbExec(db, CreateTableSql, false)
	return db
}

func QueryData(sql string, dbFile string) *[]cfTestDetail {
	db := openTable(dbFile)
	cfDetails := make([]cfTestDetail, 0)
	rows, err := db.Query(sql)
	if err != nil {
		log.Fatalf("%v\n", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var tmpDetail cfTestDetail
		err = rows.Scan(&tmpDetail.testTimeStr,
			&tmpDetail.asn,
			&tmpDetail.city,
			&tmpDetail.loc,
			&tmpDetail.ip,
			&tmpDetail.label,
			&tmpDetail.dtc,
			&tmpDetail.dtpc,
			&tmpDetail.dtpr,
			&tmpDetail.da,
			&tmpDetail.dmi,
			&tmpDetail.dmx,
			&tmpDetail.dltc,
			&tmpDetail.dltpc,
			&tmpDetail.dltpr,
			&tmpDetail.dls,
			&tmpDetail.dlds,
			&tmpDetail.dltd)
		if err != nil {
			log.Fatal(err)
		}
		cfDetails = append(cfDetails, tmpDetail)
	}
	return &cfDetails
}

func insertData(details []cfTestDetail, dbFile string) bool {
	if len(details) == 0 {
		return true
	}
	db := openTable(dbFile)
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	stmt, err := tx.Prepare(InsertDataSqlExp)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = stmt.Close()
	}()
	// TestTime
	// ASN
	// CITY
	// LOC
	// IP
	// LABEL
	// DTS
	// DTC
	// DTPC
	// DTPR
	// DA
	// DMI
	// DMX
	// DLTC
	// DLTPC
	// DLTPR
	// DLSA
	// DLDS
	// DLTD
	for _, row := range details {
		_, err = stmt.Exec(
			row.testTimeStr,
			fmt.Sprintf("AS%v", &row.asn),
			row.city,
			row.loc,
			*(row.ip),
			row.label,
			dtSource,
			row.dtc,
			row.dtpc,
			row.dtpr,
			row.da,
			row.dmi,
			row.dmx,
			row.dltc,
			row.dltpc,
			row.dltpr,
			row.dls,
			row.dlds,
			row.dltd,
		)
		if err != nil {
			log.Fatal(err)
		}
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
	return true
}
