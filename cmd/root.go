package cmd

import (
	"github.com/nats-io/nats.go"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/syntropynet/data-layer-sdk/pkg/options"
)

var (
	flagSubNatsUrls   *string
	flagPubNatsUrls   *string
	flagSubUserCreds  *string
	flagPubUserCreds  *string
	flagSubNkey       *string
	flagPubNkey       *string
	flagSubJWT        *string
	flagPubJWT        *string
	flagTLSClientCert *string
	flagTLSKey        *string
	flagCACert        *string
	flagPrefixName    *string
	flagName          *string
	flagConfig        *string
	flagCfInterval    *int

	natsSubConnection *nats.Conn
	natsPubConnection *nats.Conn
)

var rootCmd = &cobra.Command{
	Use:   "wasm-publisher",
	Short: "",
	Long:  ``,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		log.SetFlags(0)
		var err error
		natsSubConnection, err = options.MakeNats("Wasm Publisher", *flagSubNatsUrls, *flagSubUserCreds, *flagSubNkey, *flagSubJWT, *flagCACert, *flagTLSClientCert, *flagTLSKey)
		natsPubConnection, err = options.MakeNats("Wasm Subscriber", *flagPubNatsUrls, *flagPubUserCreds, *flagPubNkey, *flagPubJWT, *flagCACert, *flagTLSClientCert, *flagTLSKey)
		if err != nil {
			panic(err)
		}
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if natsSubConnection == nil || natsPubConnection == nil {
			return
		}
		natsSubConnection.Close()
		natsPubConnection.Close()
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {

	flagSubNatsUrls = rootCmd.PersistentFlags().StringP("nats-sub-url", "n", os.Getenv("NATS_SUB_URL"), "NATS server URLs (separated by comma)")
	flagPubNatsUrls = rootCmd.PersistentFlags().StringP("nats-pub-url", "N", os.Getenv("NATS_PUB_URL"), "NATS server URLs (separated by comma)")
	flagSubUserCreds = rootCmd.PersistentFlags().StringP("nats-sub-creds", "c", os.Getenv("NATS_SUB_CREDS"), "NATS User Credentials File (combined JWT and NKey file) ")
	flagPubUserCreds = rootCmd.PersistentFlags().StringP("nats-pub-creds", "C", os.Getenv("NATS_PUB_CREDS"), "NATS User Credentials File (combined JWT and NKey file) ")
	flagSubJWT = rootCmd.PersistentFlags().StringP("nats-sub-jwt", "w", os.Getenv("NATS_SUB_JWT"), "NATS JWT")
	flagPubJWT = rootCmd.PersistentFlags().StringP("nats-pub-jwt", "W", os.Getenv("NATS_PUB_JWT"), "NATS JWT")
	flagSubNkey = rootCmd.PersistentFlags().StringP("nats-sub-nkey", "k", os.Getenv("NATS_SUB_NKEY"), "NATS NKey")
	flagPubNkey = rootCmd.PersistentFlags().StringP("nats-pub-nkey", "K", os.Getenv("NATS_PUB_NKEY"), "NATS NKey")
	flagTLSKey = rootCmd.PersistentFlags().StringP("client-key", "", "", "NATS Private key file for client certificate")
	flagTLSClientCert = rootCmd.PersistentFlags().StringP("client-cert", "", "", "NATS TLS client certificate file")
	flagCACert = rootCmd.PersistentFlags().StringP("ca-cert", "", "", "NATS CA certificate file")
	flagPrefixName = rootCmd.PersistentFlags().StringP("prefix", "", os.Getenv("PUBLISHER_PREFIX"), "NATS topic prefix name as in {prefix}.solana")
	flagName = rootCmd.PersistentFlags().StringP("name", "", os.Getenv("PUBLISHER_NAME"), "NATS subject name as in {prefix}.{name}.>")
	flagConfig = rootCmd.PersistentFlags().StringP("config", "", os.Getenv("CONFIG_DIR"), "Wasmlisher config dir")
	flagCfInterval = rootCmd.PersistentFlags().IntP("cfInterval", "", 60, "Wasmlisher config reload interval in seconds")
}
