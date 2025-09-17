package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"logbull/internal/config"
	"logbull/internal/downdetect"
	"logbull/internal/features/api_keys"
	"logbull/internal/features/audit_logs"
	"logbull/internal/features/disk"
	logs_cleanup "logbull/internal/features/logs/cleanup"
	logs_core "logbull/internal/features/logs/core"
	logs_querying "logbull/internal/features/logs/querying"
	logs_receiving "logbull/internal/features/logs/receiving"

	// logs_cleanup "logbull/internal/features/logs/cleanup"
	// logs_querying "logbull/internal/features/logs/querying"
	// logs_receiving "logbull/internal/features/logs/receiving"
	projects_controllers "logbull/internal/features/projects/controllers"
	system_healthcheck "logbull/internal/features/system/healthcheck"
	users_controllers "logbull/internal/features/users/controllers"
	users_middleware "logbull/internal/features/users/middleware"
	users_services "logbull/internal/features/users/services"
	cache_utils "logbull/internal/util/cache"
	env_utils "logbull/internal/util/env"
	"logbull/internal/util/logger"
	_ "logbull/swagger" // swagger docs

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title LogBull Backend API
// @version 1.0
// @description API for LogBull
// @termsOfService http://swagger.io/terms/

// @host localhost:4005
// @BasePath /api/v1
// @schemes http
func main() {
	log := logger.GetLogger()
	config.StartListeningForShutdownSignal()
	setUpDependencies()

	cache_utils.TestCacheConnection()

	testOpenSearchConnection(log)

	runMigrations(log)

	err := users_services.GetUserService().CreateInitialAdmin()
	if err != nil {
		log.Error("Failed to create initial admin", "error", err)
		os.Exit(1)
	}

	handlePasswordReset(log)

	go generateSwaggerDocs(log)

	gin.SetMode(gin.ReleaseMode)
	ginApp := gin.Default()

	// Add GZIP compression middleware
	ginApp.Use(gzip.Gzip(
		gzip.DefaultCompression,
		// Don't compress already compressed files
		gzip.WithExcludedExtensions(
			[]string{".png", ".gif", ".jpeg", ".jpg", ".ico", ".svg", ".pdf", ".mp4"},
		),
	))

	enableCors(ginApp)
	setUpRoutes(ginApp)
	runBackgroundTasks(log)
	mountFrontend(ginApp)

	startServerWithGracefulShutdown(log, ginApp)
}

func startServerWithGracefulShutdown(log *slog.Logger, app *gin.Engine) {
	host := ""
	if config.GetEnv().EnvMode == env_utils.EnvModeDevelopment {
		// for dev we use localhost to avoid firewall
		// requests on each run for Windows
		host = "127.0.0.1"
	}

	srv := &http.Server{
		Addr:    host + ":4005",
		Handler: app,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("listen:", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Info("Shutdown signal received")

	// The context is used to inform the server it has 10 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown:", "error", err)
	}

	log.Info("Server gracefully stopped")
}

func setUpRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")

	// Mount Swagger UI
	v1.GET("/docs/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Public routes (only user auth routes should be public)
	userController := users_controllers.GetUserController()
	userController.RegisterRoutes(v1)
	logs_receiving.GetReceivingController().RegisterRoutes(v1)
	downdetect.GetDowndetectController().RegisterRoutes(v1)
	system_healthcheck.GetHealthcheckController().RegisterRoutes(v1)

	// Setup auth middleware
	userService := users_services.GetUserService()
	authMiddleware := users_middleware.AuthMiddleware(userService)

	// Protected routes
	protected := v1.Group("")
	protected.Use(authMiddleware)

	disk.GetDiskController().RegisterRoutes(protected)
	audit_logs.GetAuditLogController().RegisterRoutes(protected)
	userController.RegisterProtectedRoutes(protected)
	users_controllers.GetSettingsController().RegisterRoutes(protected)
	users_controllers.GetManagementController().RegisterRoutes(protected)
	projects_controllers.GetProjectController().RegisterRoutes(protected)
	projects_controllers.GetMembershipController().RegisterRoutes(protected)
	api_keys.GetApiKeyController().RegisterRoutes(protected)
	logs_querying.GetLogQueryController().RegisterRoutes(protected)
}

func setUpDependencies() {
	audit_logs.SetupDependencies()
	logs_core.SetupDependencies()
}

func runBackgroundTasks(log *slog.Logger) {
	log.Info("Preparing to run background tasks...")

	if err := logs_querying.GetLogQueryService().CleanupPendingQueries(); err != nil {
		log.Error("Failed to cleanup pending queries on startup", slog.String("error", err.Error()))
	}

	logs_receiving.GetLogWorkerService().StartWorkers()
	logs_cleanup.GetLogCleanupBackgroundService().StartWorkers()

	log.Info("Background tasks started successfully")
}

// Keep in mind: docs appear after second launch, because Swagger
// is generated into Go files. So if we changed files, we generate
// new docs, but still need to restart the server to see them.
func generateSwaggerDocs(log *slog.Logger) {
	if config.GetEnv().EnvMode == env_utils.EnvModeProduction {
		return
	}

	// Run swag from the current directory instead of parent
	// Use the current directory as the base for swag init
	// This ensures swag can find the files regardless of where the command is run from
	currentDir, err := os.Getwd()
	if err != nil {
		log.Error("Failed to get current directory", "error", err)
		return
	}

	cmd := exec.Command("swag", "init", "-d", currentDir, "-g", "cmd/main.go", "-o", "swagger")

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("Failed to generate Swagger docs", "error", err, "output", string(output))
		return
	}

	log.Info("Swagger documentation generated successfully")
}

