package cmd

import (
	"context"
	dlsdkOptions "github.com/syntropynet/data-layer-sdk/pkg/options"
	dlsdk "github.com/syntropynet/data-layer-sdk/pkg/service"
	wasmlisher "gitlab.com/syntropynet/amberdm/publisher/wasmlisher/internal"
	"log"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()

		name, _ := cmd.Flags().GetString("name")
		config, _ := cmd.Flags().GetString("config")

		publisherOptions := []dlsdkOptions.Option{
			dlsdk.WithContext(ctx),
			dlsdk.WithName(name),
			dlsdk.WithPrefix(*flagPrefixName),
			dlsdk.WithSubNats(natsSubConnection),
			dlsdk.WithPubNats(natsPubConnection),
			dlsdk.WithVerbose(false),
		}

		wasmlisherService := wasmlisher.New(publisherOptions, config)

		if wasmlisherService == nil {
			return
		}

		pubCtx := wasmlisherService.Start()
		defer wasmlisherService.Close()

		select {
		case <-ctx.Done():
			log.Println("Shutdown")
		case <-pubCtx.Done():
			log.Println("Publisher stopped with cause: ", context.Cause(pubCtx).Error())
		}
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	const (
		CONFIG_DIR     = "~/config.json"
		PUBLISHER_NAME = "wasmlisher"
	)

	startCmd.Flags().StringP("name", "", os.Getenv(PUBLISHER_NAME), "NATS subject name as in {prefix}.{name}.>")
	startCmd.Flags().StringP("config", "", os.Getenv(CONFIG_DIR), "Wasmlisher config dir")
}
