package main

import (
	"fmt"
	"os"
	"strconv"
)

var usage = fmt.Sprintf("Usage: %s shardNum totalShards args...", os.Args[0])

func main() {
	if err := do(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func do() error {
	if len(os.Args) < 3 {
		return fmt.Errorf(usage)
	}
	shardNum, err := strconv.Atoi(os.Args[1])
	if err != nil {
		return fmt.Errorf("%v\n%s", err, usage)
	}
	totalShards, err := strconv.Atoi(os.Args[2])
	if err != nil {
		return fmt.Errorf("%v\n%s", err, usage)
	}
	for i := 3; i < len(os.Args); i++ {
		if ((i - 3) % totalShards) == shardNum {
			fmt.Printf("%s ", os.Args[i])
		}
	}
	return nil
}
