package cmd

import (
	"context"

	"github.com/spf13/cobra"
)

var RootCommand = &cobra.Command{
	Use: "compiler",
}

func Execute(ctx context.Context) error {
	return RootCommand.ExecuteContext(ctx)
}
