/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/bwagner5/kompat/pkg/kompat"
)

type GetOptions struct {
	File string
}

var (
	getOpts = GetOptions{}
	cmdGet  = &cobra.Command{
		Use:   "get ",
		Short: "get",
		Long:  `get`,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			kompatFile, err := kompat.Parse(getOpts.File)
			if err != nil {
				fmt.Printf("Unable to parse kompat file: %v\n", err)
				os.Exit(1)
			}
			switch globalOpts.Output {
			case OutputJSON:
				fmt.Println(kompatFile.JSON())
				os.Exit(0)
			case OutputYAML:
				fmt.Println(kompatFile.YAML())
				os.Exit(0)
			case OutputMarkdown:
				fmt.Println(kompatFile.Markdown())
				os.Exit(0)
			case OutputTable:

			}
		},
	}
)

func init() {
	rootCmd.AddCommand(cmdGet)
	cmdGet.Flags().StringVarP(&getOpts.File, "file", "f", ".compatibility.yaml", "path or url to the compatibility.yaml file")
}
