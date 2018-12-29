// mysqldiff project mysql.go
package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"

	_ "github.com/go-sql-driver/mysql"
)

/*
| CREATE  TABLE `COLUMNS` (
  `TABLE_CATALOG` varchar(512) NOT NULL DEFAULT '',
  `TABLE_SCHEMA` varchar(64) NOT NULL DEFAULT '',
  `TABLE_NAME` varchar(64) NOT NULL DEFAULT '',
  `COLUMN_NAME` varchar(64) NOT NULL DEFAULT '',
  `ORDINAL_POSITION` bigint(21) unsigned NOT NULL DEFAULT '0',
  `COLUMN_DEFAULT` longtext,
  `IS_NULLABLE` varchar(3) NOT NULL DEFAULT '',
  `DATA_TYPE` varchar(64) NOT NULL DEFAULT '',
  `CHARACTER_MAXIMUM_LENGTH` bigint(21) unsigned DEFAULT NULL,
  `CHARACTER_OCTET_LENGTH` bigint(21) unsigned DEFAULT NULL,
  `NUMERIC_PRECISION` bigint(21) unsigned DEFAULT NULL,
  `NUMERIC_SCALE` bigint(21) unsigned DEFAULT NULL,
  `DATETIME_PRECISION` bigint(21) unsigned DEFAULT NULL,
  `CHARACTER_SET_NAME` varchar(32) DEFAULT NULL,
  `COLLATION_NAME` varchar(32) DEFAULT NULL,
  `COLUMN_TYPE` longtext NOT NULL,
  `COLUMN_KEY` varchar(3) NOT NULL DEFAULT '',
  `EXTRA` varchar(30) NOT NULL DEFAULT '',
  `PRIVILEGES` varchar(80) NOT NULL DEFAULT '',
  `COLUMN_COMMENT` varchar(1024) NOT NULL DEFAULT '',
  `GENERATION_EXPRESSION` longtext NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8 |
*/

func (this *MySqlServer) ShowCreate(table string) (string, error) {
	var name, res string
	err := this.QueryRow(fmt.Sprintf(
		"SHOW CREATE TABLE `%v`.`%v`",
		this.DataBase, table,
	)).Scan(&name, &res)
	if err != nil {
		return "", NewStackErr(err.Error())
	}

	return res, nil
}

func (this *MySqlServer) GetColumn(table, column string) (*Column, error) {
	sql := fmt.Sprintf(`
		select 
		COLUMN_DEFAULT,
		IS_NULLABLE, DATA_TYPE, COLUMN_TYPE, 
		COLUMN_COMMENT,COLUMN_KEY
		from information_schema.columns 
		where table_name = '%v' and column_name = '%v'
	`, table, column)

	col := &Column{
		Table: table,
		Name:  column,
	}

	err := this.QueryRow(sql).Scan(
		&col.DefaultValue, &col.IsNull,
		&col.DataType, &col.SelfType,
		&col.Comment, &col.Key,
	)
	if err != nil {
		return nil, NewStackErr(err.Error())
	}

	return col, nil
}

func (this *MySqlServer) ListColumns(table string) ([]*Column, error) {
	sql := fmt.Sprintf(`
		select 
		COLUMN_NAME, COLUMN_DEFAULT,
		 IS_NULLABLE, DATA_TYPE, COLUMN_TYPE, 
		COLUMN_COMMENT,COLUMN_KEY
		from information_schema.columns 
		where table_name = '%v'
	`, table)
	rows, err := this.Query(sql)
	if err != nil {
		return nil, NewStackErr(err.Error())
	}
	defer rows.Close()

	cols := make([]*Column, 0, 40)
	for rows.Next() {
		col := &Column{Table: table}

		err = rows.Scan(
			&col.Name, &col.DefaultValue, &col.IsNull,
			&col.DataType, &col.SelfType, &col.Comment,
			&col.Key,
		)
		if err != nil {
			return nil, NewStackErr(err.Error())
		}
		cols = append(cols, col)
	}
	err = rows.Err()
	if err != nil {
		return nil, NewStackErr(err.Error())
	}

	return cols, nil
}

func (this *MySqlServer) ListTables() ([]string, error) {
	sql := fmt.Sprintf(`
		select distinct table_name 
		from information_schema.columns 
		where table_schema='%v'
	`, this.DataBase)
	rows, err := this.Query(sql)
	if err != nil {
		return nil, NewStackErr(err.Error())
	}
	defer rows.Close()

	names := make([]string, 0, 40)
	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		if err != nil {
			return nil, NewStackErr(err.Error())
		}
		names = append(names, name)
	}
	err = rows.Err()
	if err != nil {
		return nil, NewStackErr(err.Error())
	}

	return names, nil
}

func NewMySqlServer(jsonConf string) (*MySqlServer, error) {
	jsonBS, err := ioutil.ReadFile(jsonConf)
	if err != nil {
		return nil, NewStackErr(err.Error())
	}

	mss := &MySqlServer{}

	err = json.Unmarshal(bytes.TrimSpace(jsonBS), mss)
	if err != nil {
		return nil, NewStackErr(err.Error())
	}

	fmt.Println("mysql:", mss)

	mss.DB, err = sql.Open(
		"mysql",
		fmt.Sprintf(
			"%v:%v@tcp(%v:%v)/%v?charset=utf8",
			mss.User,
			mss.Password,
			mss.Host,
			mss.Port,
			mss.DataBase,
		),
	)
	if err != nil {
		return nil, NewStackErr(err.Error())
	}

	err = mss.Ping()
	if err != nil {
		return nil, NewStackErr(err.Error())
	}

	mss.SetMaxOpenConns(3)
	mss.SetMaxIdleConns(1)
	return mss, nil
}

func LoadMysql(leftJosn, rightJosn string) error {
	var err error

	G_Left_Server, err = NewMySqlServer(leftJosn)
	if err != nil {
		return err
	}

	G_Right_Server, err = NewMySqlServer(rightJosn)
	if err != nil {
		return err
	}

	return nil
}
