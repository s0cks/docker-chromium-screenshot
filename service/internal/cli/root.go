package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"net/http"
	"time"

	"charm.land/log/v2"
	"github.com/s0cks/docker-chromium-screenshot/service/internal/meta"
	"github.com/s0cks/docker-chromium-screenshot/service/internal/server"
)

var RootCommand *cobra.Command

func init() {
	RootCommand = &cobra.Command{
		Use: "service",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			viper.SetEnvPrefix("SERVICE")
			viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
			viper.AutomaticEnv()
			return viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := server.Config{
				Addr:     fmt.Sprintf(":%d", viper.GetInt("PORT")),
				Dist:     viper.GetString("DIST"),
				Cards:    viper.GetString("CARDS_DIR"),
				Redirect: viper.GetString("INDEX_REDIRECT"),
			}
			srv := server.New(cfg)

			httpSrv := &http.Server{
				Addr:              cfg.Addr,
				Handler:           logRequests(srv),
				ReadHeaderTimeout: 10 * time.Second,
			}

			log.Info("listening....",
				"version", meta.Version,
				"commit", meta.Commit,
				"address", cfg.Addr,
			)
			httpSrv.ListenAndServe()
			return nil
		},
	}
	RootCommand.PersistentFlags().Int("port", 8080, "The port to serve on")
	RootCommand.PersistentFlags().String("index-redirect", "", "The redirect path")
	RootCommand.PersistentFlags().String("dist", "./dist", "The dist directory for the client SPA")
	RootCommand.PersistentFlags().String("cards-dir", "./cards", "The cards directory to compile")
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
