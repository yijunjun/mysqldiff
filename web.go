// mysqldiff project web.go
package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

func MethodRequire(action string, raw http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.ToLower(r.Method) != action {
			http.NotFound(w, r)
			return
		}
		raw(w, r)
	}
}

func Index(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "resource/index.html")
}

var g_table_mux sync.Mutex

type TablePack struct {
	Adds    []string
	Dels    []string
	Changes []string
}

func TableList(w http.ResponseWriter, r *http.Request) {
	g_table_mux.Lock()
	defer g_table_mux.Unlock()

	// 以左边为标准
	leftTables, err := G_Left_Server.ListTables()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rightTables, err := G_Right_Server.ListTables()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tp := &TablePack{}

	// 取公共表
	interTables := make([]string, 0, len(leftTables))

	for _, rt := range rightTables {
		if !IsIn(rt, leftTables) {
			tp.Dels = append(tp.Dels, rt)
		}
	}

	for _, lt := range leftTables {
		if IsIn(lt, rightTables) {
			interTables = append(interTables, lt)
		} else {
			tp.Adds = append(tp.Adds, lt)
		}
	}

	isChange := false
	// 比较公共表
	for _, it := range interTables {
		isChange = false

		leftCols, err := G_Left_Server.ListColumns(it)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		rightCols, err := G_Right_Server.ListColumns(it)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 取公共列
		interColMap := make(map[string]*ColumnPair, len(leftCols))

		for _, rc := range rightCols {
			if IsInColumn(rc, leftCols) {
				interColMap[rc.Name] = &ColumnPair{Right: rc}
			} else {
				tp.Changes = append(tp.Changes, it)
				isChange = true
				break
			}
		}

		if isChange {
			continue
		}

		for _, lc := range leftCols {
			if IsInColumn(lc, rightCols) {
				interColMap[lc.Name].Left = lc
			} else {
				tp.Changes = append(tp.Changes, it)
				isChange = true
				break
			}
		}

		if isChange {
			continue
		}

		// 比较公共列
		for _, cp := range interColMap {
			if !cp.Equal() {
				tp.Changes = append(tp.Changes, it)
				break
			}
		}
	}

	bs, err := json.Marshal(tp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(bs)
}

type ColumnPack struct {
	Table   string
	Adds    []string
	Dels    []string
	Changes []string
}

func ColumnList(w http.ResponseWriter, r *http.Request) {
	g_table_mux.Lock()
	defer g_table_mux.Unlock()

	tableName := r.FormValue("table")

	cp := &ColumnPack{
		Table: tableName,
	}

	leftCols, err := G_Left_Server.ListColumns(tableName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rightCols, err := G_Right_Server.ListColumns(tableName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 取公共列
	interColMap := make(map[string]*ColumnPair, len(leftCols))

	for _, rc := range rightCols {
		if IsInColumn(rc, leftCols) {
			interColMap[rc.Name] = &ColumnPair{Right: rc}
		} else {
			cp.Dels = append(cp.Dels, rc.Name)
		}
	}

	for _, lc := range leftCols {
		if IsInColumn(lc, rightCols) {
			interColMap[lc.Name].Left = lc
		} else {
			cp.Adds = append(cp.Adds, lc.Name)
		}
	}

	// 比较公共列
	for name, icp := range interColMap {
		if !icp.Equal() {
			cp.Changes = append(cp.Changes, name)
		}
	}

	bs, err := json.Marshal(cp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(bs)
}

type FieldPair struct {
	Name string
	Old  string
	Now  string
}

type FieldPack struct {
	Table   string
	Column  string
	Changes []*FieldPair
}

func FieldList(w http.ResponseWriter, r *http.Request) {
	g_table_mux.Lock()
	defer g_table_mux.Unlock()

	tableName := r.FormValue("table")

	columnName := r.FormValue("column")

	fp := &FieldPack{
		Table:  tableName,
		Column: columnName,
	}

	leftCol, err := G_Left_Server.GetColumn(tableName, columnName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rightCol, err := G_Right_Server.GetColumn(tableName, columnName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, lf := range leftCol.Fields() {
		if leftCol.FieldEqual(rightCol, lf) {
			continue
		}

		fp.Changes = append(fp.Changes, &FieldPair{
			Name: lf,
			Old:  rightCol.GetField(lf),
			Now:  leftCol.GetField(lf),
		})
	}

	bs, err := json.Marshal(fp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(bs)
}

type Info struct {
	GitHash string

	CompileTime string

	LeftJson string

	RightJson string

	LeftServer *MySqlServer

	RightServer *MySqlServer

	Now string
}

func Web(addr string) error {
	http.Handle("/", MethodRequire("get", Index))
	http.Handle("/index", MethodRequire("get", Index))
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("resource/assets"))))

	http.Handle("/api/table/list", MethodRequire("get", TableList))

	http.Handle("/api/column/list", MethodRequire("get", ColumnList))

	http.Handle("/api/field/list", MethodRequire("get", FieldList))

	http.Handle(
		"/server/info",
		MethodRequire(
			"get",
			func(w http.ResponseWriter, r *http.Request) {
				bs, err := json.Marshal(&Info{
					GitHash:     GITHASH,
					CompileTime: COMPILETIME,
					LeftJson:    G_Left_Json,
					RightJson:   G_Right_Json,
					LeftServer:  G_Left_Server,
					RightServer: G_Right_Server,
					Now:         time.Now().Format("2006-01-02 15:04:05"),
				})
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Write(bs)
			},
		),
	)

	return http.ListenAndServe(addr, nil)
}
