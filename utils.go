package main

import (
    "os"
)

func FatalErrorCheck(err error){
    if err != nil {
        Error.Println(err)
        os.Exit(1)
    }
}

func ErrorCheck(err error) bool {
    if err != nil {
        Error.Println(err)
        return true
    }
    return false
}
