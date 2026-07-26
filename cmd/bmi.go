package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/bmi"
)

var bmiCmd = &cobra.Command{
	Use:   "bmi <height_cm> <weight_kg>",
	Short: "BMI calculator",
	Long: `Calculate Body Mass Index (BMI) from height and weight.

BMI = weight(kg) / height(m)²

Categories:
  < 18.5    Underweight
  18.5-24.9 Normal weight
  25.0-29.9 Overweight
  30.0-34.9 Obese (Class I)
  35.0-39.9 Obese (Class II)
  >= 40.0   Obese (Class III)

Examples:
  lingboxbmi 170 65
  lingboxbmi 160 80`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var height, weight float64

		if _, err := fmt.Sscanf(args[0], "%f", &height); err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error: invalid height %q\n", args[0])
			return
		}
		if _, err := fmt.Sscanf(args[1], "%f", &weight); err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error: invalid weight %q\n", args[1])
			return
		}

		result, err := bmi.Calculate(height, weight)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
			return
		}

		fmt.Printf("Height: %.1f cm\n", height)
		fmt.Printf("Weight: %.1f kg\n", weight)
		fmt.Println(result.String())
	},
}

func init() {
	rootCmd.AddCommand(bmiCmd)
}
