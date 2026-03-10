Connection:

```go
    package main

import (
	"github.com/Payel-git-ol/azure/aurum"
	"github.com/Payel-git-ol/azure/env"
)

func InitDb() {
	env.Load()
	dns := env.MustGet("DNS", "")

	var db *aurum.DatabaseConnection

	db.Connection(dns)
	// Or 
	db.ConnectionDiclarative(aurum.Connection{
		port:     "",
		host:     "",
		name:     "",
		password: "",
		database: "",
	}).Sqlite()

	db.AutoMigrate(&models.User)
}

func GetUserById(id int) int {
	var db *aurum.Aurum[&models.User]
    userId := db.GetById(id)
    return userId
}

func UserCreate(req request.User) {
	var db *aurum.Aurum[&models.User]
    db.ParamCreate(req.Name, req.Age, req.Password //or data)
    db.Create()
}

func UpDateUser(reqUpdate request.User) (string, error) {
	var db *aurum.Aurum[&models.User]
    getUser, err := db.GetData(reqUpdate.Name)
    if err != nill {
        return "", err
    }
    db.UpdateData(getUser, reqUpdate.Age)
    return "User update", nill
}

```