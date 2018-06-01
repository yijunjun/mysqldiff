// mysqldiff project common.go
package main

import (
	"database/sql"
	"errors"
	"fmt"
	"path"
	"runtime"
)

func IsIn(target string, eles []string) bool {
	for _, ele := range eles {
		if target == ele {
			return true
		}
	}
	return false
}

func IsInColumn(target *Column, eles []*Column) bool {
	for _, ele := range eles {
		if target.Name == ele.Name {
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

type Column struct {
	Table        string
	Name         string
	DefaultValue sql.NullString
	IsNull       string
	DataType     string
	SelfType     string
	Comment      string
	Key          string
}

func (this *Column) FullName() string {
	return fmt.Sprintf(
		"%v.%v",
		this.Table,
		this.Name,
	)
}

func (this *Column) Fields() []string {
	return []string{
		"DefaultValue",
		"IsNull",
		"DataType",
		"ColumnType",
		"Comment",
		"Key",
	}
}

func (this *Column) Equal(other *Column) bool {
	if this.Table != other.Table {
		return false
	}

	if this.Name != other.Name {
		return false
	}

	if this.DefaultValue != other.DefaultValue {
		return false
	}

	if this.IsNull != other.IsNull {
		return false
	}

	if this.DataType != other.DataType {
		return false
	}

	if this.SelfType != other.SelfType {
		return false
	}

	if this.Comment != other.Comment {
		return false
	}

	if this.Key != other.Key {
		return false
	}

	return true
}

func (this *Column) FieldEqual(other *Column, field string) bool {
	switch field {
	case "DefaultValue":
		return this.DefaultValue == other.DefaultValue
	case "IsNull":
		return this.IsNull == other.IsNull
	case "DataType":
		return this.DataType == other.DataType
	case "ColumnType":
		return this.SelfType == other.SelfType
	case "Comment":
		return this.Comment == other.Comment
	case "Key":
		return this.Key == other.Key
	}
	return false
}

func (this *Column) GetField(field string) string {
	switch field {
	case "DefaultValue":
		if this.DefaultValue.Valid {
			return this.DefaultValue.String
		}
		return "NULL"
	case "IsNull":
		return this.IsNull
	case "DataType":
		return this.DataType
	case "ColumnType":
		return this.SelfType
	case "Comment":
		return this.Comment
	case "Key":
		return this.Key
	}
	return ""
}

func (this *Column) ToString() {
	fmt.Println(fmt.Sprintf(
		"Name:%v, Default:%v, IsNull:%v, DataType:%v, ColType:%v, Comment:%v, Key:%v",
		this.Name, this.DefaultValue.String,
		this.IsNull, this.DataType,
		this.SelfType, this.Comment, this.Key,
	))
}

type ColumnPair struct {
	Left  *Column
	Right *Column
}

func (this *ColumnPair) Equal() bool {
	return this.Left.Equal(this.Right)
}

type MySqlServer struct {
	*sql.DB  `json:"-"`
	Title    string
	Host     string
	Port     int32
	User     string
	Password string
	DataBase string
}

var G_Left_Server *MySqlServer
var G_Right_Server *MySqlServer
