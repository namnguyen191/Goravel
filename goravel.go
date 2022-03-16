package goravel

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/CloudyKit/jet/v6"
	"github.com/alexedwards/scs/v2"
	"github.com/dgraph-io/badger/v3"
	"github.com/go-chi/chi/v5"
	"github.com/gomodule/redigo/redis"
	"github.com/joho/godotenv"
	"github.com/namnguyen191/goravel/cache"
	"github.com/namnguyen191/goravel/mailer"
	"github.com/namnguyen191/goravel/render"
	"github.com/namnguyen191/goravel/session"
	"github.com/robfig/cron/v3"
)

const version = "1.0.0"

var myRedisCache *cache.RedisCache
var myBadgerCache *cache.BadgerCache
var redisPool *redis.Pool
var badgerConn *badger.DB

type Goravel struct {
	AppName       string
	Debug         bool
	Version       string
	ErrorLog      *log.Logger
	InfoLog       *log.Logger
	RootPath      string
	Routes        *chi.Mux
	Render        *render.Render
	Session       *scs.SessionManager
	DB            Database
	JetViews      *jet.Set
	config        config
	EncryptionKey string
	Cache         cache.Cache
	Scheduler     *cron.Cron
	Mail          mailer.Mail
	Server        Server
}

type config struct {
	// the port the server will listen on
	port string
	// the renderer engine that the app will be using (jet or go)
	renderer    string
	cookie      cookieConfig
	sessionType string
	database    databaseConfig
	redis       redisConfig
}

type Server struct {
	ServerName string
	Port       string
	Secure     bool
	URL        string
}

func (grv *Goravel) New(rootPath string) error {
	pathConfig := initPaths{
		rootPath:    rootPath,
		folderNames: []string{"handlers", "migrations", "views", "mail", "data", "public", "tmp", "logs", "middleware"},
	}

	err := grv.Init(pathConfig)

	if err != nil {
		return err
	}

	err = grv.checkDotEnv(rootPath)
	if err != nil {
		return err
	}

	// read .env
	err = godotenv.Load(rootPath + "/.env")
	if err != nil {
		return err
	}

	// connect to db
	if os.Getenv("DATABASE_TYPE") != "" {
		db, err := grv.OpenDB(os.Getenv("DATABASE_TYPE"), grv.BuildDSN())
		if err != nil {
			grv.ErrorLog.Println(err)
			os.Exit(1)
		}

		grv.DB = Database{
			DataBaseType: os.Getenv("DATABASE_TYPE"),
			Pool:         db,
		}
	}

	scheduler := cron.New()
	grv.Scheduler = scheduler

	// create cache
	if os.Getenv("CACHE") == "redis" || os.Getenv("SESSION_TYPE") == "redis" {
		myRedisCache = grv.createClientRedisCache()
		grv.Cache = myRedisCache
		redisPool = myRedisCache.Conn
	}
	if os.Getenv("CACHE") == "badger" {
		myBadgerCache = grv.createClientBadgerCache()
		grv.Cache = myBadgerCache
		badgerConn = myBadgerCache.Conn

		_, err := grv.Scheduler.AddFunc("@daily", func() {
			myBadgerCache.Conn.RunValueLogGC(0.7)
		})
		if err != nil {
			return err
		}
	}

	// create logger
	infoLog, errorLog := grv.startLoggers()
	grv.InfoLog = infoLog
	grv.ErrorLog = errorLog

	grv.Debug, _ = strconv.ParseBool(os.Getenv("DEBUG"))
	grv.Version = version
	grv.RootPath = rootPath

	// create mail
	grv.Mail = grv.createMailer()

	grv.Routes = grv.routes().(*chi.Mux)

	grv.config = config{
		port:     os.Getenv("PORT"),
		renderer: os.Getenv("RENDERER"),
		cookie: cookieConfig{
			name:     os.Getenv("COOKIE_NAME"),
			lifetime: os.Getenv("COOKIE_LIFETIME"),
			persist:  os.Getenv("COOKIE_PERSISTS"),
			secure:   os.Getenv("COOKIE_SECURE"),
			domain:   os.Getenv("COOKIE_DOMAIN"),
		},
		sessionType: os.Getenv("SESSION_TYPE"),
		database: databaseConfig{
			database: os.Getenv("DATABASE_TYPE"),
			dsn:      grv.BuildDSN(),
		},
		redis: redisConfig{
			host:     os.Getenv("REDIS_HOST"),
			password: os.Getenv("REDIS_PASSWORD"),
			prefix:   os.Getenv("REDIS_PREFIX"),
		},
	}

	secure := true
	if strings.ToLower(os.Getenv("SECURE")) == "false" {
		secure = false
	}

	grv.Server = Server{
		ServerName: os.Getenv("SEVER_NAME"),
		Port:       os.Getenv("PORT"),
		Secure:     secure,
		URL:        os.Getenv("APP_URL"),
	}

	// create a Session
	sess := session.Session{
		CookieLifeTime: grv.config.cookie.lifetime,
		CookiePersist:  grv.config.cookie.persist,
		CookieName:     grv.config.cookie.name,
		SessionType:    grv.config.sessionType,
		CookieDomain:   grv.config.cookie.domain,
	}

	switch grv.config.sessionType {
	case "redis":
		{
			sess.RedisPool = myRedisCache.Conn
		}
	case "mysql", "postgres", "mariadb", "postgresql":
		{
			sess.DBPool = grv.DB.Pool
		}
	}

	grv.Session = sess.InitSession()

	grv.EncryptionKey = os.Getenv("KEY")

	if grv.Debug {
		var views = jet.NewSet(
			jet.NewOSFileSystemLoader(fmt.Sprintf("%s/views", rootPath)),
			jet.InDevelopmentMode(),
		)

		grv.JetViews = views
	} else {
		var views = jet.NewSet(
			jet.NewOSFileSystemLoader(fmt.Sprintf("%s/views", rootPath)),
		)

		grv.JetViews = views
	}

	grv.createRenderer()

	go grv.Mail.ListenForMail()

	return nil
}

