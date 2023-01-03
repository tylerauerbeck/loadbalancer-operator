// Package utils contains utility functions for testing
package utils

import (
	"os"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/release"
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
