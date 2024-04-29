package main

import (
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
	DBFile    = "ip.db"
	TableName = "CFTD"
	// CreateTableSql = `create table IF NOT EXISTS CFTD (
	// TestTime    datetime,
	// ASN         int,
	// CITY        text,
	// LOC			text,
	// IP          text,
	// LABEL       text,
	// DS          text,
	// DTC         int,
	// DTPC        int,
	// DTPR        float,
	// DA          float,
	// DMI         float,
	// DMX         float,
	// DLTC        int,
	// DLTPC       int,
	// DLTPR       float,
	// DLS         float,
	// DLDS        int,
	// DLTD        float)`
	// InsertDataSqlExp = `insert into CFTD (
	// TestTime    ,
	// ASN         ,
	// CITY        ,
	// LOC			,
	// IP          ,
	// LABEL       ,
	// DS          ,
	// DTC         ,
	// DTPC        ,
	// DTPR        ,
	// DA          ,
	// DMI         ,
	// DMX         ,
	// DLTC        ,
	// DLTPC       ,
	// DLTPR       ,
	// DLS         ,
	// DLDS        ,
	// DLTD        )
	// values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
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
