//go:build !go1.14
// +build !go1.14

package testutil

import (
	"runtime"
	"testing"

	"github.com/covergo/terraform-wrapper/tfinstall"
)

func (tf *TFCache) find(t *testing.T, key string, finder func(dir string) tfinstall.ExecPathFinder) string {
	panic("not implemented for " + runtime.Version())
}
