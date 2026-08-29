package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "recon",
		Short: "recon CLI",
		// usage 출력을 생략한다.
		// usage 출력이 필요한 경우 RunE/Run 구현에서 직접 cmd.Usage()로 출력한다.
		SilenceUsage: true,
		// error 출력을 생략한다.
		// error 출력이 필요한 경우 Command 를 사용하는 호출자가 반환받은 error 를 직접 처리한다.
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "Hello World")
			return err
		},
	}

	return rootCmd
}
