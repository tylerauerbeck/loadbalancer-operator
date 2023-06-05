// Package utils contains utility functions for testing
package utils

import (
	"os"

	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"

	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/stretchr/testify/suite"

	"go.infratographer.com/x/events"
	"go.infratographer.com/x/testing/eventtools"
)

// CreateTestChart creates a dummy chart for testing purposes
func CreateTestChart(outputDir string) (string, error) {
	mockreleaseoptions := release.MockReleaseOptions{}
	mocker := release.Mock(&mockreleaseoptions)
	mchart := mocker.Chart
	mchart.Metadata.APIVersion = "v2"
	mchart.Metadata.Name = "lb-dummy"
	mchart.Templates = []*chart.File{
		{Name: "templates/test.yaml", Data: []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: lb-test\n  namespace: \"{{ .Release.Namespace }}\"\ndata:\n  test: test\n")},
	}

	return chartutil.Save(mchart, outputDir)
}

func CreateTestValues(outputDir string, yamlString string) (string, error) {
	file, err := os.Create(outputDir + "/values.yaml")
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = file.WriteString(yamlString)
	if err != nil {
		return "", err
	}

	return outputDir + "/values.yaml", err
}

type OperatorTestSuite struct {
	suite.Suite
	PubConfig  events.PublisherConfig
	SubConfig  events.SubscriberConfig
	Logger     *zap.SugaredLogger
	Kubeenv    *envtest.Environment
	Kubeconfig *rest.Config
}

func (suite *OperatorTestSuite) SetupSuite() {
	pub, sub, err := eventtools.NewNatsServer()
	if err != nil {
		panic(err)
	}

	suite.PubConfig = pub

	suite.SubConfig = sub

	env, cfg := StartKube()
	suite.Kubeenv = env
	suite.Kubeconfig = cfg
}

func (suite *OperatorTestSuite) TearDownAllSuite() {
	stop := suite.Kubeenv.Stop()

	if stop != nil {
		panic(stop)
	}
}

func CreateWorkspace(dir string) (string, string, *chart.Chart, string) {
	d, err := os.MkdirTemp("", dir)
	if err != nil {
		panic(err)
	}

	chartPath, err := CreateTestChart(d)
	if err != nil {
		panic(err)
	}

	ch, err := loader.Load(chartPath)
	if err != nil {
		panic(err)
	}

	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	return d, chartPath, ch, pwd
}
