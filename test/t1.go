package main

import (
	"time"

	sqle "github.com/dolthub/go-mysql-server"
	"github.com/dolthub/go-mysql-server/memory"
	"github.com/dolthub/go-mysql-server/server"
	"github.com/dolthub/go-mysql-server/sql"
	"github.com/dolthub/go-mysql-server/sql/information_schema"
)

// Example of how to implement a MySQL server based on a Engine:
//
// ```
// > mysql --host=127.0.0.1 --port=3306 -u root mydb -e "SELECT * FROM mytable"
// +----------+-------------------+-------------------------------+---------------------+
// | name     | email             | phone_numbers                 | created_at          |
// +----------+-------------------+-------------------------------+---------------------+
// | John Doe | john@doe.com      | ["555-555-555"]               | 2018-04-18 09:41:13 |
// | John Doe | johnalt@doe.com   | []                            | 2018-04-18 09:41:13 |
// | Jane Doe | jane@doe.com      | []                            | 2018-04-18 09:41:13 |
// | Evil Bob | evilbob@gmail.com | ["555-666-555","666-666-666"] | 2018-04-18 09:41:13 |
// +----------+-------------------+-------------------------------+---------------------+
// ```
func main() {
	engine := sqle.NewDefault(
		sql.NewDatabaseProvider(
			createTestDatabase(),
			information_schema.NewInformationSchemaDatabase(),
		))

	config := server.Config{
		Protocol: "tcp",
		Address:  "localhost:3306",
	}

	s, err := server.NewDefaultServer(config, engine)
	if err != nil {
		panic(err)
	}

	s.Start()
}

func createTestDatabase() *memory.Database {
	const (
		dbName    = "mydb"
		tableName = "mytable"
	)

	db := memory.NewDatabase(dbName)
	table := memory.NewTable(tableName, sql.NewPrimaryKeySchema(sql.Schema{
		{Name: "name", Type: sql.Text, Nullable: false, Source: tableName},
		{Name: "email", Type: sql.Text, Nullable: false, Source: tableName},
		{Name: "phone_numbers", Type: sql.JSON, Nullable: false, Source: tableName},
		{Name: "created_at", Type: sql.Timestamp, Nullable: false, Source: tableName},
	}), db.GetForeignKeyCollection())

	creationTime := time.Unix(1524044473, 0).UTC()
	db.AddTable(tableName, table)
	ctx := sql.NewEmptyContext()
	table.Insert(ctx, sql.NewRow("John Doe", "john@doe.com", []string{"555-555-555"}, creationTime))
	table.Insert(ctx, sql.NewRow("John Doe", "johnalt@doe.com", []string{}, creationTime))
	table.Insert(ctx, sql.NewRow("Jane Doe", "jane@doe.com", []string{}, creationTime))
	table.Insert(ctx, sql.NewRow("Evil Bob", "evilbob@gmail.com", []string{"555-666-555", "666-666-666"}, creationTime))
	return db
}
