package cmd

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"go.infratographer.com/loadbalanceroperator/internal/utils"
)

func TestLoadHelmChart(t *testing.T) {
	type testCase struct {
		name        string
		createChart bool
		expectError bool
	}

	testDir, err := os.MkdirTemp("", "load-helm-chart")
	if err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(testDir)

	testCases := []testCase{
		{
			name:        "valid chart",
			expectError: false,
			createChart: true,
		},
		{
			name:        "missing chart",
			expectError: true,
			createChart: false,
		},
	}

	for _, tcase := range testCases {
		t.Run(tcase.name, func(t *testing.T) {
			cpath := ""

			if tcase.createChart {
				cpath, err = utils.CreateTestChart(testDir)

				if err != nil {
					t.Fatal(err)
				}
			}

			ch, err := loadHelmChart(cpath)

			if tcase.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, ch)
			}
		})
	}
}

func TestValidateFlags(t *testing.T) {
	type flagSet struct {
		key   string
		value string
	}

	type testCase struct {
		name        string
		flagSet     []flagSet
		errors      error
		expectError bool
	}

	testCases := []testCase{
		{
			name:        "valid flags",
			flagSet:     []flagSet{{"chart-path", "chart"}, {"nats.subject-prefix", "stream"}},
			errors:      nil,
			expectError: false,
		},
		{
			name:        "missing chart-path",
			flagSet:     []flagSet{{"nats.subject-prefix", "stream"}},
			errors:      ErrChartPath,
			expectError: true,
		},
		{
			name:        "missing nats.subject-prefix",
			flagSet:     []flagSet{{"chart-path", "chart"}},
			errors:      ErrNATSSubjectPrefix,
			expectError: true,
		},
	}

	for _, tcase := range testCases {
		viper.Reset()
		t.Run(tcase.name, func(t *testing.T) {
			for _, f := range tcase.flagSet {
				viper.Set(f.key, f.value)
			}

			err := validateFlags()

			if tcase.expectError {
				assert.Error(t, err)
				assert.ErrorContains(t, err, tcase.errors.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
