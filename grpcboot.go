package grpcboot

import (
	"fmt"
	"mime"
	"net"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/rs/cors"
	"google.golang.org/grpc"
)

type TLS struct {
	Certificate string
	Key         string
}

type Config struct {
	AllowCORS      bool
	Listener       net.Listener
	GRPCServer     *grpc.Server
	SubDirectories []string
	RootDirectory  string
	TLS            TLS
}

func InitializeConfig(c Config) (Config, error) {
	if c.GRPCServer == nil {
		return c, fmt.Errorf("GRPCServer must be specified in serve config")
	}
	if c.Listener == nil {
		l, err := net.Listen("tcp", "5014")
		if err != nil {
			return c, err
		}
		c.Listener = l
	}
	if c.SubDirectories == nil {
		c.SubDirectories = []string{}
	}
	return c, nil
}

func SplitGRPCTraffic(fallback http.Handler, grpcHandler http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.Header.Get("Content-Type"), "application/grpc") ||
				websocket.IsWebSocketUpgrade(r) {
				grpcHandler.ServeHTTP(w, r)
				return
			}
			fallback.ServeHTTP(w, r)
		},
	)
}

func Serve(c Config) error {
	var err error
	c, err = InitializeConfig(c)
	if err != nil {
		return err
	}

	if c.AllowCORS {
		err := mime.AddExtensionType(".js", "application/javascript")
		if err != nil {
			return err
		}
	}

	wrapped := grpcweb.WrapServer(
		c.GRPCServer,
		grpcweb.WithOriginFunc(func(origin string) bool {
			return c.AllowCORS
		}),
	)

	mux := &http.ServeMux{}
	for _, s := range c.SubDirectories {
		if !strings.HasPrefix(s, "/") {
			s = "/" + s
		}
		withoutSuffix := s
		if strings.HasSuffix(s, "/") {
			withoutSuffix = withoutSuffix[:len(withoutSuffix)-2]
		}
		mux.Handle(
			withoutSuffix+"/",
			http.StripPrefix(withoutSuffix,
				http.FileServer(http.Dir("."+withoutSuffix)),
			),
		)
	}

	if c.RootDirectory != "" {
		mux.Handle("/", http.FileServer(http.Dir(c.RootDirectory)))
	}

	handler := cors.AllowAll().Handler(SplitGRPCTraffic(
		mux, wrapped,
	))

	var defaultTLS TLS
	if c.TLS != defaultTLS {
		return http.ServeTLS(c.Listener, handler, c.TLS.Certificate, c.TLS.Key)
	}
	return http.Serve(c.Listener, handler)
}