func testOpenSearchConnection(log *slog.Logger) {
	log.Info("Testing OpenSearch connection...")

	repository := logs_core.GetLogCoreRepository()
	err := repository.TestOpenSearchConnection()
	if err != nil {
		log.Error("Failed to connect to OpenSearch", "error", err)
		os.Exit(1)
	}

	log.Info("OpenSearch connection test successful")
}

func runMigrations(log *slog.Logger) {
	log.Info("Running database migrations...")

	cmd := exec.Command("goose", "up")
	cmd.Env = append(
		os.Environ(),
		"GOOSE_DRIVER=postgres",
		"GOOSE_DBSTRING="+config.GetEnv().DatabaseDsn,
	)

	cmd.Dir = config.GetEnv().BackendRootPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("Failed to run migrations", "error", err, "output", string(output))
		os.Exit(1)
	}

	log.Info("Database migrations completed successfully", "output", string(output))
}

func enableCors(ginApp *gin.Engine) {
	if config.GetEnv().EnvMode == env_utils.EnvModeDevelopment {
		// Setup CORS
		ginApp.Use(cors.New(cors.Config{
			AllowOrigins: []string{"*"},
			AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
			AllowHeaders: []string{
				"Origin",
				"Content-Length",
				"Content-Type",
				"Authorization",
				"Accept",
				"Accept-Language",
				"Accept-Encoding",
				"Access-Control-Request-Method",
				"Access-Control-Request-Headers",
				"Access-Control-Allow-Methods",
				"Access-Control-Allow-Headers",
				"Access-Control-Allow-Origin",
			},
			AllowCredentials: true,
		}))
	}
}

func mountFrontend(ginApp *gin.Engine) {
	staticDir := "./ui/build"
	ginApp.NoRoute(func(c *gin.Context) {
		path := filepath.Join(staticDir, c.Request.URL.Path)

		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			c.File(path)
			return
		}

		c.File(filepath.Join(staticDir, "index.html"))
	})
}

func handlePasswordReset(log *slog.Logger) {
	// Handle password reset if flag is provided
	newPassword := flag.String("new-password", "", "Set a new password for the user")
	email := flag.String("email", "", "Email of the user to reset password")

	flag.Parse()

	if *newPassword == "" {
		return
	}

	log.Info("Found reset password command - reseting password...")

	if *email == "" {
		log.Info("No email provided, please provide an email via --email=\"some@email.com\" flag")
		os.Exit(1)
	}

	resetPassword(*email, *newPassword, log)
}

func resetPassword(email string, newPassword string, log *slog.Logger) {
	log.Info("Resetting password...")

	userService := users_services.GetUserService()
	err := userService.ChangeUserPasswordByEmail(email, newPassword)
	if err != nil {
		log.Error("Failed to reset password", "error", err)
		os.Exit(1)
	}

	log.Info("Password reset successfully")
	os.Exit(0)
}
