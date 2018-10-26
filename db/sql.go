package db

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/Benson-Pao/VOD/config"

	"reflect"

	_ "github.com/denisenkom/go-mssqldb"
)

func (d *MSSQL) GetConnectString(server string, user string, password string, port string, database string) string {
	return fmt.Sprintf("server=%s;user id=%s;password=%s;port=%s;database=%s;",
		server, user, password, port, database)

}

type MSSQL struct {
	ConfigInfo *config.ConfigInfo
}

func NewDAL(ConfigInfo *config.ConfigInfo) *MSSQL {
	return &MSSQL{
		ConfigInfo: ConfigInfo,
	}
}

func DB(d *MSSQL) (*sql.DB, error) {

	db, err := sql.Open("mssql",
		d.GetConnectString(d.ConfigInfo.SQL.Server,
			d.ConfigInfo.SQL.User,
			d.ConfigInfo.SQL.Password,
			d.ConfigInfo.SQL.Port,
			d.ConfigInfo.SQL.DataBase))
	defer db.Close()
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func (d *MSSQL) Exec(SQL string, args ...interface{}) (interface{}, error) {
	db, err := DB(d)
	if err != nil {
		return nil, err
	}
	result, err := db.Exec(SQL, args...)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (d *MSSQL) Value(SQL string, args ...interface{}) (interface{}, error) {
	db, err := DB(d)
	if err != nil {
		return nil, err
	}
	var result string
	err = db.QueryRow(SQL, args...).Scan(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (d *MSSQL) Query(SQL string, DataInfo interface{}, args ...interface{}) ([]interface{}, error) {
	db, err := DB(d)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(SQL, args...)
	if err != nil {
		return nil, err
	}

	result := make([]interface{}, 0)
	s := reflect.ValueOf(DataInfo).Elem()

	columns := make([]interface{}, s.NumField())
	for i, _ := range columns {
		columns[i] = s.Field(i).Addr().Interface()
	}

	for rows.Next() {
		err = rows.Scan(columns...)
		if err != nil {
			return nil, err
		}
		result = append(result, s.Interface())
	}

	if err = rows.Close(); err != nil {
		log.Println(err)
		return nil, err
	}
	defer rows.Close()
	return result, nil
}
