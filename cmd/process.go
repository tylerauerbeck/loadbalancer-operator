package cmd

import (
	"context"
	"os"
	"os/signal"

	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/nats-io/nats.go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"go.infratographer.com/x/echox"
	"go.infratographer.com/x/versionx"

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

func init() {
	rootCmd.AddCommand(processCmd)

	// only available as a CLI arg because it shouldn't be something that could accidentially end up in a config file or env var
	processCmd.Flags().BoolVar(&processDevMode, "dev", false, "dev mode: disables all auth checks, pretty logging, etc.")
}

func process(ctx context.Context, logger *zap.SugaredLogger) error {
	if err := validateFlags(); err != nil {
		return err
	}

	client, err := newKubeAuth(viper.GetString("kube-config-path"))
	if err != nil {
		logger.Fatalw("failed to create Kubernetes client", "error", err)

		return err
	}

	js, err := newJetstreamConnection()
	if err != nil {
		logger.Fatalw("failed to create NATS jetstream connection", "error", err)

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
		Echo:            eSrv,
		Chart:           chart,
		Context:         cx,
		Debug:           viper.GetBool("logging.debug"),
		JetstreamClient: js,
		KubeClient:      client,
		Logger:          logger,
		Prefix:          viper.GetString("nats.subject-prefix"),
		Subjects:        viper.GetStringSlice("nats.subjects"),
		StreamName:      viper.GetString("nats.stream-name"),
		ValuesPath:      viper.GetString("chart-values-path"),
		Locations:       viper.GetStringSlice("locations"),
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

	return nil
}

func newJetstreamConnection() (nats.JetStreamContext, error) {
	opts := []nats.Option{}

	if !processDevMode && viper.GetString("nats.creds-file") != "" {
		opts = append(opts, nats.UserCredentials(viper.GetString("nats.creds-file")))
	}

	nc, err := nats.Connect(viper.GetString("nats.url"), opts...)
	if err != nil {
		return nil, err
	}

	js, err := nc.JetStream()
	if err != nil {
		return nil, err
	}

	return js, nil
}

func newKubeAuth(path string) (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		if path != "" {
			config, err = clientcmd.BuildConfigFromFlags("", path)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return config, nil
}

func validateFlags() error {
	if viper.GetString("nats.subject-prefix") == "" {
		return ErrNATSSubjectPrefix
	}

	if viper.GetString("chart-path") == "" {
		return ErrChartPath
	}

	return nil
}

func loadHelmChart(chartPath string) (*chart.Chart, error) {
	chart, err := loader.Load(chartPath)
	if err != nil {
		// logger.Errorw("failed to load helm chart", "error", err)
		return nil, err
	}

	return chart, nil
}
