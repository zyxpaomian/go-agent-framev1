package mysql

import (
	"database/sql"
	"fmt"
	"util/config"
	ce "util/error"
	log "util/serverlog"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLUtil struct {
	db          *sql.DB
	initialized bool
}

var DB = MySQLUtil{db: nil, initialized: false}

func (m *MySQLUtil) DbInit() {
	m.CloseConn()
	connFormat := "%s:%s@tcp(%s)/%s?autocommit=0&collation=utf8_general_ci&parseTime=true"
	connStr := fmt.Sprintf(
		connFormat,

		config.GlobalConf.GetStr("mysql", "USER_NAME"),
		config.GlobalConf.GetStr("mysql", "USER_PASS"),
		config.GlobalConf.GetStr("mysql", "ADDR_PORT"),
		config.GlobalConf.GetStr("mysql", "DATA_BASE"),
	)

	db, err := sql.Open("mysql", connStr)
	//fmt.Println(db)
	//fmt.Println(err)
	if err != nil {
		log.Errorf("MySQL 初始化失败,失败原因: ", err.Error())
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(1)

	err = db.Ping()
	//fmt.Println(err)
	if err != nil {
		log.Errorf("MySQL Ping失败,失败原因: ", err.Error())
		panic("db 初始化失败")
	}

	m.db = db
	m.initialized = true
	log.Infoln("MySQL 初始化成功")
}

func (m *MySQLUtil) CloseConn() {
	if m.initialized {
		m.db.Close()
		m.db = nil
		m.initialized = false
	}
}

func (m *MySQLUtil) GetConn() *sql.DB {
	if m.initialized == false {
		log.Errorln("MySQL 还未初始化成功")
		return nil
	}
	return m.db
}

func (m *MySQLUtil) GetTx() *sql.Tx {
	if m.initialized == false {
		log.Errorln("MySQL 还未初始化成功")
		return nil
	}
	tx, err := m.db.Begin()
	if err != nil {
		log.Errorln("MySQL 获取TX失败")
		return nil
	}
	return tx
}

func (m *MySQLUtil) SimpleQuery(sql string, args []interface{}, result ...interface{}) (int64, error) {
	if m.initialized == false {
		log.Errorln("MySQL 还未初始化成功")
		return -1, ce.DBError()
	}
	tx := m.GetTx()
	if tx == nil {
		log.Errorln("MySQL 获取TX失败")
		return -1, ce.DBError()
	}
	stmt, err := tx.Prepare(sql)
	if err != nil {
		tx.Rollback()
		log.Errorln("MySQL 获取TX失败: ", err.Error())
		return -1, ce.DBError()
	}
	rows, err := stmt.Query(args...)
	if err != nil {
		log.Errorln("MySQL 查询失败: ", err.Error())
		stmt.Close()
		return -1, ce.DBError()
	}
	var cnt int64 = 0
	for rows.Next() {
		err := rows.Scan(result...)
		if err != nil {
			log.Errorln("MySQL 查询失败: ", err.Error())
			rows.Close()
			stmt.Close()
			tx.Rollback()
			return -1, ce.DBError()
		} else {
			cnt += 1
			break
		}
	}
	err = rows.Err()
	if err != nil {
		log.Errorln("MySQL 查询失败: ", err.Error())
		rows.Close()
		stmt.Close()
		tx.Rollback()
		return -1, ce.DBError()
	}
	rows.Close()
	stmt.Close()
	tx.Commit()
	return cnt, nil
}

func (m *MySQLUtil) AllNoArgQuery(sql string, resultlist []interface{}, result ...interface{}) (int64, error) {
	// func (authdao *AuthDao) GetUserInfo() ([]*structs.UserInfo ,error) {
	// resultlist := []interface{}
	if m.initialized == false {
		log.Errorln("MySQL 还未初始化成功")
		return -1, ce.DBError()
	}
	// resultlist := []*structs.UserInfo{}
	tx := m.GetTx()
	if tx == nil {
		log.Errorln("MySQL 获取TX失败")
		return -1, ce.DBError()
	}
	stmt, err := tx.Prepare(sql)
	if err != nil {
		tx.Rollback()
		log.Errorln("MySQL 获取TX失败: ", err.Error())
		return -1, ce.DBError()
	}
	rows, err := stmt.Query()
	if err != nil {
		log.Errorln("MySQL 查询失败: ", err.Error())
		stmt.Close()
		return -1, ce.DBError()
	}
	for rows.Next() {
		// result := &structs.UserInfo{}
		err := rows.Scan(result...)
		if err != nil {
			log.Errorln("MySQL 查询失败: ", err.Error())
			rows.Close()
			stmt.Close()
			tx.Rollback()
			return -1, ce.DBError()
		} else {
			resultlist = append(resultlist, result)
		}
	}
	rows.Close()
	stmt.Close()
	tx.Commit()
	return 1, nil
}

func (m *MySQLUtil) SimpleInsert(sql string, args ...interface{}) (int, error) {
	if m.initialized == false {
		log.Errorln("MySQL 还未初始化")
		return -1, ce.DBError()
	}
	tx := m.GetTx()
	if tx == nil {
		log.Errorln("MySQL 获取TX失败")
		return -1, ce.DBError()
	}
	stmt, err := tx.Prepare(sql)
	if err != nil {
		tx.Rollback()
		log.Errorln("MySQL Prepare失败: ", err.Error())
		return -1, ce.DBError()
	}
	res, err := stmt.Exec(args...)
	if err != nil {
		stmt.Close()
		tx.Rollback()
		log.Errorln("MySQL 执行Insert失败: ", err.Error())
		return -1, ce.DBError()
	}
	InsertID, _ := res.LastInsertId()
	stmt.Close()
	err = tx.Commit()
	if err != nil {
		log.Errorln("MySQL 执行Insert失败: ", err.Error())
		return -1, ce.DBError()
	}
	return int(InsertID), nil
}

func (m *MySQLUtil) SimpleUpdate(sql string, args ...interface{}) (int, error) {
	if m.initialized == false {
		log.Errorln("MySQL 还未初始化")
		return -1, ce.DBError()
	}
	tx := m.GetTx()
	if tx == nil {
		log.Errorln("MySQL 获取TX失败")
		return -1, ce.DBError()
	}
	stmt, err := tx.Prepare(sql)
	if err != nil {
		tx.Rollback()
		log.Errorln("MySQL Prepare失败: ", err.Error())
		return -1, ce.DBError()
	}
	res, err := stmt.Exec(args...)
	if err != nil {
		stmt.Close()
		tx.Rollback()
		log.Errorln("MySQL 执行Update失败: ", err.Error())
		return -1, ce.DBError()
	}
	AddectIDs, _ := res.RowsAffected()
	stmt.Close()
	err = tx.Commit()
	if err != nil {
		log.Errorln("MySQL 执行Update失败: ", err.Error())
		return -1, ce.DBError()
	}
	return int(AddectIDs), nil
}
