package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	verbose bool
)

const banner = `
   ____       ____            _        _                 
  / ___| ___ / ___|___  _ __ | |_ __ _(_)_ __   ___ _ __ 
 | |  _ / _ \ |   / _ \| '_ \| __/ _' | | '_ \ / _ \ '__|
 | |_| | (_) | |__| (_) | | | | || (_| | | | | |  __/ |   
  \____|\___/ \____\___/|_| |_|\__\__,_|_|_| |_|\___|_|   

  A minimal container runtime built with Go
  Using Linux namespaces & cgroups v2
`

// rootCmd represents the base command.
var rootCmd = &cobra.Command{
	Use:   "gocontainer",
	Short: "A minimal container runtime using Linux namespaces and cgroups",
	Long:  banner,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose debug output")
}
