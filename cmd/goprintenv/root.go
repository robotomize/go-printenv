package main

import (
	"github.com/spf13/cobra"
	"gituhb.com/robotomize/go-printenv/internal/analysis"
	"gituhb.com/robotomize/go-printenv/internal/printer"
)

var (
	verboseFlag   bool
	goProjectPath string
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(
		&verboseFlag,
		"verbose",
		"v",
		false,
		"more verbose",
	)
	rootCmd.PersistentFlags().StringVarP(
		&goProjectPath,
		"path",
		"p",
		".",
		"go project path",
	)
}

var rootCmd = &cobra.Command{
	Use:  "goprintenv [-p project path -v verbose]",
	Long: "Print env variables used in go project",
	RunE: func(cmd *cobra.Command, args []string) error {

		ctx := cmd.Context()
		var printOpts []printer.Option

		a := analysis.New(
			goProjectPath, func(items ...analysis.OutputEntry) analysis.Printer {
				return printer.New(items, printOpts...)
			},
		)

		if _, err := cmd.OutOrStdout().Write(a.Print()); err != nil {
			return err
		}

		_ = ctx

		return nil
	},
}
