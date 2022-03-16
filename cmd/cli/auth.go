package main

import (
	"fmt"
	"runtime"
	"time"

	"github.com/fatih/color"
)

func doAuth() error {
	// migrations
	dbType := grv.DB.DataBaseType
	fileName := fmt.Sprintf("%d_create_auth_tables", time.Now().UnixMicro())

	var (
		upFile   string
		downFile string
	)
	if runtime.GOOS == "windows" {
		upFile = grv.RootPath + "\\migrations\\" + fileName + ".up.sql"
		downFile = grv.RootPath + "\\migrations\\" + fileName + ".down.sql"
	} else {
		upFile = grv.RootPath + "/migrations/" + fileName + ".up.sql"
		downFile = grv.RootPath + "/migrations/" + fileName + ".down.sql"
	}

	fmt.Println(dbType, upFile, downFile)

	err := copyFileFromTemplate("templates/migrations/auth_tables."+dbType+".sql", upFile)
	if err != nil {
		exitGracefully(err)
	}

	err = copyDataToFile([]byte(
		"drop table if exists users cascade; drop table if exists tokens cascade; drop table if exists remember_tokens;",
	),
		downFile)
	if err != nil {
		exitGracefully(err)
	}

	// run migration
	err = doMigrate("up", "")
	if err != nil {
		exitGracefully(err)
	}

	err = copyFileFromTemplate("templates/data/user.go.txt", grv.RootPath+"/data/user.go")
	if err != nil {
		exitGracefully(err)
	}

	err = copyFileFromTemplate("templates/data/token.go.txt", grv.RootPath+"/data/token.go")
	if err != nil {
		exitGracefully(err)
	}

	err = copyFileFromTemplate("templates/data/remember_token.go.txt", grv.RootPath+"/data/remember_token.go")
	if err != nil {
		exitGracefully(err)
	}

	// copy over middlewares
	err = copyFileFromTemplate("templates/middleware/auth.go.txt", grv.RootPath+"/middleware/auth.go")
	if err != nil {
		exitGracefully(err)
	}

	err = copyFileFromTemplate("templates/middleware/auth-token.go.txt", grv.RootPath+"/middleware/auth-token.go")
	if err != nil {
		exitGracefully(err)
	}

	err = copyFileFromTemplate("templates/middleware/remember.go.txt", grv.RootPath+"/middleware/remember.go")
	if err != nil {
		exitGracefully(err)
	}

	// copy over handlers
	err = copyFileFromTemplate("templates/handlers/auth-handlers.go.txt", grv.RootPath+"/handlers/auth-handlers.go")
	if err != nil {
		exitGracefully(err)
	}

	// copy over views
	err = copyFileFromTemplate("templates/mailer/password-reset.html.tmpl", grv.RootPath+"/mail/password-reset.html.tmpl")
	if err != nil {
		exitGracefully(err)
	}

	err = copyFileFromTemplate("templates/mailer/password-reset.plain.tmpl", grv.RootPath+"/mail/password-reset.plain.tmpl")
	if err != nil {
		exitGracefully(err)
	}

	err = copyFileFromTemplate("templates/views/login.jet", grv.RootPath+"/views/login.jet")
	if err != nil {
		exitGracefully(err)
	}

	err = copyFileFromTemplate("templates/views/forgot.jet", grv.RootPath+"/views/forgot.jet")
	if err != nil {
		exitGracefully(err)
	}

	err = copyFileFromTemplate("templates/views/reset-password.jet", grv.RootPath+"/views/reset-password.jet")
	if err != nil {
		exitGracefully(err)
	}

	color.Yellow("  -  users, tokens, and remember_tokens migrations created and executed")
	color.Yellow("  -  users and tokens models created")
	color.Yellow("  -  auth middleware created")
	color.Yellow("")
	color.Yellow("Don't forget to add user and token models in data/models.go, and to add appropriate middleware to your routes!")

	return nil
}
