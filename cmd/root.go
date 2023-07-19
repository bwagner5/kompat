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
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/imdario/mergo"
	"github.com/olekukonko/tablewriter"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	OutputYAML       = "yaml"
	OutputTableShort = "short"
	OutputTableWide  = "wide"
)

var (
	version = ""
)

type GlobalOptions struct {
	Verbose    bool
	Version    bool
	Output     string
	ConfigFile string
}

type RootOptions struct {
	Attribution bool
}

var (
	globalOpts = GlobalOptions{}
	rootOpts   = RootOptions{}
	rootCmd    = &cobra.Command{
		Use:     "go-cli-template",
		Version: version,
		Run: func(cmd *cobra.Command, args []string) {
			if rootOpts.Attribution {
				fmt.Println(attribution)
				os.Exit(0)
			}
		},
	}
)

//go:generate cp -r ../ATTRIBUTION.md ./
//go:embed ATTRIBUTION.md
var attribution string

func main() {
	rootCmd.Flags().BoolVar(&rootOpts.Attribution, "attribution", false, "show attributions")
	rootCmd.PersistentFlags().BoolVar(&globalOpts.Verbose, "verbose", false, "Verbose output")
	rootCmd.PersistentFlags().BoolVar(&globalOpts.Version, "version", false, "version")
	rootCmd.PersistentFlags().StringVarP(&globalOpts.Output, "output", "o", OutputTableShort,
		fmt.Sprintf("Output mode: %v", []string{OutputTableShort, OutputTableWide, OutputYAML}))
	rootCmd.PersistentFlags().StringVarP(&globalOpts.ConfigFile, "file", "f", "", "YAML Config File")

	rootCmd.AddCommand(&cobra.Command{Use: "completion", Hidden: true})
	cobra.EnableCommandSorting = false

	lo.Must0(rootCmd.Execute())
}

func ParseConfig[T any](globalOpts GlobalOptions, opts T) (T, error) {
	if globalOpts.ConfigFile == "" {
		return opts, nil
	}
	configBytes, err := os.ReadFile(globalOpts.ConfigFile)
	if err != nil {
		return opts, err
	}
	var parsedCreateOpts T
	if err := yaml.Unmarshal(configBytes, &parsedCreateOpts); err != nil {
		return opts, err
	}
	if err := mergo.Merge(&opts, parsedCreateOpts, mergo.WithOverride); err != nil {
		return opts, err
	}
	return opts, nil
}

func PrettyEncode(data any) string {
	var buffer bytes.Buffer
	enc := json.NewEncoder(&buffer)
	enc.SetIndent("", "    ")
	if err := enc.Encode(data); err != nil {
		panic(err)
	}
	return buffer.String()
}

func PrettyTable[T any](data []T, wide bool) string {
	var headers []string
	var rows [][]string
	for _, dataRow := range data {
		var row []string
		// clear headers each time so we only keep one set
		headers = []string{}
		reflectStruct := reflect.Indirect(reflect.ValueOf(dataRow))
		for i := 0; i < reflectStruct.NumField(); i++ {
			typeField := reflectStruct.Type().Field(i)
			tag := typeField.Tag.Get("table")
			if tag == "" {
				continue
			}
			subtags := strings.Split(tag, ",")
			if len(subtags) > 1 && subtags[1] == "wide" && !wide {
				continue
			}
			headers = append(headers, subtags[0])
			row = append(row, reflect.ValueOf(dataRow).Field(i).String())
		}
		rows = append(rows, row)
	}
	out := bytes.Buffer{}
	table := tablewriter.NewWriter(&out)
	table.SetHeader(headers)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t") // pad with tabs
	table.SetNoWhiteSpace(true)
	table.AppendBulk(rows) // Add Bulk Data
	table.Render()
	return out.String()
}
