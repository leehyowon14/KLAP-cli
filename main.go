package main

import (
	klasuser "KLAP/user"
	"fmt"
)

func main() {
	fmt.Println("started")
	user := klasuser.NewUser()
	user.SetID("2025404045")
	user.SetPassword("Wwkk1010$$")
	user.Login()
	user.GetCookies()
	user.LoadSemesters()
	fmt.Println(user.GetSemesters())
}
