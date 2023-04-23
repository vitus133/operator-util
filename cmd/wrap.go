/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// wrapCmd represents the wrap command
var wrapCmd = &cobra.Command{
	Use:   "wrap",
	Short: "Wrap manifests in ACM policies",
	Long: `Wraps manifests in a policies for application through 
	Advanced Cluster Management`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("wrap called")
	},
}

func init() {
	rootCmd.AddCommand(wrapCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// wrapCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// wrapCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
