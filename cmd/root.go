package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"github.com/tomas-mazak/goslb/goslb"
	"log"
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

		// start the server
		goslb.StartServer(config)
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
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "Path to config file")
	rootCmd.PersistentFlags().StringP("log-level", "l", "INFO", "log verbosity level")
	rootCmd.PersistentFlags().String("bind-addr-api", fmt.Sprintf("0.0.0.0:%d", DefaultBindPortAPI),
		"API bind address")
	rootCmd.PersistentFlags().String("bind-addr-dns", fmt.Sprintf("0.0.0.0:%d", DefaultBindPortDNS),
		"DNS server bind address")
	rootCmd.PersistentFlags().StringSlice("etcd-servers", []string{}, "List of etcd servers to connect to")
	rootCmd.PersistentFlags().StringP("domain", "d", "goslb.", "DNS domain of the GSLB records")
	viper.BindPFlag("log_level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("bind_addr_api", rootCmd.PersistentFlags().Lookup("bind-addr-api"))
	viper.BindPFlag("bind_addr_dns", rootCmd.PersistentFlags().Lookup("bind-addr-dns"))
	viper.BindPFlag("etcd_servers", rootCmd.PersistentFlags().Lookup("etcd-servers"))
	viper.BindPFlag("domain", rootCmd.PersistentFlags().Lookup("domain"))
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
		log.Println("Can't read config, using defaults:", err)
	}
}

func readConfig() *goslb.Config {
	return &goslb.Config{
		LogLevel: viper.GetString("log_level"),
		BindAddrAPI: viper.GetString("bind_addr_api"),
		BindAddrDNS: viper.GetString("bind_addr_dns"),
		EtcdServers: viper.GetStringSlice("etcd_servers"),
		Domain: viper.GetString("domain"),
		SiteMap: viper.GetStringMapStringSlice("site_map"),
	}
}
