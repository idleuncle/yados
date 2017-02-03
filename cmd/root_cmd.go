package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ======== RootCmd *cobra.Command ========
var RootCmd = &cobra.Command{
	Use:   "yados",
	Short: "Yet Another Distributed Object Storage",
	Long:  "Yet Another Distributed Object Storage - A Fast and Flexible mass small files sever.",
	Run:   nil,
}

// ======== Execute() ========
// 主程序调用cmd.Execute()启动命令行解析执行主循环。
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

var cfgFile string

// -------- init() --------
func init() {
	cobra.OnInitialize(initConfig)

	//Persistent Flags which will work for this command and all subcommands.
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.yados.yaml)")

	// Local flags, which will only run when this action is called directly.
	RootCmd.Flags().BoolP("vmodule", "v", false, "glog vmodule. -v=1 for debug.")

	//viper.SetDefault("author", "NAME HERE <EMAIL ADDRESS>")
	//viper.SetDefault("license", "mit")
}

// -------- initConfig() --------
func initConfig() {
	viper.SetConfigType("yaml")
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName(".yados")
	viper.AddConfigPath("/etc/yados/")
	viper.AddConfigPath("$HOME/.yados")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

}

// -------- execute_rootCmd() --------
func execute_rootCmd(cmd *cobra.Command, args []string) {
	fmt.Println("Execute rootCmd.")
}
