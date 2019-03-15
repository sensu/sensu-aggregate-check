package main

import (
	//	"encoding/json"
	"fmt"
	//	"io/ioutil"
	"os"

	//	"github.com/sensu/sensu-go/types"
	"github.com/spf13/cobra"
)

var (
	labels     string
	namespaces string
	stdin      *os.File
)

func main() {
	rootCmd := configureRootCommand()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func configureRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sensu-aggregate-check",
		Short: "The Sensu Go Event Aggregates Check plugin",
		RunE:  run,
	}

	cmd.Flags().StringVarP(&labels,
		"labels",
		"l",
		"",
		"aggregate=foo,app=bar")

	cmd.Flags().StringVarP(&namespaces,
		"namespaces",
		"n",
		"default",
		"us-east-1,us-west-2")

	_ = cmd.MarkFlagRequired("labels")

	return cmd
}

func run(cmd *cobra.Command, args []string) error {
	if len(args) != 0 {
		_ = cmd.Help()
		return fmt.Errorf("invalid argument(s) received")
	}

	return evalAggregate()
}

func evalAggregate() error {
	fmt.Printf("hello world: %s\n", labels)

	return nil
}
