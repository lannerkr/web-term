package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"strconv"

	"github.com/dchest/uniuri"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	adapter "github.com/gwatts/gin-adapter"

	//"github.com/syssecfsu/witty/term_conn"
	"lannerkr/witty/term_conn"
)

type Options struct {
	Wait      uint
	Port      uint
	NoAuth    bool
	CmdToExec []string
	Assets    fs.FS
	LogFile   *os.File
}

var options Options

func StartWeb(opt *Options) {
	//fmt.Println("start web")
	options = *opt

	if options.LogFile != nil {
		gin.DefaultWriter = options.LogFile
	}

	rt := gin.Default()

	// We randomly generate a key for now, should use a fixed key
	// so login can survive server reboot
	store := sessions.NewCookieStore([]byte(uniuri.NewLen(32)))
	rt.Use(sessions.Sessions("witty-session", store))

	csrfHttp := csrf.Protect([]byte(uniuri.NewLen(32)), csrf.Path("/"))
	csrfGin := adapter.Wrap(csrfHttp)
	rt.Use(csrfGin)

	rt.SetTrustedProxies(nil)

	//templ := template.Must(template.New("assets").ParseFS(options.Assets, "template/*.html"))
	templ := template.Must(template.New("assets").ParseGlob("./template/*.html"))
	//addTempl := template.Must(templ.ParseFiles("/usr/local/witty/template/indexN.html"))
	//addTempl := template.Must(templ.ParseFiles("./template/indexN.html", "./template/newSession.html"))

	rt.SetHTMLTemplate(templ)

	// handle static files
	rt.StaticFS("/assets", http.FS(options.Assets))
	//rt.Static("/records", "./records")
	//rt.LoadHTMLGlob("template/test.html")

	rt.GET("/login", loginPage)
	rt.POST("/login", login)

	g1 := rt.Group("/")

	if !options.NoAuth {
		g1.Use(AuthRequired)
	}

	// Fill in the index page
	g1.GET("/", indexPage)
	g1.GET("/logout", logout)

	// to update the tabs of current interactive and saved sessions
	g1.GET("/update/:active", updateIndex)

	// create a new interactive session
	g1.POST("/new", newInteractive)
	g1.POST("/new/:id", newInteractive)
	g1.GET("/ws_new/:id", newTermConn)

	// create a viewer of an interactive session
	g1.GET("/view/:id", viewPage)
	g1.GET("/ws_view/:id", newViewWS)

	// g1.GET("/test", func(c *gin.Context) {
	// 	c.JSON(200, gin.H{
	// 		"message": "pong",
	// 	})
	// })
	g1.GET("/newSess", newSessHandler)
	g1.POST("/newSess", newSessPostHandler)
	g1.GET("/conf", viewConfigHandler)
	g1.GET("/conf2", viewConfig2Handler)
	g1.GET("/conf2:id", viewConfig2Handler)
	g1.GET("/editSession/:gid/:sid", editSessHandler)
	g1.POST("/editSession/:gid/:sid", editSessPostHandler)
	g1.POST("/editSession/:gid/:sid/del", editSessDelHandler)
	g1.GET("/editGroup/:gid", editGrpHandler)
	g1.POST("/editGroup/:gid", editGrpPostHandler)
	g1.POST("/editGroup/:gid/del", editGrpDelHandler)

	// start/stop recording the session
	// g1.POST("/record/:id", startRecord)
	// g1.POST("/stop/:id", stopRecord)

	// create a viewer of an interactive session
	// g1.GET("/replay/:id", replayPage)

	// delete a recording
	// g1.POST("/delete/:fname", delRec)
	// Rename a recording
	// g1.POST("/rename/:oldname/:newname", renameRec)

	term_conn.Init()
	port := strconv.FormatUint(uint64(uint16(options.Port)), 10)
	rt.RunTLS(":"+port, "./tls/cert.pem", "./tls/private-key.pem")
}

type NewSess struct {
	Gname   string `form:"gname"`
	Sname   string `form:"sname"`
	Host    string `form:"host"`
	Port    string `form:"port"`
	User    string `form:"user"`
	Pwd     string `form:"password"`
	Pkey    string `form:"pkey"`
	Actionw string `form:"actionw"`
	Actiond string `form:"actiond"`
}

