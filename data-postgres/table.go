package data_postgres


import (
	. "github.com/yatlabs/bigger"
	"fmt"
	"strings"
	"time"
	"errors"
)

type (
	PostgresTable struct {
		PostgresView
	}
)


//创建对象
func (table *PostgresTable) Create(data Map) (Map) {
	table.base.lastError = nil

	// var err *Error

	//按字段生成值
	value := Map{}
	errm := Bigger.Mapping(table.fields, data, value, false, false)
	if errm != nil {
		table.base.error("data.create.parse", errm, table.name, value)
		return nil
	}

	//对拿到的值进行包装，以适合postgres
	newValue := table.base.packing(value)

	//先拿字段列表
	keys, tags, vals := []string{}, []string{}, make([]interface{},0)
	i := 1
	for k,v := range newValue {
		if k == table.key {
			if v == nil {
				continue
			}
			//id不直接跳过,可以指定ID
			//continue
		}
		keys = append(keys, k)
		vals = append(vals, v)
		tags = append(tags, fmt.Sprintf("$%d", i))
		i++
	}

	exec,err := table.base.begin()
	if err != nil {
		table.base.error("data.create.begin", err, table.name)
		return nil
	}

	sql := fmt.Sprintf(`INSERT INTO "%s"."%s" ("%s") VALUES (%s) RETURNING "%s";`, table.schema, table.view, strings.Join(keys, `","`), strings.Join(tags, `,`), table.key)
	row := exec.QueryRow(sql, vals...)
	if row == nil {
		table.base.error("data.create.queryrow", errors.New("无返回行"), table.name)
		return nil
	}

	id := int64(0)
	err = row.Scan(&id)
	if err != nil {
		table.base.error("data.create.scan", err, table.name, sql)
		return nil
	}
	value[table.key] = id


	//注意这里，如果手动提交事务， 那这里直接返回，是不需要提交的
	if table.base.manual {

		//这里应该保存触发器
		table.base.trigger(EventDataCreate, Map{ "base": table.base.name, "table": table.name, "entity": value })

		//成功了，但是没有提交事务
		return value

	} else {

		//这里应该有触发器
		// TRIGGER.Touch(TriggerCreate, Map{ "base": table.base.name, "table": table.name, "entity": value })

		//成功了
		return value
	}



}

//修改对象
func (table *PostgresTable) Change(item Map, data Map) (Map) {
	table.base.lastError = nil

	if item == nil || item[table.key] == nil {
		table.base.error("data.change.empty", errors.New("无效数据"), table.name)
		return nil
	}

	//记录修改时间
	if data[FieldChanged] == nil {
		data[FieldChanged] = time.Now()
	}

	//按字段生成值
	value := Map{}
	errm := Bigger.Mapping(table.fields, data, value, true, false)
	if errm != nil {
		table.base.error("data.change.parse", errm, table.name)
		return nil
	}

	//包装值，因为golang本身数据类型和数据库的不一定对版
	//需要预处理一下
	newValue := table.base.packing(value)

	if inc,ok := data[INC]; ok {
		newValue[INC] = inc
	}

	//先拿字段列表
	sets, vals := []string{}, make([]interface{}, 0)
	i := 1
	for k, v := range newValue {
		//主值不在修改之中
		if k == table.key {
			continue
		} else if k == INC {
			if vm,ok := v.(Map); ok {
				for kk,vv := range vm {
					vals = append(vals, vv)
					sets = append(sets, fmt.Sprintf(`"%s"="%s"+$%d`, kk, kk, i))
					i++
				}
			}
		} else {
			//keys = append(keys, k)
			vals = append(vals, v)
			sets = append(sets, fmt.Sprintf(`"%s"=$%d`, k, i))
			i++
		}
	}
	//条件是主键
	vals = append(vals, item[table.key])

	//开启事务
	exec,err := table.base.begin()
	if err != nil {
		table.base.error("data.change.begin", err, table.name)
		return nil
	}

	//更新数据库
	sql := fmt.Sprintf(`UPDATE "%s"."%s" SET %s WHERE "%s"=$%d`, table.schema, table.view, strings.Join(sets, `,`), table.key, i)
	_, err = exec.Exec(sql, vals...)
	if err != nil {
		table.base.error("data.change.exec", err, table.name, sql, vals)
		return nil
	}
	//几行被改不需要显示
	//result.RowsAffected()

	//LOGGER.Logger.Error("change", "exec", sql, vals, cccc, err)

	// 不改item
	// 先复制item
	newItem := Map{}
	for k, v := range item { newItem[k] = v }
	for k, v := range value { newItem[k] = v }


	
	table.base.trigger(EventDataChange, Map{ "base": table.base.name, "table": table.name, "before": item, "after": newItem })

	return newItem

	// //注意这里，如果手动提交事务， 那这里直接返回，是不需要提交的
	// if table.base.manual {
	// 	//成功了，但是没有提交事务
	// 	return newItem
	// } else {
	// 	return newItem
	// }

}

