//go:build !release

package internal

import "fmt"

func GetVersion() string {
	return fmt.Sprintf("%s-debug", Version)
}
