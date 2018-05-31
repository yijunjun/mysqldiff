// mysqldiff project main.go
package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"path"
	"runtime"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

func IsIn(target string, eles []string) bool {
	for _, ele := range eles {
		if target == ele {
			return true
		}
	}
	return false
}

func NewStackErr(reason string) error {
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		return errors.New(reason)
	}
	fun := runtime.FuncForPC(pc)
	return fmt.Errorf(
		"[%v %v:%v] %v",
		path.Base(file),
		fun.Name(),
		line,
		reason,
	)
}

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
	*sql.DB  `json:"-"`
	Title    string
	Host     string
	Port     int32
	User     string
	Password string
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
		return nil, NewStackErr(err.Error())
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
	rows, err := this.Query("select distinct table_name from information_schema.columns where table_schema='dw'")
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

	fmt.Println(
		string(jsonBS),
		strings.TrimSpace(string(jsonBS)),
		string(bytes.TrimSpace(jsonBS)),
	)

	err = json.Unmarshal(bytes.Trim(jsonBS, "\t\n"), mss)
	if err != nil {
		return nil, NewStackErr(err.Error())
	}

	mss.DB, err = sql.Open(
		"mysql",
		fmt.Sprintf(
			"%v:%v@tcp(%v:%v)/?charset=utf8",
			mss.User,
			mss.Password,
			mss.Host,
			mss.Port,
		),
	)
	if err != nil {
		return nil, NewStackErr(err.Error())
	}

	err = mss.Ping()
	if err != nil {
		return nil, NewStackErr(err.Error())
	}
	return mss, nil
}

func LoadConfig(leftJosn, rightJosn string) error {
	leftServer, err := NewMySqlServer(leftJosn)
	if err != nil {
		return NewStackErr(err.Error())
	}

	rightServer, err := NewMySqlServer(rightJosn)
	if err != nil {
		return NewStackErr(err.Error())
	}

	// 以左边为标准
	leftTables, err := leftServer.ListTables()
	if err != nil {
		return NewStackErr(err.Error())
	}

	rightTables, err := rightServer.ListTables()
	if err != nil {
		return NewStackErr(err.Error())
	}

	for _, rt := range rightTables {
		if !IsIn(rt, leftTables) {
			fmt.Println("delete", rt)
		}
	}

	for _, lt := range leftTables {
		if !IsIn(lt, rightTables) {
			fmt.Println("add", lt)
		}
	}

	// 取公共

	cols, err := leftServer.ListColumns("dw_user")
	if err != nil {
		return NewStackErr(err.Error())
	}

	for _, col := range cols {
		col.ToString()
	}

	_, err = rightServer.ListColumns("dw_user")
	if err != nil {
		return NewStackErr(err.Error())
	}

	return nil
}

func main() {
	var leftJson, rightJson string

	flag.StringVar(&leftJson, "leftJson", "left.json", "left json conf")
	flag.StringVar(&rightJson, "rightJson", "right.json", "right json conf")

	flag.Parse()

	fmt.Println(fmt.Sprintf(
		"leftJson:%v, rightJson:%v",
		leftJson, rightJson,
	))

	fmt.Println(LoadConfig(leftJson, rightJson))
}
