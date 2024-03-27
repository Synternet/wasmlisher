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

		publisherOptions := []dlsdkOptions.Option{
			dlsdk.WithContext(ctx),
			dlsdk.WithName(*flagName),
			dlsdk.WithPrefix(*flagPrefixName),
			dlsdk.WithSubNats(natsSubConnection),
			dlsdk.WithPubNats(natsPubConnection),
			dlsdk.WithVerbose(false),
		}

		wasmlisherService := wasmlisher.New(publisherOptions, *flagConfig, *flagCfInterval)

		if wasmlisherService == nil {
			return
		}

		pubCtx := wasmlisherService.Start()
		defer wasmlisherService.Close()

		select {
		case <-ctx.Done():
			log.Println("Shutdown")
			_ = wasmlisherService.Close()
		case <-pubCtx.Done():
			log.Println("Publisher stopped with cause: ", context.Cause(pubCtx).Error())
			_ = wasmlisherService.Close()
		}
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
