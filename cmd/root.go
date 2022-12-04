package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/ilyabrin/disk"
	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

const CLI_VERSION = "v1.0.0"

var configFile string

var yadisk *disk.Client

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "yad",
	Version: CLI_VERSION,
	// Aliases: []string{"v"},
	Short: "Yandex.Disk command line tool",
	// TODO: update description text
	Long: `
    Yandex.Disk command line tool ` + CLI_VERSION + `

    Work with Yandex.Disk service from terminal:
    quickly create, delete and share files and folders.`,
}

var versionCmd = &cobra.Command{
	Use:     "version",
	Version: CLI_VERSION,
	Aliases: []string{"v", "ver"},
	Run: func(cmd *cobra.Command, args []string) {
		println(CLI_VERSION)
	},
}

type Account struct {
	Name        string `mapstructure:"name"`
	AccessToken string `mapstructure:"access_token"`
}
type Config struct {
	Accounts []Account `mapstructure:"accounts"`
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func init() {
	if configFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(configFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".yad" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".yad")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Println("Using config file:", viper.ConfigFileUsed())
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return
	}

	// TODO: add config.Accounts["home"].AccessToken
	fmt.Println(config.Accounts[0].Name)

	// TODO: make it better
	token := os.Getenv(config.Accounts[0].AccessToken)
	if len(token) > 0 {
		yadisk = disk.New(token)
	} else {
		println("access_token should be declared in config file")
	}

	rootCmd.AddCommand(versionCmd)
	// cobra.OnInitialize(initConfig)
	// rootCmd.Flags().BoolP("meta", "m", false, "Help message for meta")
}

// wip: if Output("json") { ... }
// println("Output:", Output("table"))
// TODO: yad --output=format // [ json ]
func Output(value string) bool {
	return viper.GetString("output") == value
}
