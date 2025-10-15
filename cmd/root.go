package cmd

import (
	platformmeshcontext "github.com/platform-mesh/golang-commons/config"
	"github.com/platform-mesh/golang-commons/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/platform-mesh/iam-service/internal/config"
)

var (
	serviceCfg = &config.ServiceConfig{}
	defaultCfg *platformmeshcontext.CommonServiceConfig
	v          *viper.Viper
	log        *logger.Logger
)

var rootCmd = &cobra.Command{
	Use:   "iam-service",
	Short: "the platform mesh iam-service",
}

func init() {
	rootCmd.AddCommand(serverCmd)

	var err error
	v, defaultCfg, err = platformmeshcontext.NewDefaultConfig(rootCmd)
	if err != nil {
		panic(err)
	}
	err = platformmeshcontext.BindConfigToFlags(v, serverCmd, serviceCfg)
	if err != nil {
		panic(err)
	}

	cobra.OnInitialize(initLog)
}

func initLog() {
	lCfg := logger.DefaultConfig()
	lCfg.Level = defaultCfg.Log.Level
	lCfg.NoJSON = defaultCfg.Log.NoJson

	var err error
	log, err = logger.New(lCfg)
	if err != nil {
		panic(err)
	}
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}
