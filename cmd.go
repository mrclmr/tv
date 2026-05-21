package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"

	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	var list bool

	rootCmd := &cobra.Command{
		Use:          "tv [channel]",
		Short:        "Open a live TV channel in VLC",
		SilenceUsage: true,
		Args: cobra.MaximumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			channels, err := livestreamURLs(cmd.Context())
			if err != nil {
				return nil, cobra.ShellCompDirectiveError | cobra.ShellCompDirectiveNoFileComp
			}
			names := make([]string, 0, len(channels)+len(aliases))
			for _, ch := range channels {
				names = append(names, ch.title)
			}
			for alias, target := range aliases {
				if _, found := channelToURL(target, channels); found {
					names = append(names, alias)
				}
			}
			return names, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			channels, err := livestreamURLs(cmd.Context())
			if err != nil {
				return err
			}
			if list || len(args) == 0 {
				lines := make([]string, 0, len(channels)+len(aliases))
				for _, ch := range channels {
					lines = append(lines, ch.title)
				}
				for alias, target := range aliases {
					if _, found := channelToURL(target, channels); found {
						lines = append(lines, alias+" -> "+target)
					}
				}
				slices.Sort(lines)
				for _, line := range lines {
					fmt.Println(line)
				}
				return nil
			}
			u, found := channelToURL(resolveAlias(args[0], channels), channels)
			if !found {
				return errors.New("channel " + args[0] + " not found")
			}
			c := exec.CommandContext(cmd.Context(), "vlc", u)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		},
	}
	rootCmd.Flags().BoolVarP(&list, "list", "l", false, "List all available channels")
	return rootCmd
}
