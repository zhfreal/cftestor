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
)

type DBRecord struct {
	TestTimeStr string  `gorm:"column:TestTime"`
	Asn         int     `gorm:"column:ASN"`
	City        string  `gorm:"column:CITY"`
	Loc         string  `gorm:"column:LOC"`
	IP          string  `gorm:"column:IP"`
	Label       string  `gorm:"column:LABEL"`
	DS          string  `gorm:"column:DS"`
	DTC         int     `gorm:"column:DTC"`
	DTPC        int     `gorm:"column:DTPC"`
	DTPR        float64 `gorm:"column:DTPR"`
	DA          float64 `gorm:"column:DA"`
	DMI         float64 `gorm:"column:DMI"`
	DMX         float64 `gorm:"column:DMX"`
	DLTC        int     `gorm:"column:DLTC"`
	DLTPC       int     `gorm:"column:DLTPC"`
	DLTPR       float64 `gorm:"column:DLTPR"`
	DLS         float64 `gorm:"column:DLS"`
	DLDS        int64   `gorm:"column:DLDS"`
	DLTD        float64 `gorm:"column:DLTD"`
}

func (a *DBRecord) TableName() string {
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
	return db.AutoMigrate(&DBRecord{})
}

func AddCFDTRecords(db *gorm.DB, records []DBRecord) error {
	err := AddTableCFDT(db)
	if err != nil {
		return err
	}
	return db.Save(&records).Error
}
