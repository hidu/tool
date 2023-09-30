// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/9/27

package internal

import (
	"flag"
	"log"
	"os"
	"strconv"
	"time"
)

func FlagEnvString(name string, envKey string, value string, usage string) *string {
	ev := os.Getenv(envKey)
	if ev != "" {
		value = ev
	}
	usage += " ( Env Key: " + envKey + " )"
	return flag.String(name, value, usage)
}

func FlagEnvInt(name string, envKey string, value int, usage string) *int {
	ev := os.Getenv(envKey)
	if ev != "" {
		nv, err := strconv.Atoi(ev)
		if err == nil {
			value = nv
		} else {
			log.Printf("parser flag %q from env.%q=%q failed: %v\n", name, envKey, ev, err)
		}
	}
	usage += " ( Env Key: " + envKey + " )"
	return flag.Int(name, value, usage)
}

func FlagEnvDuration(name string, envKey string, value time.Duration, usage string) *time.Duration {
	ev := os.Getenv(envKey)
	if ev != "" {
		nv, err := time.ParseDuration(ev)
		if err == nil {
			value = nv
		} else {
			log.Printf("parser flag %q from env.%q=%q failed: %v\n", name, envKey, ev, err)
		}
	}
	usage += " ( Env Key: " + envKey + " )"
	return flag.Duration(name, value, usage)
}

func FlagEnvBool(name string, envKey string, value bool, usage string) *bool {
	ev := os.Getenv(envKey)
	if ev != "" {
		nv, err := strconv.ParseBool(ev)
		if err == nil {
			value = nv
		} else {
			log.Printf("parser flag %q from env.%q=%q failed: %v\n", name, envKey, ev, err)
		}
	}
	usage += " ( Env Key: " + envKey + " )"
	return flag.Bool(name, value, usage)
}
