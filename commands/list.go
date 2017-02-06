package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wallix/awless/cloud"
	"github.com/wallix/awless/cloud/aws"
	"github.com/wallix/awless/display"
	"github.com/wallix/awless/graph"
	"github.com/wallix/awless/shell"
	"github.com/wallix/awless/sync"
)

var (
	listingFormat string

	listOnlyIDs    bool
	listForService string
	localResources bool
	sortBy         []string
)

func init() {
	RootCmd.AddCommand(listCmd)

	for _, srvName := range aws.ServiceNames {
		listCmd.AddCommand(listAllResourceInServiceCmd(srvName))
	}

	for apiName, types := range aws.ResourceTypesPerAPI {
		for _, resType := range types {
			listCmd.AddCommand(listSpecificResourceCmd(apiName, resType))
		}
	}

	listCmd.PersistentFlags().StringVar(&listingFormat, "format", "table", "Format for the display of resources: table or csv")
	listCmd.PersistentFlags().BoolVar(&listOnlyIDs, "ids", false, "List only ids")
	listCmd.PersistentFlags().StringSliceVar(&sortBy, "sort", []string{"Id"}, "Sort tables by column(s) name(s)")
}

var listCmd = &cobra.Command{
	Use:                "list",
	PersistentPreRun:   applyHooks(initAwlessEnvHook, initCloudServicesHook, checkStatsHook),
	PersistentPostRunE: saveHistoryHook,
	Short:              "List various type of items: instances, vpc, subnet ...",
}

var listSpecificResourceCmd = func(apiName string, resType string) *cobra.Command {
	return &cobra.Command{
		Use:   cloud.PluralizeResource(resType),
		Short: fmt.Sprintf("List AWS %s %s", apiName, cloud.PluralizeResource(resType)),

		Run: func(cmd *cobra.Command, args []string) {
			var g *graph.Graph

			srv, err := cloud.GetServiceForType(resType)
			exitOn(err)

			if localResources {
				g = sync.LoadCurrentLocalGraph(srv.Name())
			} else {
				g, err = srv.FetchByType(resType)
			}
			exitOn(err)

			printResources(g, graph.ResourceType(resType))
		},
	}
}

var listAllResourceInServiceCmd = func(srvName string) *cobra.Command {
	return &cobra.Command{
		Use:   srvName,
		Short: fmt.Sprintf("List all %s resources", srvName),

		Run: func(cmd *cobra.Command, args []string) {
			g := sync.LoadCurrentLocalGraph(srvName)
			displayer := display.BuildOptions(
				display.WithFormat(listingFormat),
				display.WithIDsOnly(listOnlyIDs),
			).SetSource(g).Build()
			exitOn(displayer.Print(os.Stdout))
		},
	}
}

func printResources(g *graph.Graph, resType graph.ResourceType) {
	displayer := display.BuildOptions(
		display.WithRdfType(resType),
		display.WithHeaders(display.DefaultsColumnDefinitions[resType]),
		display.WithMaxWidth(shell.GetTerminalWidth()),
		display.WithFormat(listingFormat),
		display.WithIDsOnly(listOnlyIDs),
		display.WithSortBy(sortBy...),
	).SetSource(g).Build()

	exitOn(displayer.Print(os.Stdout))
}