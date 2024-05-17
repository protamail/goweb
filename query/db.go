package query

import (
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/protamail/goweb/conf"
)

type DB struct {
	DB             *sql.DB
	driverName     string //"oracle", "postgres", etc
	connStr        string
	stmtCache      map[string]*sql.Stmt
}

func RegisterDB(dbName string, driverName string, connStr string) {
	if conf.Debug {
		log.Println("Registering DB:", dbName)
	}
	db, ok := DBMap[dbName]
	if ok && db.DB != nil {
		db.DB.Close()
		for _, stmt := range db.stmtCache {
			stmt.Close()
		}
	}
	DBMap[dbName] = &DB{nil, driverName, connStr, make(map[string]*sql.Stmt)}
}

var DBMap = make(map[string]*DB)

func assertOK(err error, sqlStr string) {
	if err != nil {
		log.Panic(err, "\nSQL: ", sqlStr)
	}
}

// open and cache databases and stmts
func GetDB(connName string) *DB {
	var err error
	d, ok := DBMap[connName]
	if !ok {
		log.Panicf("Unknown database: %s", connName)
	}
	if d.DB == nil {
		if conf.Debug {
			log.Println("Opening", connName, "DB")
		}
		d.DB, err = sql.Open(d.driverName, d.connStr)
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

func TryExec(dbName string, sqlArgs ...any) sql.Result {
	defer func() {
		recover()
	}()
	return Exec(dbName, sqlArgs...)
}

func Exec(dbName string, sqlArgs ...any) sql.Result {
	var err error
	var startTime time.Time
	if conf.Debug {
		startTime = time.Now()
	}
	db := GetDB(dbName)
	sqlStr, args := prepareSqlArgs(db.driverName, sqlArgs)
	//don't prepare statements for DDLs, updates, etc.
	result, err := db.DB.Exec(sqlStr, args...)
	assertOK(err, sqlStr)
	if conf.Debug {
		rows, _ := result.RowsAffected()
		log.Println("Running SQL:", sqlStr, argsToLog(args),
			"\nElapsed:", time.Now().Sub(startTime), "Rows affected:", rows)
	}
	return result
}

func argsToLog(sqlArgs []any) []string {
	result := make([]string, 0, len(sqlArgs))
	for i, arg := range sqlArgs {
		result = append(result, fmt.Sprintf("%v", arg))
		if len(result[i]) > 100 {
			result[i] = result[i][0:90]+"..."+result[i][len(result[i])-10:]
		}
	}
	return result
}

func Val[T any](dbName string, sqlArgs ...any) T {
	return *One[T](dbName, sqlArgs...)
}

func One[T any](dbName string, sqlArgs ...any) *T {
	r := All[T](dbName, sqlArgs...)
	switch len(r) {
	case 1:
		return &r[0]
	case 0:
		return new(T)
	}
	log.Panicf("Query returned more than one row: %v", sqlArgs)
	return nil
}

func All[T any](dbName string, sqlArgs ...any) (result []T) {
	var err error
	var startTime time.Time
	if conf.Debug {
		startTime = time.Now()
	}
	db := GetDB(dbName)
	sqlStr, args := prepareSqlArgs(db.driverName, sqlArgs)
	stmt, ok := db.stmtCache[sqlStr]
	if !ok {
		if conf.Debug {
			log.Println("Preparing stmt for:", sqlStr)
		}
		stmt, err = db.DB.Prepare(sqlStr)
		assertOK(err, sqlStr)
		db.stmtCache[sqlStr] = stmt
	}
	rows, err := stmt.Query(args...)
	assertOK(err, sqlStr)
	defer func() {
		rows.Close()
		assertOK(rows.Err(), sqlStr)
	}()

	colTypes, err := rows.ColumnTypes()
	assertOK(err, sqlStr)
	colNames := make([]string, len(colTypes))
	for i, colType := range colTypes {
		colNames[i] = strings.ReplaceAll(colType.Name(), "_", "")
	}

	var targetRow T
	scanFields := findScanFields(&targetRow, colNames)

	result = make([]T, 0, 10)
	mult := 20 //initial grow factor, subsequently *2
	for rows.Next() {

		err = rows.Scan(scanFields...)
		assertOK(err, sqlStr)

		if cap(result) <= len(result) {
			result = append(make([]T, 0, len(result)*mult), result...)
			mult = 2
		}
		//storing structures inline as opposed to just pointers is less fragmentation,
		//easier to work with, and doesn't have performance disadvantages on results less than 10K rows
		//but even for 1M row results, performance hit is 10-20%
		result = append(result, targetRow)
	}
	if conf.Debug {
		log.Println("Running query:", sqlStr, argsToLog(args),
			"\nElapsed:", time.Now().Sub(startTime), "Rows returned:", len(result))
	}
	return
}

type destField struct {
	kind reflect.Kind
	val  reflect.Value
}

func (dest *destField) Scan(src any) error {
	s, ok := src.([]byte)
	if ok && dest.kind != reflect.Slice {
		//[]byte may be returned for a string, e.g. for pg numeric
		//so if dest is not a slice, convert it to explicit string
		src = string(s)
	}
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
	case []byte:
		dest.val.SetBytes(src.([]byte))
		return nil
	case nil:
		dest.val.SetZero()
		return nil
	case time.Time:
		dest.val.Set(reflect.ValueOf(src))
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
				log.Panicf("Unable to map '%s' column to a %s struct field, make sure the field is: exported, underscore collapsed, case-insensitive",
					colNames[i], targetType.Name())
			}
		}
	} else if len(colNames) == 1 { //case of scalar value return as opposed to struct
		result[0] = &destField{targetValue.Kind(), targetValue}
	} else {
		log.Panicf("Query returns too many columns, expecting 1")
	}
	return
}

