package query

import (
	"fmt"
	"log"
	"strings"
)

func checkCap[T any](arr []T) (result []T) {
	if cap(arr) <= len(arr) {
		result = append(make([]T, 0, len(arr)*2), arr...)
	}
	return
}

func PrepareSqlOra(sqlArgs []any) (string, []any) {
	sqlStr := make([]string, 0, len(sqlArgs))
	args := make([]any, 0, len(sqlArgs))
	i := 1
	for _, arg := range sqlArgs {
		switch arg.(type) {
		case Arg:
			if len(arg.(Arg).Sql) != 0 {
				sqlStr = checkCap(sqlStr)
				args = checkCap(args)
				sqlStr = append(sqlStr, fmt.Sprintf("%s:%d", arg.(Arg).Sql, i))
				args = append(args, arg.(Arg).Arg)
				i++
			}
		case []Arg:
			for _, arg1 := range arg.([]Arg) {
				if len(arg1.Sql) != 0 {
					sqlStr = checkCap(sqlStr)
					args = checkCap(args)
					sqlStr = append(sqlStr, fmt.Sprintf("%s:%d", arg1.Sql, i))
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