//删除对象
// func (table *PostgresTable) Remove(items ...Map) int64 {
// 	table.base.lastError = nil

// 	ids := make([]string, 0)
// 	for _,item := range items {
// 		if item[StatusField] == nil && item[table.key] != nil {
// 			ids = append(ids, fmt.Sprintf("%v", item[table.key]))
// 		}
// 	}

// 	if len(ids) == 0 {
// 		table.base.error("data.remove.empty", Bigger.Erring("无效数据"), table.name)
// 		return int64(0)
// 	}



// 	nums := []string{}
// 	args := []Any{
// 		StatusRemoved,
// 	}
// 	for i,id := range ids {
// 		nums = append(nums, fmt.Sprintf("$%d", i+2))
// 		args = append(args, id)
// 	}


// 	//开启事务
// 	exec,err := table.base.begin()
// 	if err != nil {
// 		table.base.error("data.remove.begin", err, table.name)
// 		return int64(0)
// 	}

// 	//更新数据库
// 	sql := fmt.Sprintf(`UPDATE "%s"."%s" SET "status"=$1 WHERE "%s" IN(%s)`, table.schema, table.view, table.key, strings.Join(nums, ","))
// 	result, err := exec.Exec(sql, args...)
// 	if err != nil {
// 		table.base.error("data.remove.exec", err, table.name, ids)
// 		return int64(0)
// 	}

// 	for _,item := range items {
// 		item[StatusField] = StatusRemoved
// 	}
	
// 	affected := int64(0)
// 	if val,err := result.RowsAffected(); err == nil {
// 		affected = val
// 	}

// 	//注意这里，如果手动提交事务， 那这里直接返回，是不需要提交的
// 	if table.base.manual {
// 		//成功了，但是没有提交事务
// 	} else {
// 		//
// 	}

// 	return affected

// }


//恢复对象
// func (table *PostgresTable) Recover(items ...Map) (int64) {
// 	table.base.lastError = nil

// 	ids := []string{}
// 	for _,item := range items {
// 		if item[StatusField] != nil && item[table.key] != nil {
// 			ids = append(ids, fmt.Sprintf("%v", item[table.key]))
// 		}
// 	}

// 	if len(ids) == 0 {
// 		table.base.error("data.remove.empty", Bigger.Erring("无效数据"), table.name)
// 		return int64(0)
// 	}

// 	nums := []string{}
// 	args := []Any{ }
// 	for i,id := range ids {
// 		nums = append(nums, fmt.Sprintf("$%d", i+1))
// 		args = append(args, id)
// 	}

// 	//开启事务
// 	exec,err := table.base.begin()
// 	if err != nil {
// 		table.base.error("data.recover.begin", err, table.name)
// 		return int64(0)
// 	}

// 	//更新数据库
// 	sql := fmt.Sprintf(`UPDATE "%s"."%s" SET "status"=NULL WHERE "%s" IN(%s)`, table.schema, table.view, table.key, strings.Join(nums, ","))
// 	result, err := exec.Exec(sql, args...)
// 	if err != nil {
// 		table.base.error("data.recover.exec", err, table.name, ids)
// 		return int64(0)
// 	}

// 	for _,item := range items {
// 		item[StatusField] = nil
// 	}

// 	affected := int64(0)
// 	if val,err := result.RowsAffected(); err == nil {
// 		affected = val
// 	}

// 	//注意这里，如果手动提交事务， 那这里直接返回，是不需要提交的
// 	if table.base.manual {
// 		//成功了，但是没有提交事务
// 	} else {
// 		//
// 	}

// 	return affected

// }


