package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/joho/godotenv"

	// We don't use this package directly, but we need to register its drivers
	// with the database/sql package
	pq "github.com/lib/pq"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("Usage: ./[executable name] [username] [database name]")
	}
	projectUsername := pq.QuoteIdentifier(os.Args[1])
	projectDatabaseName := pq.QuoteIdentifier(os.Args[2])
	err := godotenv.Load(".env")
	if err != nil {
		log.Print("Failed to load .env file")
		log.Print("Assuming process environment has necessary environment variables")
		log.Fatal(err)
	}
	host := safelyGetEnvVar("DB_host")
	port := safelyGetEnvVar("DB_port")
	adminDatabaseName := safelyGetEnvVar("DB_name")
	adminUser := safelyGetEnvVar("DB_admin_user")
	adminPassword := safelyGetEnvVar("DB_admin_password")
	connectionString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, adminUser, adminPassword, adminDatabaseName)
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	// test connection (sql.Open doesn't actually connect until you query the db)
	if err := db.Ping(); err != nil {
		log.Print("Failed to ping database")
		log.Fatal(err)
	}
	petname.NonDeterministicMode()
	password := petname.Generate(2, "")
	// Can't formatting using $param in tx.Exec because https://github.com/lib/pq/issues/694
	// Have to use our own ad-hoc transactions because postgres doesn't support transactions for "CREATE DATABASE" statements
	createUserQuery := fmt.Sprintf("CREATE USER %s NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT PASSWORD '%s';", projectUsername, password)
	safeExec(db, createUserQuery)

	// we create the createUserRollback query here because if createUserQuery failed, the most likely error
	// is that the user already existed and we don't want to delete an existing user!
	createUserRollback := fmt.Sprintf("DROP USER %s;", projectUsername)

	createDatabaseQuery := fmt.Sprintf("CREATE DATABASE %s WITH OWNER = %s;", projectDatabaseName, projectUsername)
	safeExec(db, createDatabaseQuery, createUserRollback)

	// same concept as the user rollback query. The most likely error when creating a database is that it already exists.
	// so, we only execute the rollback if creating a database was successful, but some other command afterwards fails
	createDatabaseRollback := fmt.Sprintf("DROP DATABASE %s;", projectDatabaseName)

	// don't allow anyone to access the database
	// https://dba.stackexchange.com/questions/17790/created-user-can-access-all-databases-in-postgresql-without-any-grants
	revokeConnectQuery := fmt.Sprintf("REVOKE CONNECT ON DATABASE %s FROM PUBLIC;", projectDatabaseName)
	safeExec(db, revokeConnectQuery, createDatabaseRollback, createUserRollback)

	// allow the project user to access the database
	grantConnectQuery := fmt.Sprintf("GRANT connect ON DATABASE %s TO %s;", projectDatabaseName, projectUsername)
	safeExec(db, grantConnectQuery, createDatabaseRollback, createUserRollback)

	// allow the admin user to access the database
	grantConnectQuery = fmt.Sprintf("GRANT connect ON DATABASE %s TO %s;", projectDatabaseName, pq.QuoteIdentifier(adminUser))
	safeExec(db, grantConnectQuery, createDatabaseRollback, createUserRollback)

	log.Println("All commands executed successfully")
	log.Printf("Created user %s with password \"%s\"\n", projectUsername, password)
	log.Println("Don't lose the password! It is annoying to reset it.", projectUsername, password)
	log.Printf("Created database %s\n", projectDatabaseName)
}

// ad-hoc SQL transactions: accepts a query and a variable number of rollback queries
// if the query fails, it will execute the rollback queries in order, printing any errors
// the rollback queries should undo the current query and all previous queries
func safeExec(db *sql.DB, query string, rollbackQueries ...string) {
	log.Printf("Executing query: %s\n", query)
	_, err := db.Exec(query)
	if err != nil {
		log.Fatalf("Error while executing query: %v\n", err)
		log.Println("Executing rollback queries...")
		success := true
		for _, rollbackQuery := range rollbackQueries {
			log.Printf("Executing rollback query: %s\n", rollbackQuery)
			_, err := db.Exec(rollbackQuery)
			if err != nil {
				success = false
				log.Printf("Error while executing rollback query: %v\n", err)
			}
		}
		if success {
			log.Fatal("Succesfully executed rollback")
		} else {
			log.Fatal("Failed to execute rollback, errors are above")
		}
	}
}

func safelyGetEnvVar(varName string) string {
	value := os.Getenv(varName)
	if len(value) == 0 {
		log.Fatalf("Failed to load environment variable: %s", value)
	}
	return value
}
