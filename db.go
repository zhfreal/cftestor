package main

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

// TestTime      datetime     测试时间
// ASN           int          测试所使用本地网络的ASN (自动获取)
// CITY          text         测试所在地 (自动获取)
// IP            text         目标CF的IP地址
// LABEL         text         落地服务器标识
// DTS           text         延迟类型(SSL or HTTPS)
// DTC           int          延迟测试次数
// DTPC          int          延迟测试通过次数
// DTPR          float        延迟测试成功率
// DA            float        平均延迟
// DMI           float        最小延迟
// DMX           float        最大延迟
// DLTC          int          下载尝试次数
// DLTPC         int          下载成功次数
// DLTPR         float        下载成功率
// DLSA          float        下载平均速度(KB/s)
// DLDS          int          总下载数据大小(byte)
// DLTD          float        总下载时间(秒)
const (
	DBFile         = "ip.db"
	CreateTableSql = `create table IF NOT EXISTS CFTD (
    TestTime    datetime, 
    ASN         int, 
    CITY        text, 
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
    IP          ,
    LABEL       ,
    DS         ,
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
    values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
)

type cfTestDetail struct {
	testTimeStr string
	asn         int
	city        string
	label       string
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
			&row.testTimeStr,
			&row.asn,
			&row.city,
			&row.ip,
			&row.label,
			dtSource,
			&row.dtc,
			&row.dtpc,
			&row.dtpr,
			&row.da,
			&row.dmi,
			&row.dmx,
			&row.dltc,
			&row.dltpc,
			&row.dltpr,
			&row.dls,
			&row.dlds,
			&row.dltd,
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
