package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"github.com/tomas-mazak/goslb/goslb"
)

const (
	DefaultBindPortAPI = 8080
	DefaultBindPortDNS = 8053
)

var (
	cfgFile string
)

var rootCmd = &cobra.Command{
	Use:   "goslb",
	Short: "Lightweight DNS load balancer",
	Long:  `Lightweight DNS load balancer (GSLB) written in go`,
	Run: func(cmd *cobra.Command, args []string) {
		// read configuration
		config := readConfig()

		// setup logging
		goslb.InitLogger(config)

		// start API
		go goslb.ApiServer(config)

		// start DNS server
		goslb.DnsServer(config)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Path to config file")
	rootCmd.PersistentFlags().StringP("log-level", "", "INFO", "Log verbosity level")
	rootCmd.PersistentFlags().StringP("bind-addr-api", "", fmt.Sprintf("0.0.0.0:%d", DefaultBindPortAPI),
		"API bind address")
	rootCmd.PersistentFlags().StringP("bind-addr-dns", "", fmt.Sprintf("0.0.0.0:%d", DefaultBindPortDNS),
		"DNS server bind address")
	viper.BindPFlag("log_level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("bind_addr_api", rootCmd.PersistentFlags().Lookup("bind-addr-api"))
	viper.BindPFlag("bind_addr_dns", rootCmd.PersistentFlags().Lookup("bind-addr-dns"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("goslb") // name of config file (without extension)
		viper.AddConfigPath("/etc/goslb")
		viper.AddConfigPath("$HOME/.goslb")
		viper.AddConfigPath("./config")
	}
	viper.SetEnvPrefix("goslb")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		goslb.Log.Println("Can't read config, using defaults:", err)
	}
}

func readConfig() *goslb.Config {
	return &goslb.Config{
		LogLevel: viper.GetString("log_level"),
		BindAddrAPI: viper.GetString("bind_addr_api"),
		BindAddrDNS: viper.GetString("bind_addr_dns"),
	}
}
