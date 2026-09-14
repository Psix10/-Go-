package main

import (
	"fmt"
	"os"
	"runtime"
)

func main() {
	username := os.Getenv("USERNAME")
	fmt.Println("1. Имя пользователя:", username)

	args := os.Args[1:]
	fmt.Println("2. Аргументы командной строки:", args)

	goVersion := runtime.Version()
	fmt.Println("3. Версия Go (через runtime):", goVersion)
	fmt.
}