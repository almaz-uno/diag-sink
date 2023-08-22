/*
Copyright © 2023 Maxim Kovrov
*/
package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "diag-sink",
	Short: "Starting to listen messages",
	Long:  `Starting to listen messages`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		fi, _ := os.Stderr.Stat()
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
			NoColor:    fi == nil || fi.Mode()&os.ModeNamedPipe == 0,
		})

		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		if l, e := zerolog.ParseLevel(viper.GetString("level")); e == nil {
			zerolog.SetGlobalLevel(l)
		}

		return nil
	},

	RunE: func(cmd *cobra.Command, args []string) error {
		listen := viper.GetString("listen")
		echoServer := echo.New()
		echoServer.Pre(middleware.RemoveTrailingSlash())
		echoServer.Use(middleware.CORS())
		echoServer.Use(middleware.Recover())

		go func() {
			<-cmd.Context().Done()

			closeCtx, closeCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer closeCancel()

			log.Info().Err(echoServer.Shutdown(closeCtx)).Msg("Server stopped")
		}()

		tf := viper.GetString("out")
		log.Info().Str("output", tf).Msg("Saving messages to file")
		echoServer.POST("/sink", createSink(tf))

		log.Info().Str("listen", listen).Msg("Start listening")
		if e := echoServer.Start(listen); errors.Is(e, http.ErrServerClosed) || e == nil {
			return nil
		} else {
			return e
		}
	},
}

func createSink(outFile string) echo.HandlerFunc {
	return func(c echo.Context) error {
		var w io.Writer
		if outFile == "-" {
			w = os.Stdout
		} else {

			f, e := os.OpenFile(outFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
			if e != nil {
				return e
			}
			defer f.Close()
			w = f
		}

		_, err := io.Copy(w, c.Request().Body)
		fmt.Fprintln(w)
		return err
	}
}

var ExecuteContext = rootCmd.ExecuteContext

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "f", "", "config file (default is $HOME/.diag-sink.yaml)")
	rootCmd.PersistentFlags().StringP("listen", "l", "localhost:2288", "[host]:<port> to listen for; it supports HTTP only!")
	rootCmd.PersistentFlags().StringP("level", "L", "info", "logging level")
	rootCmd.PersistentFlags().StringP("out", "o", "-", "output file; it will be use stdout if equals -")

	mustRegisterFlags(rootCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".diag-sink" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".diag-sink")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func mustRegisterFlags(cc *cobra.Command) {
	if err := viper.BindPFlags(cc.PersistentFlags()); err != nil {
		panic("unable to bind flags " + err.Error())
	}
}
