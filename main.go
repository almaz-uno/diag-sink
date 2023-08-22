/*
Copyright © 2023 Maxim Kovrov
*/
package main

import (
	"context"

	"github.com/almaz-uno/diag-sink/cmd"
	"github.com/almaz-uno/diag-sink/pkg/rtflow"
)

func main() {
	rtflow.Main(func(ctx context.Context, cancel context.CancelFunc) error {
		defer cancel()
		return cmd.ExecuteContext(ctx)
	})
}
