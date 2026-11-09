package main

import (
	"os"

	router "atonixcorp/cloud/devtools/cli"
)

func main() {
	router.Dispatch(os.Args)
}