func newSessPostHandler(c *gin.Context) {
	session := &NewSess{}
	if err := c.Bind(session); err != nil {
		fmt.Println(err)
		return
	}
	//fmt.Println(session)

	updateConfig(session)
	//c.Redirect(http.StatusFound, "/conf")
	// c.HTML(http.StatusOK, "redirectHome.html", nil)
	id := session.Sname + ":newsessionupdate"
	c.HTML(http.StatusOK, "term.html", gin.H{
		"title":     "interactive terminal",
		"path":      "/ws_new/" + id,
		"id":        id,
		"id2":       "newsessionupdate",
		"logo":      "keyboard",
		"csrfToken": csrf.Token(c.Request),
	})
}
func newSessHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "newSession.html",
		gin.H{
			"csrfField": csrf.TemplateField(c.Request),
			"csrfToken": csrf.Token(c.Request),
		})
}

func updateConfig(s *NewSess) {

	var exist bool = false
	var newSession Session

	var newPwd []byte
	var err error
	if s.Pwd != "" {
		newPwd, err = encrypt([]byte(s.Pwd))
		if err != nil {
			fmt.Println(err)
		}
		newSession.Usepwd = true
	}

	newSession.Name = s.Sname
	newSession.Host = s.Host
	newSession.Port = s.Port
	newSession.User = s.User
	newSession.Pwd = newPwd
	newSession.KeyPath = s.Pkey
	newSession.Actions = append(newSession.Actions, s.Actionw)
	newSession.Actions = append(newSession.Actions, s.Actiond)

	var result Group
	getConfig(&result)

	for i, v := range result.Groups {
		if s.Gname == v.Gname {
			fmt.Println("group name is existed ...")
			newSession.Id.Gid = i
			newSession.Id.Sid = len(result.Groups[i].Sessions)

			result.Groups[i].Sessions = append(result.Groups[i].Sessions, newSession)
			exist = true
			break
		}
	}

	if !exist {
		var newConf Groups
		newConf.Gname = s.Gname
		newConf.Sessions = append(newConf.Sessions, newSession)
		newConf.Sessions[0].Id.Gid = len(result.Groups)
		result.Groups = append(result.Groups, newConf)
	}

	fmt.Println("=======================")
	fmt.Println(result)
	fmt.Println("=======================")

	jsonString, _ := json.Marshal(result)
	os.WriteFile("/usr/local/witty/config/sessionconf.json", jsonString, os.ModePerm)

}

func editConfig(s *NewSess, gid, sid int) {
	//var exist bool = false
	var newSession Session
	var newPwd []byte
	var err error
	if s.Pwd != "" {
		newPwd, err = encrypt([]byte(s.Pwd))
		if err != nil {
			fmt.Println(err)
		}
		newSession.Usepwd = true
	}

	newSession.Name = s.Sname
	newSession.Host = s.Host
	newSession.Port = s.Port
	newSession.User = s.User
	newSession.Pwd = newPwd
	newSession.KeyPath = s.Pkey
	newSession.Id.Gid = gid
	newSession.Id.Sid = sid
	newSession.Actions = append(newSession.Actions, s.Actionw)
	newSession.Actions = append(newSession.Actions, s.Actiond)

	var result Group
	getConfig(&result)
	result.Groups[gid].Sessions[sid] = newSession

	// LOOP:
	// 	for i, v := range result.Groups {
	// 		if v.Gname == s.Gname {
	// 			for j, sess := range v.Sessions {
	// 				if sess.Name == s.Sname {
	// 					result.Groups[i].Sessions[j] = newSession
	// 					exist = true
	// 					break LOOP
	// 				}
	// 			}
	// 		}
	// 	}

	// 	if !exist {
	// 		return
	// 	}

	fmt.Println("=======================")
	fmt.Println(result)
	fmt.Println("=======================")

	jsonString, _ := json.Marshal(result)
	os.WriteFile("/usr/local/witty/config/sessionconf.json", jsonString, os.ModePerm)

}
func delConfig(gid, sid int) {
	var result Group
	getConfig(&result)

	group := result.Groups[gid]
	newSessions := append(group.Sessions[:sid], group.Sessions[sid+1:]...)
	result.Groups[gid].Sessions = newSessions
	for i := range result.Groups[gid].Sessions {
		result.Groups[gid].Sessions[i].Id.Gid = gid
		result.Groups[gid].Sessions[i].Id.Sid = i
	}

	jsonString, _ := json.Marshal(result)
	os.WriteFile("/usr/local/witty/config/sessionconf.json", jsonString, os.ModePerm)
}

