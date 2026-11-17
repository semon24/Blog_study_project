package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	blogpb "blog-study-project/gen/blog/v1"
	"blog-study-project/internal/blog"
	"blog-study-project/internal/storage"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	dsn := "host=localhost port=5432 user=blog password=blog_password dbname=blog sslmode=disable"

	db, err := storage.OpenPostgres(dsn)
	if err != nil {
		log.Fatalf("Ошибка подключения к PostgreSQL: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Ошибка получения подключения: %v", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("PostgreSQL недоступен: %v", err)
	}

	log.Println("Подключение к PostgreSQL установлено")

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer redisClient.Close()

	redisCtx, redisCancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = redisClient.Ping(redisCtx).Err()
	redisCancel()
	if err != nil {
		log.Fatalf("Подключение к Redis не удалось: %v", err)
	}
	log.Println("Подключение к Redis установлено")

	listener, err := net.Listen("tcp", "localhost:8000")
	if err != nil {
		log.Fatal(err)
	}

	if err := db.AutoMigrate(&storage.Author{}, &storage.Post{}); err != nil {
		log.Fatalf("Ошибка создания таблиц: %v", err)
	}

	log.Println("Таблицы базы данных готовы")

	grpcServer := grpc.NewServer()

	blogServer := &blog.Server{DB: db, Redis: redisClient}
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
