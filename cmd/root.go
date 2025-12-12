package cmd

import (
	apisv1alpha1 "github.com/kcp-dev/sdk/apis/apis/v1alpha1"
	tenancyv1alpha1 "github.com/kcp-dev/sdk/apis/tenancy/v1alpha1"
	accountsv1alpha1 "github.com/platform-mesh/account-operator/api/v1alpha1"
	platformmeshcontext "github.com/platform-mesh/golang-commons/config"
	"github.com/platform-mesh/golang-commons/logger"
	securityv1alpha1 "github.com/platform-mesh/security-operator/api/v1alpha1"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/platform-mesh/iam-service/pkg/config"
)

var (
	scheme     = runtime.NewScheme()
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
	utilruntime.Must(accountsv1alpha1.AddToScheme(scheme))
	utilruntime.Must(tenancyv1alpha1.AddToScheme(scheme))
	utilruntime.Must(apisv1alpha1.AddToScheme(scheme))
	utilruntime.Must(securityv1alpha1.AddToScheme(scheme))
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
