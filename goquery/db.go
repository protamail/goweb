package query

import (
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type prepareSQLFunc func([]any) (string, []any)
type DB struct {
	DB             *sql.DB
	DriverName     string //"oracle", "pgx", etc
	ConnStr        string
	PrepareSqlArgs prepareSQLFunc
	StmtCache      map[string]*sql.Stmt
}

var Debug = false

func NewDB(driverName string, connStr string, f prepareSQLFunc) *DB {
	return &DB{nil, driverName, connStr, f, make(map[string]*sql.Stmt)}
}

var DBMap = make(map[string]*DB)

func ResetDBMap() {
	if Debug {
		log.Println("Initializing DBs")
	}
	for _, db := range DBMap {
		if db.DB != nil {
			db.DB.Close()
			for _, stmt := range db.StmtCache {
				stmt.Close()
			}
		}
	}
	DBMap = make(map[string]*DB)
}

func assertOK(err error) {
	if err != nil {
		log.Panic(err)
	}
}

// open and cache databases and stmts
func GetDB(connName string) *DB {
	var err error
	d, ok := DBMap[connName]
	if !ok {
		log.Panicf("Unknown conn name: %s", connName)
	}
	if d.DB == nil {
		if Debug {
			log.Println("Opening DB", connName, d.ConnStr)
		}
		d.DB, err = sql.Open(d.DriverName, d.ConnStr)
		if err != nil {
			d.DB = nil
			log.Panic(err)
		}
	}
	return d
}

type Arg struct {
	Sql string
	Arg any
}

func WithArg(sql string, arg any) Arg {
	return Arg{sql, arg}
}

func Exec(db *DB, sqlArgs ...any) sql.Result {
	var err error
	var startTime time.Time
	if Debug {
		startTime = time.Now()
	}
	sqlStr, args := db.PrepareSqlArgs(sqlArgs)
	stmt, ok := db.StmtCache[sqlStr]
	if !ok {
		if Debug {
			log.Println("Preparing stmt for:", sqlStr)
		}
		stmt, err = db.DB.Prepare(sqlStr)
		assertOK(err)
		db.StmtCache[sqlStr] = stmt
	}
	result, err := stmt.Exec(args...)
	assertOK(err)
	if Debug {
		rows, _ := result.RowsAffected()
		log.Println("Running SQL:", sqlStr, args, "Rows affected:", rows,
			"Elapsed:", time.Now().Sub(startTime))
	}
	return result
}

func Val[T any](db *DB, sqlArgs ...any) T {
	return *One[T](db, sqlArgs...)
}

func One[T any](db *DB, sqlArgs ...any) *T {
	r := All[T](db, sqlArgs...)
	switch len(r) {
	case 1:
		return r[0]
	case 0:
		return new(T)
	}
	log.Panicf("Query returned more than one row: %v", sqlArgs)
	return nil
}

func All[T any](db *DB, sqlArgs ...any) (result []*T) {
	var err error
	var startTime time.Time
	if Debug {
		startTime = time.Now()
	}
	sqlStr, args := db.PrepareSqlArgs(sqlArgs)
	stmt, ok := db.StmtCache[sqlStr]
	if !ok {
		if Debug {
			log.Println("Preparing stmt for:", sqlStr)
		}
		stmt, err = db.DB.Prepare(sqlStr)
		assertOK(err)
		db.StmtCache[sqlStr] = stmt
	}
	rows, err := stmt.Query(args...)
	assertOK(err)
	defer func() {
		rows.Close()
		assertOK(rows.Err())
	}()

	colTypes, err := rows.ColumnTypes()
	assertOK(err)
	colNames := make([]string, len(colTypes))
	for i, colType := range colTypes {
		colNames[i] = strings.ReplaceAll(colType.Name(), "_", "")
	}

	var targetRow T
	scanFields := findScanFields(&targetRow, colNames)

	result = make([]*T, 0, 10)
	mult := 20 //initial grow factor, subsequently *2
	for rows.Next() {

		err = rows.Scan(scanFields...)
		assertOK(err)

		if cap(result) <= len(result) {
			result = append(make([]*T, 0, len(result)*mult), result...)
			mult = 2
		}
		resultRow := new(T)
		*resultRow = targetRow
		result = append(result, resultRow)
	}
	if Debug {
		log.Println("Running query:", sqlStr, args, "Rows returned:", len(result),
			"Elapsed:", time.Now().Sub(startTime))
	}
	return
}

type destField struct {
	kind reflect.Kind
	val  reflect.Value
}

func (dest *destField) Scan(src any) error {
	switch src.(type) {
	case string:
		switch dest.kind {
		case reflect.String:
			dest.val.SetString(src.(string))
			return nil
		case reflect.Int64, reflect.Int, reflect.Int32, reflect.Int16, reflect.Int8:
			i, err := strconv.ParseInt(src.(string), 10, 64)
			if err == nil {
				dest.val.SetInt(i)
				return nil
			}
		case reflect.Float64, reflect.Float32:
			f, err := strconv.ParseFloat(src.(string), 64)
			if err == nil {
				dest.val.SetFloat(f)
				return nil
			}
		case reflect.Bool:
			if strings.EqualFold(src.(string), "Y") {
				dest.val.SetBool(true)
				return nil
			} else if strings.EqualFold(src.(string), "N") {
				dest.val.SetBool(false)
				return nil
			}
			b, err := strconv.ParseBool(src.(string))
			if err == nil {
				dest.val.SetBool(b)
				return nil
			}
		}
	case nil:
		dest.val.SetZero()
		return nil
	case time.Time:
		dest.val.Set(reflect.ValueOf(src))
		return nil
	case []byte:
		dest.val.SetBytes(src.([]byte))
		return nil
	case bool:
		dest.val.SetBool(src.(bool))
		return nil
	case float64:
		switch dest.kind {
		case reflect.Float64, reflect.Float32:
			dest.val.SetFloat(src.(float64))
			return nil
		case reflect.String:
			dest.val.SetString(strconv.FormatFloat(src.(float64), 'f', -1, 64))
			return nil
		}
	case int64:
		switch dest.kind {
		case reflect.Int64, reflect.Int, reflect.Int32, reflect.Int16, reflect.Int8:
			dest.val.SetInt(src.(int64))
			return nil
		case reflect.String:
			dest.val.SetString(strconv.FormatInt(src.(int64), 10))
			return nil
		}
	}
	return fmt.Errorf("Can't convert '%s' %T to %s", src, src, dest.val.Type())
}

func findScanFields(targetPtr any, colNames []string) (result []any) {
	result = make([]any, len(colNames))
	targetValue := reflect.ValueOf(targetPtr).Elem()
	targetType := targetValue.Type()

	if targetValue.Kind() == reflect.Struct {
		for _, structField := range reflect.VisibleFields(targetType) {
			fieldValue := targetValue.FieldByIndex(structField.Index)
			if structField.IsExported() {
				for i, colName := range colNames {
					if strings.EqualFold(strings.ReplaceAll(structField.Name, "_", ""), colName) {
						result[i] = &destField{fieldValue.Kind(), fieldValue}
					}
				}
			}
		}
		for i, t := range result {
			if t == nil {
				log.Panicf("Unable to map %s column to a %s struct field", colNames[i], targetType.Name())
			}
		}
	} else if len(colNames) == 1 { //case of scalar value return as opposed to struct
		result[0] = &destField{targetValue.Kind(), targetValue}
	} else {
		log.Panicf("Query returns too many columns, expecting 1")
	}
	return
}