func (grv *Goravel) Init(p initPaths) error {
	root := p.rootPath

	for _, path := range p.folderNames {
		// create folder if it does not exist
		err := grv.CreateDirIfNotExist(root + "/" + path)

		if err != nil {
			return err
		}
	}

	return nil
}

// ListenAndServe starts web server
func (grv *Goravel) ListenAndServe() {
	srv := http.Server{
		Addr:         fmt.Sprintf(":%s", os.Getenv("PORT")),
		ErrorLog:     grv.ErrorLog,
		Handler:      grv.Routes,
		IdleTimeout:  30 * time.Second,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 600 * time.Second,
	}

	// close DB when app close
	if grv.DB.Pool != nil {
		defer grv.DB.Pool.Close()
	}

	if redisPool != nil {
		defer redisPool.Close()
	}

	if badgerConn != nil {
		defer badgerConn.Close()
	}

	grv.InfoLog.Printf("Listening on port %s", os.Getenv("PORT"))
	err := srv.ListenAndServe()
	grv.ErrorLog.Fatal(err)
}

func (grv *Goravel) checkDotEnv(path string) error {
	err := grv.CreateFileIfNotExist(fmt.Sprintf("%s/.env", path))

	if err != nil {
		return err
	}

	return nil
}

func (grv *Goravel) CreateFileIfNotExist(path string) error {
	var _, err = os.Stat(path)

	if os.IsNotExist(err) {
		var file, err = os.Create(path)

		if err != nil {
			return err
		}

		defer func(file *os.File) {
			_ = file.Close()
		}(file)
	}

	return nil
}

func (grv *Goravel) startLoggers() (*log.Logger, *log.Logger) {
	var infoLog *log.Logger
	var errorLog *log.Logger

	infoLog = log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog = log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	return infoLog, errorLog
}

func (grv *Goravel) createRenderer() {
	myRenderer := render.Render{
		Renderer: grv.config.renderer,
		RootPath: grv.RootPath,
		Port:     grv.config.port,
		JetViews: grv.JetViews,
		Session:  grv.Session,
	}

	grv.Render = &myRenderer
}

func (grv *Goravel) createMailer() mailer.Mail {
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	m := mailer.Mail{
		Domain:      os.Getenv("MAIL_DOMAIN"),
		Templates:   grv.RootPath + "/mail",
		Host:        os.Getenv("SMTP_HOST"),
		Port:        port,
		Username:    os.Getenv("SMTP_USERNAME"),
		Password:    os.Getenv("SMTP_PASSWORD"),
		Encryption:  os.Getenv("SMTP_ENCRYPTION"),
		FromName:    os.Getenv("FROM_NAME"),
		FromAddress: os.Getenv("FROM_ADDRESS"),
		Jobs:        make(chan mailer.Message, 20),
		Results:     make(chan mailer.Result, 20),
		API:         os.Getenv("MAILER_API"),
		APIKey:      os.Getenv("MAILER_KEY"),
		APIUrl:      os.Getenv("MAILER_URL"),
	}

	return m
}

func (grv *Goravel) createClientRedisCache() *cache.RedisCache {
	cacheClient := cache.RedisCache{
		Conn:   grv.createRedisPool(),
		Prefix: grv.config.redis.prefix,
	}

	return &cacheClient
}

func (grv *Goravel) createClientBadgerCache() *cache.BadgerCache {
	cacheClient := cache.BadgerCache{
		Conn:   grv.createBadgerConn(),
		Prefix: grv.config.redis.prefix,
	}

	return &cacheClient
}

func (grv *Goravel) createRedisPool() *redis.Pool {
	return &redis.Pool{
		MaxIdle:     50,
		MaxActive:   10000,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial(
				"tcp",
				grv.config.redis.host,
				redis.DialPassword(grv.config.redis.password),
			)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")

			return err
		},
	}
}

func (grv *Goravel) createBadgerConn() *badger.DB {
	db, err := badger.Open(badger.DefaultOptions(grv.RootPath + "/tmp/badger"))
	if err != nil {
		return nil
	}

	return db
}

func (grv *Goravel) BuildDSN() string {
	var dsn string

	switch os.Getenv("DATABASE_TYPE") {
	case "postgres", "postgresql":
		dsn = fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=%s timezone=UTC connect_timeout=5",
			os.Getenv("DATABASE_HOST"),
			os.Getenv("DATABASE_PORT"),
			os.Getenv("DATABASE_USER"),
			os.Getenv("DATABASE_NAME"),
			os.Getenv("DATABASE_SSL_MODE"),
		)

		if os.Getenv("DATABASE_PASS") != "" {
			dsn = fmt.Sprintf("%s password=%s", dsn, os.Getenv("DATABASE_PASS"))
		}
	default:
	}

	return dsn
}
