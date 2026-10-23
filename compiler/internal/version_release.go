//go:build release

package internal

import "fmt"

func GetVersion() string {
	if Version[0] >= '0' && Version <= '9' {
		return Version
	}

	return fmt.Sprintf("%s-debug", Version)
}