func checkCap[T any](arr []T) []T {
	if cap(arr) == len(arr) {
		arr = append(make([]T, 0, len(arr)*2), arr...)
	}
	return arr
}

func prepareSqlArgs(driverName string, sqlArgs []any) (string, []any) {
	sqlStr := make([]string, 0, len(sqlArgs))
	args := make([]any, 0, len(sqlArgs))
	i := 1
	for _, arg := range sqlArgs {
		switch arg.(type) {
		case Arg:
			if len(arg.(Arg).Sql) != 0 {
				sqlStr = checkCap(sqlStr)
				args = checkCap(args)
				switch driverName {
				//trailing space is important
				case "oracle":
					sqlStr = append(sqlStr, fmt.Sprintf("%s:%d ", arg.(Arg).Sql, i))
				case "postgres":
					sqlStr = append(sqlStr, fmt.Sprintf("%s$%d ", arg.(Arg).Sql, i))
				default:
					sqlStr = append(sqlStr, arg.(Arg).Sql+"? ")
				}
				args = append(args, arg.(Arg).Arg)
				i++
			}
		case []Arg:
			for _, arg1 := range arg.([]Arg) {
				if len(arg1.Sql) != 0 {
					sqlStr = checkCap(sqlStr)
					args = checkCap(args)
					switch driverName {
					case "oracle":
						sqlStr = append(sqlStr, fmt.Sprintf("%s:%d ", arg1.Sql, i))
					case "postgres":
						sqlStr = append(sqlStr, fmt.Sprintf("%s?%d ", arg1.Sql, i))
					default:
						sqlStr = append(sqlStr, arg1.Sql+"? ")
					}
					args = append(args, arg1.Arg)
					i++
				}
			}
		case string:
			sqlStr = checkCap(sqlStr)
			sqlStr = append(sqlStr, arg.(string))
		default:
			log.Panicf("Invalid arg type: %T, expecting Arg or string", arg)
		}
	}
	return strings.Join(sqlStr, ""), args
}
