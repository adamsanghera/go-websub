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
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add [topic_url ...]",
	Short: "Makes a subscription request to the given topic url",
	Long:  "Makes a subscription request to the given topic url",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Usage()
		}
		for _, topic := range args {
			// TODO(adam) Potentially make a call to the subscription
			// client to see if the topics are already discovered or not
			if _, err := url.ParseRequestURI(topic); err != nil {
				return fmt.Errorf("'%s' is not a valid url", topic)
			}
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			fmt.Println("Sending subscription request...")
			// TODO(adam) send request to daemon
		}
	},
}

func init() {
	RootCmd.AddCommand(addCmd)

	// TODO(adam) add flags for async, etc
}
