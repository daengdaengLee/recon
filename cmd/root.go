package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "recon",
		Short: "recon CLI",
		// 모든 에러(런타임 에러, 잘못된 인자/플래그 에러)에서 usage 출력을 생략한다.
		// 에러 메시지 자체는 SilenceErrors가 false이므로 cobra가 stderr에 그대로 출력한다.
		// usage 출력이 필요한 경우 RunE/Run 구현에서 직접 cmd.Usage()로 출력한다.
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "Hello World")
			return err
		},
	}

	return rootCmd
}
