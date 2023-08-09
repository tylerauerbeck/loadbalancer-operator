package cmd

import (
	"context"
	"errors"
	"os"
	"os/signal"

	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"go.infratographer.com/ipam-api/pkg/ipamclient"
	"go.infratographer.com/loadbalancer-manager-haproxy/pkg/lbapi"
	"go.infratographer.com/x/echox"
	"go.infratographer.com/x/oauth2x"
	"go.infratographer.com/x/versionx"
	"go.infratographer.com/x/viperx"

	"go.infratographer.com/loadbalanceroperator/internal/config"
	"go.infratographer.com/loadbalanceroperator/internal/srv"
)

// processCmd represents the base command when called without any subcommands
var processCmd = &cobra.Command{
	Use:   "process",
	Short: "Begin processing requests from queues.",
	Long:  `Begin processing requests from message queues to create LBs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return process(cmd.Context(), logger)
	},
}

var (
	processDevMode bool
)

const (
	DefaultLBMetricsPort = 29782
)

func init() {
	// only available as a CLI arg because it shouldn't be something that could accidentially end up in a config file or env var
	processCmd.Flags().BoolVar(&processDevMode, "dev", false, "dev mode: disables all auth checks, pretty logging, etc.")

	processCmd.PersistentFlags().String("api-endpoint", "http://localhost:7608", "endpoint for load balancer API. defaults to supergraph if set")
	viperx.MustBindFlag(viper.GetViper(), "api-endpoint", processCmd.PersistentFlags().Lookup("api-endpoint"))

	processCmd.PersistentFlags().String("ipam-endpoint", "http://localhost:7905", "endpoint for ipam API. defaults to supergraph if set.")
	viperx.MustBindFlag(viper.GetViper(), "ipam-endpoint", processCmd.PersistentFlags().Lookup("ipam-endpoint"))

	processCmd.PersistentFlags().String("supergraph-endpoint", "", "endpoint for supergraph gateway")
	viperx.MustBindFlag(viper.GetViper(), "supergraph-endpoint", processCmd.PersistentFlags().Lookup("supergraph-endpoint"))

	processCmd.PersistentFlags().String("chart-path", "", "path that contains deployment chart")
	viperx.MustBindFlag(viper.GetViper(), "chart-path", processCmd.PersistentFlags().Lookup("chart-path"))

	processCmd.PersistentFlags().String("chart-values-path", "", "path that contains values file to configure deployment chart")
	viperx.MustBindFlag(viper.GetViper(), "chart-values-path", processCmd.PersistentFlags().Lookup("chart-values-path"))

	processCmd.PersistentFlags().StringSlice("event-locations", nil, "location id(s) to filter events for")
	viperx.MustBindFlag(viper.GetViper(), "event-locations", processCmd.PersistentFlags().Lookup("event-locations"))

	processCmd.PersistentFlags().StringSlice("change-topics", nil, "change topics to subscribe to")
	viperx.MustBindFlag(viper.GetViper(), "change-topics", processCmd.PersistentFlags().Lookup("change-topics"))

	processCmd.PersistentFlags().StringSlice("event-topics", nil, "event topics to subscribe to")
	viperx.MustBindFlag(viper.GetViper(), "event-topics", processCmd.PersistentFlags().Lookup("event-topics"))

	processCmd.PersistentFlags().String("kube-config-path", "", "path to a valid kubeconfig file")
	viperx.MustBindFlag(viper.GetViper(), "kube-config-path", processCmd.PersistentFlags().Lookup("kube-config-path"))

	processCmd.PersistentFlags().String("helm-containerport-key", "containerPorts", "key to use for injecting port values for deployment into chart")
	viperx.MustBindFlag(viper.GetViper(), "helm-containerport-key", processCmd.PersistentFlags().Lookup("helm-containerport-key"))

	processCmd.PersistentFlags().String("helm-serviceport-key", "service.ports", "key to use for injecting port values for service into chart")
	viperx.MustBindFlag(viper.GetViper(), "helm-serviceport-key", processCmd.PersistentFlags().Lookup("helm-serviceport-key"))

	processCmd.PersistentFlags().Int("loadbalancer-metrics-port", DefaultLBMetricsPort, "port to expose deployed load balancer metrics on")
	viperx.MustBindFlag(viper.GetViper(), "loadbalancer-metrics-port", processCmd.PersistentFlags().Lookup("loadbalancer-metrics-port"))

	rootCmd.AddCommand(processCmd)
}

func process(ctx context.Context, logger *zap.SugaredLogger) error {
	if err := validateFlags(); err != nil {
		return err
	}

	client, err := newKubeAuth(viper.GetString("kube-config-path"))
	if err != nil {
		logger.Fatalw("failed to create Kubernetes client", "error", err)
		err = errors.Join(err, errInvalidKubeClient)

		return err
	}

	chart, err := loadHelmChart(viper.GetString("chart-path"))
	if err != nil {
		logger.Fatalw("failed to load helm chart from provided path", "error", err)
		return err
	}

	cx, cancel := context.WithCancel(ctx)

	eSrv, err := echox.NewServer(
		logger.Desugar(),
		echox.ConfigFromViper(viper.GetViper()),
		versionx.BuildDetails(),
	)
	if err != nil {
		logger.Fatal("failed to initialize new server", zap.Error(err))
	}

	server := &srv.Server{
		Echo:             eSrv,
		Chart:            chart,
		Context:          cx,
		Debug:            viper.GetBool("logging.debug"),
		KubeClient:       client,
		Logger:           logger,
		EventTopics:      viper.GetStringSlice("event-topics"),
		ChangeTopics:     viper.GetStringSlice("change-topics"),
		SubscriberConfig: config.AppConfig.Events.Subscriber,
		ValuesPath:       viper.GetString("chart-values-path"),
		Locations:        viper.GetStringSlice("event-locations"),
		MetricsPort:      viper.GetInt("loadbalancer-metrics-port"),

		ContainerPortKey: viper.GetString("helm-containerport-key"),
		ServicePortKey:   viper.GetString("helm-serviceport-key"),
	}

	// init lbapi client
	if config.AppConfig.OIDC.Client.Issuer != "" {
		oidcTS, err := oauth2x.NewClientCredentialsTokenSrc(ctx, config.AppConfig.OIDC.Client)
		if err != nil {
			logger.Fatalw("failed to create oauth2 token source", "error", err)
		}

		oauthHTTPClient := oauth2x.NewClient(ctx, oidcTS)
		server.APIClient = lbapi.NewClient(viper.GetString("supergraph-endpoint"), lbapi.WithHTTPClient(oauthHTTPClient))
		server.IPAMClient = ipamclient.NewClient(viper.GetString("supergraph-endpoint"), ipamclient.WithHTTPClient(oauthHTTPClient))
	} else {
		server.APIClient = lbapi.NewClient(viper.GetString("supergraph-endpoint"))
		server.IPAMClient = ipamclient.NewClient(viper.GetString("supergraph-endpoint"))
	}

	if err := server.Run(cx); err != nil {
		logger.Fatalw("failed starting server", "error", err)
		cancel()
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	recvSig := <-sigCh
	signal.Stop(sigCh)
	cancel()
	logger.Infof("exiting. Performing necessary cleanup", recvSig)

	if err := server.Shutdown(); err != nil {
		logger.Errorw("failed to shutdown server", zap.Error(err))
	}

	return nil
}

func newKubeAuth(path string) (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		if path != "" {
			config, err = clientcmd.BuildConfigFromFlags("", path)
			if err != nil {
				return nil, errors.Join(err, errInvalidKubeClient)
			}
		} else {
			return nil, errors.Join(err, errInvalidKubeClient)
		}
	}

	return config, nil
}

func validateFlags() error {
	if viper.GetString("chart-path") == "" {
		return errChartPath
	}

	if len(viper.GetStringSlice("event-topics")) < 1 {
		return errRequiredTopics
	}

	return nil
}

func loadHelmChart(chartPath string) (*chart.Chart, error) {
	chart, err := loader.Load(chartPath)
	if err != nil {
		logger.Errorw("failed to load helm chart", "error", err)

		return nil, errors.Join(err, errInvalidHelmChart)
	}

	return chart, nil
}
