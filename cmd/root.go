package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	"github.com/andy2046/gopie/pkg/log"
	"github.com/andy2046/kubeschema/pkg/validator"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	logger = log.NewLogger(func(c *log.Config) error {
		c.Level = log.INFO
		c.Prefix = "kubeschema:"
		return nil
	})

	version bool

	info        = color.New(color.FgHiGreen, color.BgBlack).SprintFunc()
	warn        = color.New(color.FgHiYellow, color.BgBlack).SprintFunc()
	red         = color.New(color.FgHiRed, color.BgBlack).SprintFunc()
	versionInfo = color.New(color.BlinkSlow, color.FgHiGreen, color.BgBlack).SprintFunc()

	// RootCmd is the command to run for kubernetes json schema validation.
	RootCmd = &cobra.Command{
		Use:   "kubeschema <file> [file...]",
		Short: "Validate kubernetes json schema for helm chart YAML file",
		Long:  `Validate kubernetes json schema for helm chart YAML file`,
		Run: func(cmd *cobra.Command, args []string) {
			if version {
				printVersion()
				os.Exit(0)
			}
			success, windowsStdinIssue := true, false
			stat, err := os.Stdin.Stat()
			if err != nil {
				if runtime.GOOS != "windows" {
					logger.Error(red(err))
					os.Exit(1)
				} else {
					windowsStdinIssue = true
				}
			}
			// check whether there is anything in stdin if no argument or the argument is `-`
			if (len(args) < 1 || args[0] == "-") && !windowsStdinIssue && ((stat.Mode() & os.ModeCharDevice) == 0) {
				var buffer bytes.Buffer
				scanner := bufio.NewScanner(os.Stdin)
				for scanner.Scan() {
					buffer.WriteString(scanner.Text() + "\n")
				}
				results, err := validator.Validate(buffer.Bytes(), viper.GetString("filename"))
				if err != nil {
					logger.Error(red(err))
					os.Exit(1)
				}
				success = logResults(results, success)
			} else {
				if len(args) < 1 {
					logger.Error(red("Missing filename in argument"))
					os.Exit(1)
				}
				for _, fileName := range args {
					filePath, _ := filepath.Abs(fileName)
					fileContents, err := ioutil.ReadFile(filePath)
					if err != nil {
						logger.Error(red("Failed to open file ", fileName))
						os.Exit(1)
					}
					results, err := validator.Validate(fileContents, fileName)
					if err != nil {
						logger.Error(red(err))
						os.Exit(1)
					}
					success = logResults(results, success)
				}
			}
			if !success {
				os.Exit(1)
			}
		},
	}
)

func printVersion() {
	versionFromEnv := viper.GetString("version") // KUBESCHEMA_VERSION
	if versionFromEnv == "" {
		versionFromEnv = "0.1.0"
	}
	fmt.Println(versionInfo("kubeschema v", versionFromEnv, " -- github.com/andy2046/kubeschema"))
}

func logResults(results []validator.ValidationResult, success bool) bool {
	for _, result := range results {
		if len(result.Errors) > 0 {
			success = false
			logger.Warn(red("The file ", result.FileName, " contains an invalid ", result.Kind))
			for _, desc := range result.Errors {
				logger.Info("-->", warn(desc))
			}
		} else if result.Kind == "" {
			logger.Info(warn("The file ", result.FileName, " is empty"))
		} else {
			logger.Info(info("The file ", result.FileName, " contains a valid ", result.Kind))
		}
	}
	return success
}

// Execute adds command flags to the root command and execute it.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		logger.Error(red(err))
		os.Exit(1)
	}
}

func init() {
	viper.SetEnvPrefix("KUBESCHEMA")
	viper.AutomaticEnv()
	RootCmd.Flags().StringVarP(&validator.Version, "kubernetes-version", "v", "master", "Kubernetes version to validate against")
	RootCmd.Flags().StringVarP(&validator.SchemaLocation, "schema-location", "", validator.DefaultSchemaLocation,
		"base URL used to download schemas. It also can be specified with the environment variable `KUBESCHEMA_SCHEMA_LOCATION`")
	viper.BindPFlag("schema_location", RootCmd.Flags().Lookup("schema-location"))
	RootCmd.PersistentFlags().StringP("filename", "f", "from stdin", "filename to be displayed for YAML file from stdin")
	viper.BindPFlag("filename", RootCmd.PersistentFlags().Lookup("filename"))
	RootCmd.Flags().BoolVarP(&version, "version", "", false, "display the kubeschema version information")
}