func viewConfigHandler(c *gin.Context) {
	var result Group
	getConfig(&result)

	c.HTML(http.StatusOK, "configview.html",
		gin.H{
			"csrfField": csrf.TemplateField(c.Request),
			"csrfToken": csrf.Token(c.Request),
			"group":     result.Groups,
		})
}
func viewConfig2Handler(c *gin.Context) {
	var result Group
	getConfig(&result)

	c.HTML(http.StatusOK, "configview2.html",
		gin.H{
			"csrfField": csrf.TemplateField(c.Request),
			"csrfToken": csrf.Token(c.Request),
			"groups":    result.Groups,
			"inframe":   c.Param("id"),
		})
}

func editSessPostHandler(c *gin.Context) {
	gid, _ := strconv.Atoi(c.Param("gid"))
	sid, _ := strconv.Atoi(c.Param("sid"))
	session := &NewSess{}
	if err := c.Bind(session); err != nil {
		fmt.Println(err)
		return
	}
	//fmt.Println(session)
	editConfig(session, gid, sid)
	//c.Redirect(http.StatusFound, "/conf")
	c.HTML(http.StatusOK, "redirectHome.html", nil)
}
func editSessDelHandler(c *gin.Context) {
	gid, _ := strconv.Atoi(c.Param("gid"))
	sid, _ := strconv.Atoi(c.Param("sid"))

	delConfig(gid, sid)
	c.Redirect(http.StatusFound, "/conf2")
	//c.HTML(http.StatusOK, "redirectHome.html", nil)
}
func editSessHandler(c *gin.Context) {
	gid, _ := strconv.Atoi(c.Param("gid"))
	sid, _ := strconv.Atoi(c.Param("sid"))
	var result Group
	getConfig(&result)
	//fmt.Printf("%v : %v\n", gid, sid)
	groupname := result.Groups[gid].Gname
	session := result.Groups[gid].Sessions[sid]
	actions := result.Groups[gid].Sessions[sid].Actions
	//pid := fmt.Sprintln(session.Usepwd)

	c.HTML(http.StatusOK, "editSession.html",
		gin.H{
			"csrfField": csrf.TemplateField(c.Request),
			"csrfToken": csrf.Token(c.Request),
			"group":     groupname,
			"session":   session,
			"actions":   actions,
			"gid":       gid,
			"sid":       sid,
		})
}

func editGrpPostHandler(c *gin.Context) {
	gid, _ := strconv.Atoi(c.Param("gid"))

	session := &NewSess{}
	if err := c.Bind(session); err != nil {
		fmt.Println(err)
		return
	}

	var result Group
	getConfig(&result)
	result.Groups[gid].Gname = session.Gname
	jsonString, _ := json.Marshal(result)
	os.WriteFile("/usr/local/witty/config/sessionconf.json", jsonString, os.ModePerm)

	c.HTML(http.StatusOK, "redirectHome.html", nil)
}
func editGrpDelHandler(c *gin.Context) {
	gid, _ := strconv.Atoi(c.Param("gid"))

	var result Group
	getConfig(&result)
	result.Groups = append(result.Groups[:gid], result.Groups[gid+1:]...)
	for i := 0; i < len(result.Groups); i++ {
		for j := range result.Groups[i].Sessions {
			result.Groups[i].Sessions[j].Id.Gid = i
		}
	}

	jsonString, _ := json.Marshal(result)
	os.WriteFile("/usr/local/witty/config/sessionconf.json", jsonString, os.ModePerm)

	c.Redirect(http.StatusFound, "/conf2")
}
func editGrpHandler(c *gin.Context) {
	gid, _ := strconv.Atoi(c.Param("gid"))
	//sid, _ := strconv.Atoi(c.Param("sid"))
	var result Group
	getConfig(&result)
	groupname := result.Groups[gid].Gname
	sessions := result.Groups[gid].Sessions

	c.HTML(http.StatusOK, "editGroup.html",
		gin.H{
			"csrfField": csrf.TemplateField(c.Request),
			"csrfToken": csrf.Token(c.Request),
			"group":     groupname,
			"sessions":  sessions,
			"gid":       gid,
		})
}
