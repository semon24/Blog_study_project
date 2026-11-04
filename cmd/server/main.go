package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"strings"

	blogpb "blog-study-project/gen/blog/v1"
	"blog-study-project/internal/blog"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	listener, err := net.Listen("tcp", "localhost:8000")
	if err != nil {
		log.Fatal(err)
	}

	grpcServer := grpc.NewServer()

	blogServer := &blog.Server{}
	blogpb.RegisterBlogServiceServer(grpcServer, blogServer)

	go func() {
		log.Println("gRPC-сервер запущен на localhost:8000")
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal(err)
		}
	}()
	defer grpcServer.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gateway := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			if strings.EqualFold(key, "X-User-ID") {
				return "x-user-id", true
			}
			return runtime.DefaultHeaderMatcher(key)
		}),
	)

	err = blogpb.RegisterBlogServiceHandlerFromEndpoint(
		ctx,
		gateway,
		"localhost:8000",
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
	)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "api/openapi/blog.swagger.json")
	})
	mux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/openapi.json"),
	))
	mux.Handle("/", gateway)

	log.Println("HTTP Gateway запущен на http://localhost:8080")
	log.Println("Swagger UI: http://localhost:8080/swagger/index.html")
	err = http.ListenAndServe("localhost:8080", mux)
	if err != nil {
		log.Fatal(err)
	}
}
