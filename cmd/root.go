package cmd

import (
	"fmt"
	"github.com/twatzl/webdav-downloader/downloader"
	"log"
	"os"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var server string

var rootCmd = &cobra.Command{
	Use:   "webdav-downloader",
	Short: "webdav-downloader is a small utility for downloading directories from webdav servers (i.e. Nextcloud)",
	Long:  `TODO`,
	Run: func(cmd *cobra.Command, args []string) {

		remoteDir := viper.GetString("remoteDir")

		cfg := downloader.Config{
			Protocol: viper.GetString("protocol"),
			Host:     viper.GetString("host"),
			BaseDir:  viper.GetString("baseDir"),
			LocalDir: viper.GetString("localDir"),
			User:     viper.GetString("user"),
			Pass:     viper.GetString("pass"),
		}

		if cfg.Host == "" {
			log.Fatal("host is required")
		}

		if cfg.Protocol != "http" && cfg.Protocol != "https" {
			log.Fatal("protocol must be http or https")
		}

		downloader.DownloadDir(&cfg, remoteDir)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	const PROTOCOL = "protocol"
	const HOST = "host"
	const BASE_DIR = "baseDir"
	const REMOTE_DIR = "remoteDir"
	const LOCAL_DIR = "localDir"

	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.webdav-downloader.yaml)")
	rootCmd.PersistentFlags().StringP(PROTOCOL, "", "http", "protocol to use")
	rootCmd.PersistentFlags().StringP(HOST, "", "", "webdav host")
	rootCmd.PersistentFlags().StringP(BASE_DIR, "", "", "base dir (e.g. /remote.php/webdav)")
	rootCmd.PersistentFlags().StringP(REMOTE_DIR, "", "", "path which will be appended to the remote path (e.g. /some/dir)")
	rootCmd.PersistentFlags().StringP(LOCAL_DIR, "", "", "path which will be used on the local machine (e.g. /some/other/dir)")

	_ = viper.BindPFlag(PROTOCOL, rootCmd.PersistentFlags().Lookup(PROTOCOL))
	_ = viper.BindPFlag(HOST, rootCmd.PersistentFlags().Lookup(HOST))
	_ = viper.BindPFlag(BASE_DIR, rootCmd.PersistentFlags().Lookup(BASE_DIR))
	_ = viper.BindPFlag(REMOTE_DIR, rootCmd.PersistentFlags().Lookup(REMOTE_DIR))
	_ = viper.BindPFlag(LOCAL_DIR, rootCmd.PersistentFlags().Lookup(LOCAL_DIR))

}

func initConfig() {
	// Don't forget to read config either from cfgFile or from home directory!
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".webdav-downloader" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".webdav-downloader")
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Can't read config:", err)
		//os.Exit(1)
	}
}
