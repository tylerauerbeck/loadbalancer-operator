/*
Copyright © 2023 The Infratographer Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package cmd is the root of our application
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"

	"github.com/spf13/viper"
)

var (
	cfgFile string
	logger  *zap.SugaredLogger
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "loadbalanceroperator",
	Short: "A controller for processing requests for loadbalancers from a specified queue.",
	Long:  `A controller for processing requests for loadbalancers from a specified queue.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.loadbalanceroperator.yaml)")

	rootCmd.PersistentFlags().Bool("debug", false, "enable debug logging")
	viperBindFlag("logging.debug", rootCmd.PersistentFlags().Lookup("debug"))

	rootCmd.PersistentFlags().Bool("pretty", false, "enable pretty (human readable) logging output")
	viperBindFlag("logging.pretty", rootCmd.PersistentFlags().Lookup("pretty"))

	rootCmd.PersistentFlags().String("nats-url", "", "NATS server connection url")
	viperBindFlag("nats.url", rootCmd.PersistentFlags().Lookup("nats-url"))

	rootCmd.PersistentFlags().String("nats-creds-file", "", "Path to the file containing the NATS nkey keypair")
	viperBindFlag("nats.creds-file", rootCmd.PersistentFlags().Lookup("nats-creds-file"))

	rootCmd.PersistentFlags().String("nats-subject-prefix", "", "prefix for NATS subjects")
	viperBindFlag("nats.subject-prefix", rootCmd.PersistentFlags().Lookup("nats-subject-prefix"))

	rootCmd.PersistentFlags().StringSlice("nats-subjects", nil, "NATS subjects to subscribe to")
	viperBindFlag("nats.subjects", rootCmd.PersistentFlags().Lookup("nats-subjects"))

	rootCmd.PersistentFlags().String("nats-stream-name", "loadbalanceroperator", "prefix for NATS subjects")
	viperBindFlag("nats.stream-name", rootCmd.PersistentFlags().Lookup("nats-stream-name"))

	rootCmd.PersistentFlags().String("healthcheck-port", ":8080", "port to run healthcheck probe on")
	viperBindFlag("healthcheck-port", rootCmd.PersistentFlags().Lookup("healthcheck-port"))

	rootCmd.PersistentFlags().String("chart-path", "", "path that contains deployment chart")
	viperBindFlag("chart-path", rootCmd.PersistentFlags().Lookup("chart-path"))

	rootCmd.PersistentFlags().String("chart-values-path", "", "path that contains values file to configure deployment chart")
	viperBindFlag("chart-values-path", rootCmd.PersistentFlags().Lookup("chart-values-path"))

	rootCmd.PersistentFlags().String("kube-config-path", "", "path to a valid kubeconfig file")
	viperBindFlag("kube-config-path", rootCmd.PersistentFlags().Lookup("kube-config-path"))

	rootCmd.PersistentFlags().StringSlice("helm-cpu-flag", nil, "flag to set cpu limit for helm chart")
	viperBindFlag("helm-cpu-flag", rootCmd.PersistentFlags().Lookup("helm-cpu-flag"))

	rootCmd.PersistentFlags().StringSlice("helm-memory-flag", nil, "flag to set memory limit for helm chart")
	viperBindFlag("helm-memory-flag", rootCmd.PersistentFlags().Lookup("helm-memory-flag"))

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.AddCommand(processCmd)

	if viper.GetBool("logging.debug") {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".loadbalanceroperator" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".loadbalanceroperator")
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.SetEnvPrefix("loadbalanceroperator")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}

	setupLogging()
}

func setupLogging() {
	cfg := zap.NewProductionConfig()
	if viper.GetBool("logging.pretty") {
		cfg = zap.NewDevelopmentConfig()
	}

	if viper.GetBool("logging.debug") {
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	} else {
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	l, err := cfg.Build()
	if err != nil {
		panic(err)
	}

	logger = l.Sugar().With("app", "loadbalanceroperator")
	defer logger.Sync() //nolint:errcheck
}

// viperBindFlag provides a wrapper around the viper bindings that handles error checks
func viperBindFlag(name string, flag *pflag.Flag) {
	err := viper.BindPFlag(name, flag)
	if err != nil {
		panic(err)
	}
}
