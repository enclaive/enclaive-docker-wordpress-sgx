package main

import (
    "os"
)

func GramineSetup() {
    f, err := os.Create("/dev/attestation/user_report_data")
    if err != nil { panic(err) }
    defer f.Close();

    var userdata = [64]byte{}
    userdata[0] = byte('h')
    userdata[1] = byte('e')
    userdata[2] = byte('l')
    userdata[3] = byte('l')
    userdata[4] = byte('o')
    f.Write(userdata[:])
}