//逻辑删除和恢复已经抛弃
//这两个功能应该是逻辑层干的事，不应和驱动混在一起
//此为物理删除单条记录，并返回记录，所以要先查询单条
func (table *PostgresTable) Remove(args ...Any) (Map) {
	table.base.lastError = nil

	//如果args是传整个item来，那只要处理id就行了
	if len(args) == 1 {
		if args[0] == nil {
			table.base.error("data.remove.empty", errors.New("无效数据"), table.name)
			return nil
		}
		if vv,ok := args[0].(Map); ok {
			if id,ok := vv[table.key]; ok {
				args = []Any{
					Map{ table.key: id },
				}
			}
		}
	}

	item := table.First(args...)
	if err := table.base.Erred(); err != nil {
		table.base.error("data.remove.first", err, table.name)
		return nil
	}

	//开启事务
	exec,err := table.base.begin()
	if err != nil {
		table.base.error("data.remove.begin", err, table.name)
		return nil
	}

	sql := fmt.Sprintf(`DELETE FROM "%s"."%s" WHERE %s=$1`, table.schema, table.view, table.key)
	_,err = exec.Exec(sql, item[table.key])
	if err != nil {
		table.base.error("data.delete.begin", err, table.name, sql, item[table.key])
		return nil
	}

	table.base.trigger(EventDataRemove, Map{ "base": table.base.name, "table": table.name, "entity": item })
	
	//注意这里，如果手动提交事务， 那这里直接返回，是不需要提交的
	// if table.base.manual {
	// 	//成功了，但是没有提交事务
	// } else {
	// 	//
	// }

	return item
}


//批量删除，这可是真删
func (table *PostgresTable) Delete(args ...Any) (int64) {
	table.base.lastError = nil

	//生成条件
	where,builds,_,err := table.base.parsing(1,args...)
	if err != nil {
		table.base.error("data.delete.parse", err, table.name)
		return int64(0)
	}

	//开启事务
	exec,err := table.base.begin()
	if err != nil {
		table.base.error("data.delete.begin", err, table.name)
		return int64(0)
	}

	sql := fmt.Sprintf(`DELETE FROM "%s"."%s" WHERE %s`, table.schema, table.view, where)
	result,err := exec.Exec(sql, builds...)
	if err != nil {
		table.base.error("data.delete.begin", err, table.name, sql, builds)
		return int64(0)
	}
	
	affected := int64(0)
	if val,err := result.RowsAffected(); err != nil {
		table.base.error("data.update.affected", err, table.name)
	} else {
		affected = val
	}

	//注意这里，如果手动提交事务， 那这里直接返回，是不需要提交的
	// if table.base.manual {
	// 	//成功了，但是没有提交事务
	// } else {
	// 	//
	// }

	return affected
}


//批量更新，直接更了， 没有任何relate相关处理的
func (table *PostgresTable) Update(update Map, args ...Any) (int64) {
	table.base.lastError = nil

	//注意，args[0]为更新的内容，之后的为查询条件
	//sets := args[0]
	//args = args[1:]

	// var err *Error

	//按字段生成值
	value := Map{}
	errm := Bigger.Mapping(table.fields, update, value, true, false)
	if errm != nil {
		table.base.error("data.delete.mapping", errm, table.name)
		return int64(0)
	}

	//包装值，因为golang本身数据类型和数据库的不一定对版
	//需要预处理一下
	newValue := table.base.packing(value)

	if inc,ok := update[INC]; ok {
		newValue[INC] = inc
	}

	//先拿字段列表
	sets, vals := []string{}, make([]interface{}, 0)
	i := 1
	for k, v := range newValue {
		//主值不在修改之中
		if k == table.key {
			continue
		} else if k == INC {
			if vm,ok := v.(Map); ok {
				for kk,vv := range vm {
					vals = append(vals, vv)
					sets = append(sets, fmt.Sprintf(`"%s"="%s"+$%d`, kk, kk, i))
					i++
				}
			}
		} else {
			//keys = append(keys, k)
			vals = append(vals, v)
			sets = append(sets, fmt.Sprintf(`"%s"=$%d`, k, i))
			i++
		}
	}

		//生成条件
	where,builds,_,err := table.base.parsing(i, args...)
	if err != nil {
		table.base.error("data.delete.parse", err, table.name)
		return int64(0)
	}

	//把builds的args加到vals中
	for _,v := range builds {
		vals = append(vals, v)
	}

	//开启事务
	exec, err := table.base.begin()
	if err != nil {
		table.base.error("data.update.begin", err, table.name)
		return int64(0)
	}

	//更新数据库
	sql := fmt.Sprintf(`UPDATE "%s"."%s" SET %s WHERE %s`, table.schema, table.view, strings.Join(sets, `,`), where)
	result, err := exec.Exec(sql, vals...)
	if err != nil {
		table.base.error("data.update.begin", err, table.name, sql, vals)
		return int64(0)
	}

	affected := int64(0)
	if val,err := result.RowsAffected(); err != nil {
		table.base.error("data.update.affected", err, table.name)
	} else {
		affected = val
	}

	//注意这里，如果手动提交事务， 那这里直接返回，是不需要提交的
	// if table.base.manual {
	// 	//成功了，但是没有提交事务
	// } else {
	// 	//这是真成功了
	// }

	return affected
}





