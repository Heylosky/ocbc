package main

import (
	"fmt"
	"github.com/Heylosky/ocbcProject/config"
	"github.com/Heylosky/ocbcProject/controller"
	"github.com/Heylosky/ocbcProject/service"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
	"github.com/kataras/iris/v12/sessions"
	"os"
	"path"
	"runtime"
	"strings"
	"time"
)

func init() {
}

func main() {
	config.Configs = config.InitConfig()
	addr := ":" + config.Configs.Port

	app := newApp()

	app.Logger().SetLevel(config.Configs.LogLevel)
	f := newLogFile()
	defer f.Close()
	app.Logger().SetOutput(f)
	app.Logger().SetTimeFormat("2006-01-02T15:04:05-07:00")

	configuration(app)

	mvcHandle(app)

	app.Run(
		//iris.Addr(addr),
		//iris.TLS(addr, "./7542961/7542961_www.sdesk.vip.pem", "./7542961/7542961_www.sdesk.vip.key"),
		iris.TLS(addr, config.Configs.CertPath.CertFile, config.Configs.CertPath.KeyFile),
		iris.WithoutServerError(iris.ErrServerClosed),
		//iris.WithOptimizations,
	)
}

func todayFilename() string {
	today := time.Now().Format("Jan-02-2006")
	return "./logs/" + today + ".txt"
}

func newLogFile() *os.File {
	filename := todayFilename()
	// Open the file, this will append to the today's file if server restarted.
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0777)
	if err != nil {
		fmt.Println(err)
	}
	return f
}

func newApp() *iris.Application {
	app := iris.New()

	app.Use(before)

	app.HandleDir("/static", "./static")
	app.HandleDir("/manage/static", "./static")
	app.HandleDir("/img", "./static/img")

	app.RegisterView(iris.HTML("./static", ".html"))
	app.Get("/", func(ctx iris.Context) {
		ctx.View("index.html")
	})
	/*app.Get("/path", func(ctx iris.Context) {
		filePath := ctx.Path()
		app.Logger().Info(filePath)
		ctx.WriteString("请求的路径：" + filePath)
	})*/

	app.Get("/bak/rsa", func(ctx iris.Context) {
		/*ctx.Header("Access-Control-Allow-Origin", "http://127.0.0.1:8080")
		ctx.Header("Access-Control-Allow-Credentials", "true")*/

		//obtain customer ip address
		ip := strings.Split(ctx.Request().Header.Get("x-Forwarded-For"), ",")[0]
		if ip == "" {
			ip = ctx.Request().Header.Get("X-Real-Ip")
		}
		if ip == "" {
			ip = ctx.Request().RemoteAddr
		}

		//obtain program meta info
		pc, file, line, ok := runtime.Caller(0)
		if !ok {
			app.Logger().Warn("runtime.Caller() failed")
		}
		funcName := runtime.FuncForPC(pc).Name() //get function name
		fileName := path.Base(file)              // get file name

		app.Logger().Debugf("%s %s line%d: %s GET: %s: request public key", funcName, fileName, line, ip, ctx.Path())
		app.Logger().Infof("%s GET: %s: request public key", ip, ctx.Path())
		content, err := os.ReadFile(config.Configs.RsaPath.PublicKey)
		if err != nil {
			app.Logger().Warnf("%s %s line%d: %", funcName, fileName, line, err)
		}

		ctx.WriteString(string(content))
	})

	return app
}

func mvcHandle(app *iris.Application) {
	sessManager := sessions.New(sessions.Config{
		Cookie:  "mycookiesessionnameid",
		Expires: -1,
	})

	adminService := service.NewAdminService()
	admin := mvc.New(app.Party("/admin"))
	admin.Register(
		adminService,
		sessManager.Start,
	)
	admin.Handle(new(controller.AdminController))

	//数据展示模块
	//peopleService := service.NewPeopleService()
	people := mvc.New(app.Party("/people"))
	people.Register(
		//peopleService,
		sessManager.Start,
	)
	people.Handle(new(controller.PeopleController))
}

func before(ctx iris.Context) {
	//get customer ip address
	ip := strings.Split(ctx.Request().Header.Get("x-Forwarded-For"), ",")[0]
	if ip == "" {
		ip = ctx.Request().Header.Get("X-Real-Ip")
	}
	if ip == "" {
		ip = ctx.Request().RemoteAddr
	}
	iris.New().Logger().Infof("%s request: %s", ip, ctx.Path())
	ctx.Next()
}

func configuration(app *iris.Application) {
	app.Configure(iris.WithConfiguration(iris.Configuration{
		Charset: "UTF-8",
	}))

	app.OnErrorCode(iris.StatusNotFound, func(context iris.Context) {
		context.JSON(iris.Map{
			"errmsg": iris.StatusNotFound,
			"msg":    "Page Not Found",
			"data":   iris.Map{},
		})
	})

	app.OnErrorCode(iris.StatusInternalServerError, func(context iris.Context) {
		context.JSON(iris.Map{
			"errmsg": iris.StatusInternalServerError,
			"msg":    "Server Internal Error",
			"data":   iris.Map{},
		})
	})
}
