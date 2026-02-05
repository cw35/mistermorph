package configutil

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func FlagOrViperString(cmd *cobra.Command, flagName, viperKey string) string {
	v, _ := cmd.Flags().GetString(flagName)
	if cmd.Flags().Changed(flagName) {
		return v
	}
	if viperKey != "" && viper.IsSet(viperKey) {
		return viper.GetString(viperKey)
	}
	return v
}

func FlagOrViperStringArray(cmd *cobra.Command, flagName, viperKey string) []string {
	v, _ := cmd.Flags().GetStringArray(flagName)
	if cmd.Flags().Changed(flagName) {
		return v
	}
	if viperKey != "" && viper.IsSet(viperKey) {
		return viper.GetStringSlice(viperKey)
	}
	return v
}

func FlagOrViperBool(cmd *cobra.Command, flagName, viperKey string) bool {
	v, _ := cmd.Flags().GetBool(flagName)
	if cmd.Flags().Changed(flagName) {
		return v
	}
	if viperKey != "" && viper.IsSet(viperKey) {
		return viper.GetBool(viperKey)
	}
	return v
}

func FlagOrViperInt(cmd *cobra.Command, flagName, viperKey string) int {
	v, _ := cmd.Flags().GetInt(flagName)
	if cmd.Flags().Changed(flagName) {
		return v
	}
	if viperKey != "" && viper.IsSet(viperKey) {
		return viper.GetInt(viperKey)
	}
	return v
}

func FlagOrViperInt64(cmd *cobra.Command, flagName, viperKey string) int64 {
	v, _ := cmd.Flags().GetInt64(flagName)
	if cmd.Flags().Changed(flagName) {
		return v
	}
	if viperKey != "" && viper.IsSet(viperKey) {
		return viper.GetInt64(viperKey)
	}
	return v
}

func FlagOrViperDuration(cmd *cobra.Command, flagName, viperKey string) time.Duration {
	v, _ := cmd.Flags().GetDuration(flagName)
	if cmd.Flags().Changed(flagName) {
		return v
	}
	if viperKey != "" && viper.IsSet(viperKey) {
		return viper.GetDuration(viperKey)
	}
	return v
}

func FlagOrViperFloat64(cmd *cobra.Command, flagName, viperKey string) float64 {
	v, _ := cmd.Flags().GetFloat64(flagName)
	if cmd.Flags().Changed(flagName) {
		return v
	}
	if viperKey != "" && viper.IsSet(viperKey) {
		return viper.GetFloat64(viperKey)
	}
	return v
}
