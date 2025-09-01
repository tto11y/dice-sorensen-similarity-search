package main

import (
	"dice-sorensen-similarity-search/internal/auth"
	"dice-sorensen-similarity-search/internal/bitbucket"
	"dice-sorensen-similarity-search/internal/config"
	"dice-sorensen-similarity-search/internal/constants"
	"dice-sorensen-similarity-search/internal/database"
	"dice-sorensen-similarity-search/internal/environment"
	"dice-sorensen-similarity-search/internal/logging"
	"dice-sorensen-similarity-search/internal/markdowndoc"
	"dice-sorensen-similarity-search/internal/routes"
	"fmt"
	"github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapio"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
	"io"
	"net/http/httptest"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

var initializationState sync.Map

func main() {
	c := config.InitConfig()

	logger := logging.InitLogging(c)

	controllerRegistry, err := injectDependencies(c, logger)
	if err != nil {
		logger.LogErrorf(nil, "injecting depencies failed: %s", err.Error())
		return
	}

	ginLogger := logging.InitGinLogger(c)

	gin.DefaultWriter = io.MultiWriter(&zapio.Writer{Log: ginLogger, Level: config.Config().Logging.Level})
	if config.Config().Logging.Level == zap.DebugLevel {
		logger.LogDebug(nil, "Enabling Gin debug (writes to access log)")
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()

	r.Use(
		ginzap.GinzapWithConfig(ginLogger, &ginzap.Config{
			TimeFormat: time.RFC3339,
			UTC:        false,
			SkipPaths:  []string{"/status"},
		}),
		ginzap.RecoveryWithZap(ginLogger, true),
	)

	// Routes
	routes.InitRouter(r, controllerRegistry)

	SetupCloseHandler(logger)
	go func() {
		checkAllInitializations(logger)
		// fetch markdowns on startup
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		bitbucketController := controllerRegistry[constants.Bitbucket].(bitbucket.Api)
		bitbucketController.FetchMarkdownsFromBitbucket(ctx)
	}()

	if len(config.Config().ListeningAddress) == 0 && len(config.Config().ListeningPort) == 0 {
		panic("No listening address/port provided")
	}

	logger.LogInfof(nil, "API running. Listening on %s:%s", config.Address(), config.Port())

	err = r.Run(config.Address() + ":" + config.Port())
	if err != nil {
		logger.LogErrorf(nil, "Listening on %s:%s failed: %s", config.Address(), config.Port(), err.Error())
		return
	}
}

func injectDependencies(config *config.Configuration, logger logging.Logger) (map[int]any, error) {
	db, err := database.InitDatabase(config, logger)
	if err != nil {
		logger.LogError(nil, "error initializing database: ", err)
		return nil, err
	}

	env := environment.Environment(
		&database.GormRepository{DB: db},
		logger,
	)

	bitbucketReader, err := bitbucket.InitBitbucket(config, env)
	if err != nil {
		logger.LogErrorf(logging.GetLogTypeInitialization(), "Error initializing Bitbucket Api: %v", err)
		return nil, err
	}

	bitbucketController := &bitbucket.Controller{
		Env:                 env,
		BitbucketReader:     bitbucketReader,
		ProjectName:         config.BitBucket.ProjectName,
		RepositoryName:      config.BitBucket.Repository,
		MarkdownHousekeeper: &bitbucket.DefaultMarkdownHousekeeper{Env: env},
	}

	// the Collator is used for lexicographic order with locale-aware sorting (like filesystems do),
	// instead of Go's default pure Unicode code point ordering
	c := collate.New(language.English)
	markdownDocController := &markdowndoc.Controller{
		Env:                       env,
		NavigationItemTreeService: markdowndoc.NavigationItemTreeService{Env: env, Collator: c},
		MarkdownSearchMatchMapper: markdowndoc.MarkdownSearchMatchMapper{Env: env},
	}

	authController := &auth.Controller{
		Env:         env,
		AuthService: &auth.AuthService{Env: env},
	}

	controllerRegistry := make(map[int]any)
	controllerRegistry[constants.Bitbucket] = bitbucketController
	controllerRegistry[constants.MarkdownDoc] = markdownDocController
	controllerRegistry[constants.Auth] = authController

	return controllerRegistry, nil
}

func checkAllInitializations(logger logging.Logger) {
	internalCounter := 15
	failedInits, unfinishedInits := make([]string, 0), make([]string, 0)
	time.Sleep(time.Second * 2)
	for internalCounter != 0 {
		allWorkedOn := true
		failedInits = []string{}
		unfinishedInits = []string{}
		initializationState.Range(func(key, value interface{}) bool {
			if value == "failed" {
				failedInits = append(failedInits, key.(string))
			} else {
				unfinishedInits = append(unfinishedInits, key.(string))
				logger.LogWarnf(nil, "Initialization: waiting for %v", key)
				allWorkedOn = false
			}
			return true
		})
		if allWorkedOn {
			break
		}
		time.Sleep(time.Second * 2)
		if internalCounter%5 == 0 {
			logger.LogDebug(nil, "Waiting for all initialization(s) to complete...")
		}
		internalCounter--
	}
	if len(failedInits) > 0 || len(unfinishedInits) > 0 || internalCounter == 0 {
		if len(unfinishedInits) > 0 {
			logger.LogErrorf(nil, "%v Initialization function(s) did not complete in time: %v",
				len(unfinishedInits), strings.Join(unfinishedInits, ", "))
		}
		if len(failedInits) > 0 {
			logger.LogErrorf(nil, "%v Initialization function(s) failed: %v",
				len(failedInits), strings.Join(failedInits, ", "))
		}
	} else {
		logger.LogInfo(nil, "Initialization completed successfully")
	}
}

func SetupCloseHandler(logger logging.Logger) {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-c
		fmt.Println()
		logger.LogWarnf(nil, "Cleaning up...")
		time.Sleep(1 * time.Second)
		os.Exit(1)
	}()
}
