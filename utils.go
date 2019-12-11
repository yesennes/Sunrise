package main

import (
    "os"
    "fmt"
)

func FatalErrorCheck(err error){
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}
