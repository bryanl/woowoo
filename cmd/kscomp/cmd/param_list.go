// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"github.com/bryanl/woowoo/action"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	vParamListNamespace = "param-list-ns"
)

// listCmd represents the list command
var paramListCmd = &cobra.Command{
	Use:   "list",
	Short: "param list",
	Long:  `param list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		nsName := viper.GetString(vParamListNamespace)
		return action.ParamList(fs, nsName)
	},
}

func init() {
	paramCmd.AddCommand(paramListCmd)

	paramListCmd.Flags().String(flagNamespace, "", "Component namespace")
	viper.BindPFlag(vParamListNamespace, paramListCmd.Flags().Lookup(flagNamespace))
}
