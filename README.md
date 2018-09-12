# thevent
[![Build Status](https://img.shields.io/travis/dhui/thevent/master.svg)](https://travis-ci.org/dhui/thevent) [![Code Coverage](https://img.shields.io/codecov/c/github/dhui/thevent.svg)](https://codecov.io/gh/dhui/thevent) [![GoDoc](https://godoc.org/github.com/dhui/thevent?status.svg)](https://godoc.org/github.com/dhui/thevent) [![Go Report Card](https://goreportcard.com/badge/github.com/dhui/thevent)](https://goreportcard.com/report/github.com/dhui/thevent) [![GitHub Release](https://img.shields.io/github/release/dhui/thevent/all.svg)](https://github.com/dhui/thevent/releases)
![Min Go version](https://img.shields.io/badge/Go-1.10%2C%201.11-lightgrey.svg)

thevent is a typed hierarchical event system

# Table of Contents
<!-- TOC depthFrom:2 -->

- [Features](#features)
- [Example](#example)
- [Requirements](#requirements)
- [What's with the name?](#whats-with-the-name)
- [Pronunciation](#pronunciation)
- [Benchmarks](#benchmarks)

<!-- /TOC -->

## Features
* Typed events
* Typed event data (enforced via reflection)
    * Makes writing subscribers easier since there's no type assertion boilerplate
* Hierarchical events
    * Dispatching an event will also dispatch sub/child events.
    * Sub/child event data are also typed and contain a reference to the parent's event data
* All event handlers are context.Context aware

## Example
```go
package main

import "context"
import "github.com/dhui/thevent"

type User struct {
    ID   int
    Name string
}

func trackLogin(ctx context.Context, u User) error  { return nil }
func welcomeUser(ctx context.Context, u User) error { return nil }

func main() {
    // Create event with a single handler
    userLogin := thevent.Must(thevent.New(User{}, trackLogin))
    // Add another handler
    userLogin.AddHandlers(welcomeUser)

    user := User{ID: 1, Name: "Test User"}
    // Dispatch event
    userLogin.Dispatch(context.Background(), user)
}
```

## Requirements
* thevent relies solely on the Go standard library and has no external dependencies
* thevent needs Go 1.10 due to this [bug](https://github.com/golang/go/issues/21122) in earlier versions of Go.

## What's with the name?
thevent is short for **T**yped**H**ierachical**Event**s

## Pronunciation
the·vent

## Benchmarks
Last run on: 2018-02-19
```shell
$ go test -bench . -benchmem
goos: darwin
goarch: amd64
pkg: github.com/dhui/thevent
BenchmarkEvents/4handlers/Dispatch-8         	  300000	      3620 ns/op	     424 B/op	      12 allocs/op
BenchmarkEvents/4handlers/DispatchWithResults-8         	  500000	      3725 ns/op	     424 B/op      12 allocs/op
BenchmarkEvents/4handlers/DispatchAsync-8               	  500000	      3374 ns/op	     431 B/op      12 allocs/op
BenchmarkEvents/4handlers/DispatchAsyncWithResults-8     	  500000	      4223 ns/op	     525 B/op      13 allocs/op
BenchmarkEvents/8handlers/Dispatch-8                    	  200000	      7409 ns/op	     744 B/op      19 allocs/op
BenchmarkEvents/8handlers/DispatchWithResults-8         	  200000	      7302 ns/op	     744 B/op      19 allocs/op
BenchmarkEvents/8handlers/DispatchAsync-8               	  200000	      6557 ns/op	     747 B/op      19 allocs/op
BenchmarkEvents/8handlers/DispatchAsyncWithResults-8     	  200000	      8216 ns/op	     843 B/op      21 allocs/op
BenchmarkEvents/16handlers/Dispatch-8                   	  100000	     14516 ns/op	    1384 B/op      35 allocs/op
BenchmarkEvents/16handlers/DispatchWithResults-8        	  100000	     14574 ns/op	    1384 B/op      35 allocs/op
BenchmarkEvents/16handlers/DispatchAsync-8              	  200000	     13073 ns/op	    1386 B/op      35 allocs/op
BenchmarkEvents/16handlers/DispatchAsyncWithResults-8    	  100000	     15409 ns/op	    1484 B/op      37 allocs/op
BenchmarkEvents/32handlers/Dispatch-8                   	   50000	     28980 ns/op	    2664 B/op      67 allocs/op
BenchmarkEvents/32handlers/DispatchWithResults-8        	   50000	     28509 ns/op	    2664 B/op      67 allocs/op
BenchmarkEvents/32handlers/DispatchAsync-8              	   50000	     26661 ns/op	    2662 B/op      67 allocs/op
BenchmarkEvents/32handlers/DispatchAsyncWithResults-8    	   50000	     30775 ns/op	    2774 B/op      69 allocs/op
BenchmarkEvents/64handlers/Dispatch-8                   	   20000	     56665 ns/op	    5224 B/op     131 allocs/op
BenchmarkEvents/64handlers/DispatchWithResults-8        	   30000	     56846 ns/op	    5224 B/op     131 allocs/op
BenchmarkEvents/64handlers/DispatchAsync-8              	   30000	     51559 ns/op	    5186 B/op     131 allocs/op
BenchmarkEvents/64handlers/DispatchAsyncWithResults-8    	   20000	     63692 ns/op	    5350 B/op     132 allocs/op
BenchmarkEvents/128handlers/Dispatch-8                  	   10000	    111836 ns/op	   10344 B/op     259 allocs/op
BenchmarkEvents/128handlers/DispatchWithResults-8       	   10000	    117269 ns/op	   10344 B/op     260 allocs/op
BenchmarkEvents/128handlers/DispatchAsync-8             	   10000	    102235 ns/op	   10291 B/op     258 allocs/op
BenchmarkEvents/128handlers/DispatchAsyncWithResults-8   	   10000	    122780 ns/op	   10466 B/op     261 allocs/op
BenchmarkEvents/256handlers/Dispatch-8                  	   10000	    225310 ns/op	   20584 B/op     515 allocs/op
BenchmarkEvents/256handlers/DispatchWithResults-8       	   10000	    227159 ns/op	   20584 B/op     515 allocs/op
BenchmarkEvents/256handlers/DispatchAsync-8             	   10000	    209598 ns/op	   20568 B/op     515 allocs/op
BenchmarkEvents/256handlers/DispatchAsyncWithResults-8   	   10000	    260025 ns/op	   21073 B/op     521 allocs/op
BenchmarkEvents/512handlers/Dispatch-8                  	    3000	    475614 ns/op	   41064 B/op    1027 allocs/op
BenchmarkEvents/512handlers/DispatchWithResults-8       	    3000	    480914 ns/op	   41064 B/op    1027 allocs/op
BenchmarkEvents/512handlers/DispatchAsync-8             	    3000	    451856 ns/op	   40884 B/op    1023 allocs/op
BenchmarkEvents/512handlers/DispatchAsyncWithResults-8   	    3000	    613101 ns/op	   43851 B/op    1058 allocs/op
BenchmarkEvents/1024handlers/Dispatch-8                 	    2000	    931152 ns/op	   82024 B/op    2051 allocs/op
BenchmarkEvents/1024handlers/DispatchWithResults-8      	    2000	    967402 ns/op	   82024 B/op    2051 allocs/op
BenchmarkEvents/1024handlers/DispatchAsync-8            	    2000	    853199 ns/op	   81926 B/op    2049 allocs/op
BenchmarkEvents/1024handlers/DispatchAsyncWithResults-8  	    2000	   1161128 ns/op	   88930 B/op    2117 allocs/op
BenchmarkEvents/2048handlers/Dispatch-8                 	    1000	   1893302 ns/op	  163944 B/op    4099 allocs/op
BenchmarkEvents/2048handlers/DispatchWithResults-8      	    1000	   1923960 ns/op	  163944 B/op    4099 allocs/op
BenchmarkEvents/2048handlers/DispatchAsync-8            	    1000	   1685960 ns/op	  163731 B/op    4094 allocs/op
BenchmarkEvents/2048handlers/DispatchAsyncWithResults-8  	    1000	   2474525 ns/op	  185803 B/op    4361 allocs/op
BenchmarkEvents/4096handlers/Dispatch-8                 	     500	   3723553 ns/op	  327784 B/op    8195 allocs/op
BenchmarkEvents/4096handlers/DispatchWithResults-8      	     500	   3821736 ns/op	  327784 B/op    8195 allocs/op
BenchmarkEvents/4096handlers/DispatchAsync-8            	     500	   3364377 ns/op	  327234 B/op    8182 allocs/op
BenchmarkEvents/4096handlers/DispatchAsyncWithResults-8  	     300	   5148670 ns/op	  367133 B/op    8679 allocs/op
PASS
ok  	github.com/dhui/thevent	82.142s
```