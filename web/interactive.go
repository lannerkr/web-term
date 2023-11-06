package web

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/dchest/uniuri"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/syssecfsu/witty/term_conn"
)

type InteractiveSession struct {
	Ip  string
	Cmd string
	Id  string
}

func collectSessions(c *gin.Context, cmd string) (players []InteractiveSession) {
	term_conn.ForEachSession(func(tc *term_conn.TermConn) {
		players = append(players, InteractiveSession{
			Id:  tc.Name,
			Ip:  tc.Ip,
			Cmd: cmd,
		})
	})

	return
}

func indexPage(c *gin.Context) {
	var result Group
	getConfig(&result)
	//fmt.Println(result)

	// pwdN := string(result.Groups[0].Sessions[0].Pwd)
	// if pwdN != "" {
	// 	decpwd, _ := decrypt(result.Groups[0].Sessions[0].Pwd)
	// 	fmt.Printf("decrypted pwd : %v\n", string(decpwd))
	// }

	var disabled = ""

	if options.NoAuth {
		disabled = "disabled"
	}

	c.HTML(http.StatusOK, "indexN.html",
		gin.H{
			"disabled":  disabled,
			"csrfField": csrf.TemplateField(c.Request),
			"csrfToken": csrf.Token(c.Request),
			"groups":    result.Groups,
			// "sessions":  result.Groups[0].Sessions,
			// "sessions2": result.Groups[1].Sessions,
			// "name0":     result.Sessions[0].Name,
		})
}

func updateIndex(c *gin.Context) {
	var active0, active1 string

	// setup which tab is active, it is hard to do in javascript at
	// client side due to timing issues.
	which := c.Param("active")
	if which == "0" {
		active0 = "active"
		active1 = ""
	} else {
		active0 = ""
		active1 = "active"
	}

	players := collectSessions(c, options.CmdToExec[0])
	// records := collectRecords(c)

	c.HTML(http.StatusOK, "tab.html", gin.H{
		"players": players,
		// "records": records,
		"active0": active0,
		"active1": active1,
	})
}

func newInteractive(c *gin.Context) {
	id := uniuri.New()
	if c.Param("id") != "" {
		id = c.Param("id") + ":" + uniuri.New()
	}

	//id = id+":"+pid

	c.HTML(http.StatusOK, "term.html", gin.H{
		"title":     "interactive terminal",
		"path":      "/ws_new/" + id,
		"id":        id,
		"logo":      "keyboard",
		"csrfToken": csrf.Token(c.Request),
	})
}

func newTermConn(c *gin.Context) {
	id := c.Param("id")
	if strings.Contains(id, ":") {
		str := strings.Split(id, ":")
		if str[1] == "newsessionupdate" {
			options.CmdToExec = []string{"ssh-copy-id"}
		}
		//fmt.Printf("newTermconn: %v\n", options.CmdToExec)
		updateCmd(c, str[0])

	} else {
		options.CmdToExec = []string{"sudo", "-H", "-u", "lannerkr", "bash"}
	}

	term_conn.ConnectTerm(c.Writer, c.Request, false, id, options.CmdToExec)
}
func updateCmd(c *gin.Context, id string) {

	var result Group
	getConfig(&result)
LOOP:
	for _, g := range result.Groups {
		for _, v := range g.Sessions {
			//fmt.Println(id)
			if id == v.Name {
				if v.Usepwd {
					fmt.Println("USE PWD is TRUE ...")
					newpwd, err := decrypt(v.Pwd)
					if err != nil {
						fmt.Println(err)
						goto NEXT
					}
					//fmt.Printf("decrypted pwd : %v\n", string(newpwd))
					options.CmdToExec = []string{"sudo", "-H", "-u", "lannerkr", "sshpass", "-p", string(newpwd), "ssh", "-l", v.User, "-p", v.Port, v.Host}
					break LOOP
				}
			NEXT:
				if len(options.CmdToExec) == 1 && options.CmdToExec[0] == "ssh-copy-id" {
					//fmt.Printf("cmdtoexec len : %v\n", len(options.CmdToExec))
					options.CmdToExec = []string{"sudo", "-H", "-u", "lannerkr", "ssh-copy-id", "-i", v.KeyPath, "-p", v.Port, v.User + "@" + v.Host}
					break LOOP
				}
				options.CmdToExec = []string{"sudo", "-H", "-u", "lannerkr", "ssh", "-l", v.User, "-p", v.Port, "-i", v.KeyPath, v.Host}
				break LOOP
			}
		}
	}
	//fmt.Printf("updateCmd: %v\n", options.CmdToExec)

	// if id == "oci" {
	// 	options.CmdToExec = []string{"sudo", "-H", "-u", "lannerkr", "ssh", "lannerkr@oci.physis.mooo.com"}
	// } else if id == "oca" {
	// 	options.CmdToExec = []string{"sudo", "-H", "-u", "lannerkr", "ssh", "lannerkr@oca.physis.mooo.com"}
	// } else if id == "docker" {
	// 	options.CmdToExec = []string{"sudo", "-H", "-u", "lannerkr", "ssh", "-p", "20022", "lannerkr@www.physis.kr"}
	// }
}

func viewPage(c *gin.Context) {
	id := c.Param("id")
	c.HTML(http.StatusOK, "term.html", gin.H{
		"title":     "viewer terminal",
		"path":      "/ws_view/" + id,
		"id":        id,
		"logo":      "view",
		"csrfToken": csrf.Token(c.Request),
	})
}

func newViewWS(c *gin.Context) {
	id := c.Param("id")
	term_conn.ConnectTerm(c.Writer, c.Request, true, id, nil)
}
