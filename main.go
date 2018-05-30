// mysqldiff project main.go
package main

import (
	"database/sql"
	"fmt"

	"dw7758.com/common"
	"dw7758.com/common/config"
	_ "github.com/go-sql-driver/mysql"
)

type Columns struct {
	Name         string
	DefaultValue sql.NullString
	IsNull       string
	DataType     string
	SelfType     string
	Comment      string
	Key          string
}

func (this *Columns) ToString() {
	fmt.Println(fmt.Sprintf(
		"Name:%v, Default:%v, IsNull:%v, DataType:%v, ColType:%v, Comment:%v, Key:%v",
		this.Name, this.DefaultValue.String,
		this.IsNull, this.DataType,
		this.SelfType, this.Comment, this.Key,
	))
}

type MySqlServer struct {
	*sql.DB
}

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

func (this *MySqlServer) ListColumns(table string) ([]*Columns, error) {
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
		return nil, common.NewStackErr(err.Error())
	}
	defer rows.Close()

	cols := make([]*Columns, 0, 40)
	for rows.Next() {
		col := &Columns{}

		err = rows.Scan(
			&col.Name, &col.DefaultValue, &col.IsNull,
			&col.DataType, &col.SelfType, &col.Comment,
			&col.Key,
		)
		if err != nil {
			return nil, common.NewStackErr(err.Error())
		}
		cols = append(cols, col)
	}
	err = rows.Err()
	if err != nil {
		return nil, common.NewStackErr(err.Error())
	}

	return cols, nil
}

func (this *MySqlServer) ListTables() ([]string, error) {
	rows, err := this.Query("select distinct table_name from information_schema.columns where table_schema='dw'")
	if err != nil {
		return nil, common.NewStackErr(err.Error())
	}
	defer rows.Close()

	names := make([]string, 0, 40)
	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		if err != nil {
			return nil, common.NewStackErr(err.Error())
		}
		names = append(names, name)
	}
	err = rows.Err()
	if err != nil {
		return nil, common.NewStackErr(err.Error())
	}

	return names, nil
}

func NewMySqlServer(iniconf config.Configer) (*MySqlServer, error) {
	MysqlHost := iniconf.DefaultString("MysqlHost", "127.0.0.1")
	MysqlPort := iniconf.DefaultInt("MysqlPort", 3306)
	MysqlUser := iniconf.DefaultString("MysqlUser", "dw")
	MysqlPassword := iniconf.DefaultString("MysqlPassword", "dw_wd251")

	mss := &MySqlServer{}

	var err error
	mss.DB, err = sql.Open("mysql", fmt.Sprintf("%v:%v@tcp(%v:%v)/?charset=utf8",
		MysqlUser,
		MysqlPassword,
		MysqlHost,
		MysqlPort,
	))
	if err != nil {
		return nil, common.NewStackErr(err.Error())
	}
	err = mss.Ping()
	if err != nil {
		return nil, common.NewStackErr(err.Error())
	}
	return mss, nil
}

func LoadConfig() error {
	leftConf, err := common.LoadIniConfig("left.conf")
	if err != nil {
		return common.NewStackErr(err.Error())
	}

	leftServer, err := NewMySqlServer(leftConf)
	if err != nil {
		return common.NewStackErr(err.Error())
	}

	rightConf, err := common.LoadIniConfig("right.conf")
	if err != nil {
		return common.NewStackErr(err.Error())
	}

	rightServer, err := NewMySqlServer(rightConf)
	if err != nil {
		return common.NewStackErr(err.Error())
	}

	cols, err := leftServer.ListColumns("dw_user")
	if err != nil {
		return common.NewStackErr(err.Error())
	}

	for _, col := range cols {
		col.ToString()
	}

	_, err = rightServer.ListColumns("dw_user")
	if err != nil {
		return common.NewStackErr(err.Error())
	}

	return nil

	//	leftTables, err := leftServer.ListTables()
	//	if err != nil {
	//		common.ErrorWithStack(err.Error())
	//		return
	//	}

	//	rightTables, err := rightServer.ListTables()
	//	if err != nil {
	//		common.ErrorWithStack(err.Error())
	//		return
	//	}

	//	fmt.Println("leftTables:", leftTables)

	//	fmt.Println("rightTables:", rightTables)
}

func main() {
	fmt.Println("err:", LoadConfig())
}
