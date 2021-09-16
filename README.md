## Disclaimer

This code was forked from the original one from Hashicorp to remove dependencies that we believe were not useful.
Since we heavily develop kubernetes operators, `go-setter` dependencies that are heavily updated caused problem to us.

## Usage

The `Terraform` struct must be initialised with `NewTerraform(workingDir, execPath)`. 

Top-level Terraform commands each have their own function, which will return either `error` or `(T, error)`, where `T` is a `terraform-json` type.


### Example


```go
package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"log"

	"github.com/covergo/terraform-exec/tfexec"
	"github.com/covergo/terraform-exec/tfinstall"
)

func main() {
	tmpDir, err := ioutil.TempDir("", "tfinstall")
	if err != nil {
		log.Fatalf("error creating temp dir: %s", err)
	}
	defer os.RemoveAll(tmpDir)

	execPath, err := tfinstall.Find(context.Background(), tfinstall.LatestVersion(tmpDir, false))
	if err != nil {
		log.Fatalf("error locating Terraform binary: %s", err)
	}

	workingDir := "/path/to/working/dir"
	tf, err := tfexec.NewTerraform(workingDir, execPath)
	if err != nil {
		log.Fatalf("error running NewTerraform: %s", err)
	}

	err = tf.Init(context.Background(), tfexec.Upgrade(true))
	if err != nil {
		log.Fatalf("error running Init: %s", err)
	}

	state, err := tf.Show(context.Background())
	if err != nil {
		log.Fatalf("error running Show: %s", err)
	}

	fmt.Println(state.FormatVersion) // "0.1"
}
```

## Contributing

Please see [CONTRIBUTING.md](./CONTRIBUTING.md).
