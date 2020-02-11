//                        Copyright (C) 2019 Rafael Santiago
//
// Use of this source is governed by GPL-v2 license that can
// be found in the COPYING file.
//
// Questions about the name? I really do not know... but
//
// "Googol - [Goo]gol is yet another [g]ame [o]f [l]ife"
//                                                 or still,
//
// "Googol - [G]ame [o]f life err...[o]h! in [go] [l]ang"...
//
// Sorry, it was silly. Anyway, it is just a well-simple program
// that outputs some generations of Conway's game of life as an
// animated GIF.
//
// For the sake of simplicity, trying to suckless, I have decided
// to concentrate all stuff in one single file. Thus, just
//
//              'go build googol.go' and
//              go ahead, away or home...
//
// You can also use my build system to automate some stuff such
// as compile and test on a pretty messy environment and also
// install googol:
//
//      - compile and build: 'hefesto'
//      - install: 'hefesto --install'
//
// Of course, you need to install Hefesto before. For doing it
// get its code and some instructions at:
//
//           <https://github.com/rafael-santiago/hefesto>
//
// Being 'googol' just a toy, 'shitty' program, I did not mind
// of writing tests for it. If you want to, go ahead.
//
// The best way of knowing what could really be done here is by
// reading 'doc/todo.txt'.
//
// Instructions about how to use googol can be found in
// 'README.md' or 'doc/MANUAL.txt'.
//
//                    -- Rafael / Thu 12 Dec 2019 06:33:55 AM AKST
//

package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"image"
	"image/color"
	"image/gif"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const googolVersion = "v1"
const gDefaultBoardWidth = "200"
const gDefaultBoardHeight = "200"
const gDefaultDelay = "50"
const gDefaultGenTotal = "10"
const gDefaultBkColor = "white"
const gDefaultFgColor = "black"
const gDefaultEndless = false
const gDefaultAddr = "localhost"
const gDefaultPort = "8080"
const gDefaultHttps = false

type GoogolRequest struct {
	Proto           string
	Addr            string
	Port            string
	BoardWidth      string
	BoardHeight     string
	GIFWidth        string
	GIFHeight       string
	Delay           string
	CellSizeInPx    string
	GenTotal        string
	BkColor         template.HTML
	SelectedBkColor color.Color
	FgColor         template.HTML
	SelectedFgColor color.Color
	Endless         string
	GIFData         string
	InitialState    []string
	Error           template.HTML
}

var gAvailColors = map[string]color.Color{"black": color.RGBA{0x00, 0x00, 0x00, 0xFF},
	"white":   color.RGBA{0xFF, 0xFF, 0xFF, 0xFF},
	"red":     color.RGBA{0xFF, 0x00, 0x00, 0xFF},
	"green":   color.RGBA{0x00, 0xFF, 0x00, 0xFF},
	"gray":    color.RGBA{0x80, 0x80, 0x80, 0xFF},
	"blue":    color.RGBA{0x00, 0x00, 0xFF, 0xFF},
	"cyan":    color.RGBA{0x00, 0xFF, 0xFF, 0xFF},
	"yellow":  color.RGBA{0xFF, 0xFF, 0x00, 0xFF},
	"magenta": color.RGBA{0xFF, 0x00, 0xFF, 0xFF}}

var gAvailCommands = map[string]func() int{"gif": dumpGIF,
	"httpd": httpdGIFdumper,
	"help":  help,
	"version": func() int {
		fmt.Fprintf(os.Stdout, "googol-%s\n", googolVersion)
		return 0
	}}

var gAvailCommandHelpers = map[string]func() int{"gif": helpGIF,
	"httpd": helpHttpd,
	"version": func() int {
		fmt.Fprintf(os.Stdout, "usage: googol version\n")
		return 0
	}}

// INFO(Rafael): This is a little bit clumsy but it does the job.

var setField = func(strPtr *string, data interface{}) {
	switch data.(type) {
	case string:
		*strPtr = data.(string)
	case []string:
		*strPtr = data.([]string)[0]
	default:
		fmt.Fprintf(os.Stderr, "ERROR: Data %v not supported by setString().\n", data)
	}
}

var setCheckboxState = func(data interface{}) string {
	var state string
	switch data.(type) {
	case bool:
		if data.(bool) {
			state = "checked"
		}
	case string:
		if data.(string) == "1" {
			state = "checked"
		}
	case []string:
		if data.([]string)[0] == "1" {
			state = "checked"
		}
	default:
		fmt.Fprintf(os.Stderr, "ERROR: Data %v not supported by setBool().\n", data)
	}
	return state
}

var getColorOption = func(data interface{}) (template.HTML, color.Color) {
	var selColor string
	switch data.(type) {
	case string:
		selColor = data.(string)
	case []string:
		selColor = data.([]string)[0]
	}
	colors := reflect.ValueOf(gAvailColors).MapKeys()
	colorsLen := len(colors) + 1
	colorList := make([]string, colorsLen)
	for c := 0; c < colorsLen - 1; c++ {
		colorList[c] = colors[c].String()
	}
        colorList[colorsLen - 1] = "random"
	sort.Strings(colorList)
	var strData string
	for _, c := range colorList {
		if c != selColor {
			strData += "<option value=\"" + c + "\">" + c + "</option>\n"
		} else {
			strData += "<option value=\"" + c + "\" selected>" + c + "</option>\n"
		}
	}
	return template.HTML(strData), getColor(selColor)
}

var getInitialState = func(data interface{}) []string {
	var state []string
	var dataList []string
	switch data.(type) {
	case string:
		dataList = strings.Split(data.(string), " ")
	case []string:
		if len(data.([]string)) > 1 {
			dataList = data.([]string)
		} else {
			dataList = strings.Split(data.([]string)[0], " ")
		}
	}
	for _, s := range dataList {
		matches, _ := regexp.MatchString(`--[0-9]+,[0-9]+\.`, s)
		if matches {
			state = append(state, s)
		}
	}
	return state
}

var gFieldsFiller = map[string]func(*GoogolRequest, interface{}){
	"Addr":         nil,
	"Port":         nil,
	"Proto":        nil,
	"InitialState": func(req *GoogolRequest, data interface{}) { req.InitialState = getInitialState(data) },
	"BoardWidth":   func(req *GoogolRequest, data interface{}) { setField(&req.BoardWidth, data) },
	"BoardHeight":  func(req *GoogolRequest, data interface{}) { setField(&req.BoardHeight, data) },
	"GIFWidth":     func(req *GoogolRequest, data interface{}) { setField(&req.GIFWidth, data) },
	"GIFHeight":    func(req *GoogolRequest, data interface{}) { setField(&req.GIFHeight, data) },
	"Delay":        func(req *GoogolRequest, data interface{}) { setField(&req.Delay, data) },
	"CellSizeInPx": func(req *GoogolRequest, data interface{}) { setField(&req.CellSizeInPx, data) },
	"GenTotal":     func(req *GoogolRequest, data interface{}) { setField(&req.GenTotal, data) },
	"BkColor":      func(req *GoogolRequest, data interface{}) { req.BkColor, req.SelectedBkColor = getColorOption(data) },
	"FgColor":      func(req *GoogolRequest, data interface{}) { req.FgColor, req.SelectedFgColor = getColorOption(data) },
	"Endless":      func(req *GoogolRequest, data interface{}) { req.Endless = setCheckboxState(data) }}

var gDefaultFields = map[string]func(*GoogolRequest){
	"Addr": func(req *GoogolRequest) { req.Addr = getOption("addr", "localhost") },
	"Port": func(req *GoogolRequest) {
		var port string
		if !getBoolOption("https", gDefaultHttps) {
			port = gDefaultPort
		} else {
			port = "443"
		}
		req.Port = getOption("port", port)
	},
	"Proto": func(req *GoogolRequest) {
		if !getBoolOption("https", gDefaultHttps) {
			req.Proto = "http"
		} else {
			req.Proto = "https"
		}
	},
	"InitialState": func(req *GoogolRequest) { req.InitialState = getInitialState(os.Args[2:]) },
	"BoardWidth":   func(req *GoogolRequest) { req.BoardWidth = getOption("board-width", gDefaultBoardWidth) },
	"BoardHeight":  func(req *GoogolRequest) { req.BoardHeight = getOption("board-height", gDefaultBoardHeight) },
	"GIFWidth":     func(req *GoogolRequest) { req.GIFWidth = getOption("gif-width", gDefaultBoardWidth) },
	"GIFHeight":    func(req *GoogolRequest) { req.GIFHeight = getOption("gif-height", gDefaultBoardHeight) },
	"Delay":        func(req *GoogolRequest) { req.Delay = getOption("delay", gDefaultDelay) },
	"CellSizeInPx": func(req *GoogolRequest) { req.CellSizeInPx = getOption("cell-size-in-px", "1") },
	"GenTotal":     func(req *GoogolRequest) { req.GenTotal = getOption("gen-total", gDefaultGenTotal) },
	"BkColor": func(req *GoogolRequest) {
		req.BkColor, req.SelectedBkColor = getColorOption(getOption("bk-color", gDefaultBkColor))
	},
	"FgColor": func(req *GoogolRequest) {
		req.FgColor, req.SelectedFgColor = getColorOption(getOption("fg-color", gDefaultFgColor))
	},
	"Endless": func(req *GoogolRequest) { req.Endless = setCheckboxState(getBoolOption("endless", gDefaultEndless)) }}

var gMaxBoardWidth int = 500

var gMaxBoardHeight int = 500

var gFormTemplate string = `
<html>
    <title>Googol webserver</title>
    <h1>Googol webserver</h1>
    <table border=0>
        <tr bgcolor="black">
            <td align="right"><font color="white" size="2"><b>Game parameters</b></font></td>
        </tr>
        <tr>
            <td>
                <form method="post" action="{{.Proto}}://{{.Addr}}:{{.Port}}/googol">
                    <table border=0>
                        <tr>
                            <td><b>Initial state</b>:</td>
                            <td><input type="text" name="InitialState" style="text-align:right;width:430px" value="{{range .InitialState}}{{.}} {{end}}"></td>
                        </tr>
                        <tr>
                            <td><b>Board width</b>:</td>
                            <td><input type="number" name="BoardWidth" style="text-align:right;width:430px" value="{{.BoardWidth}}"></td>
                        </tr>
                        <tr>
                            <td><b>Board height</b>:</td>
                            <td><input type="number" name="BoardHeight" style="text-align:right;width:430px" value="{{.BoardHeight}}"></td>
                        </tr>
                        <tr>
                            <td><b>GIF width</b>:</td>
                            <td><input type="number" name="GIFWidth" style="text-align:right;width:430px" value="{{.GIFWidth}}"></td>
                        </tr>
                        <tr>
                            <td><b>GIF height</b>:</td>
                            <td><input type="number" name="GIFHeight" style="text-align:right;width:430px" value="{{.GIFHeight}}"></td>
                        </tr>
                        <tr>
                            <td><b>Delay</b>:</td>
                            <td><input type="number" name="Delay" style="text-align:right;width:430px" value="{{.Delay}}"></td>
                        </tr>
                        <tr>
                            <td><b>Cell size in pixels</b>:</td>
                            <td><input type="number" name="CellSizeInPx" style="text-align:right;width:430px" size=50 value="{{.CellSizeInPx}}"></td>
                        </tr>
                        <tr>
                            <td><b>Generation total</b>:</td>
                            <td><input type="number" name="GenTotal" style="text-align:right;width:430px" size=50 value="{{.GenTotal}}"></td>
                        </tr>
                        <tr>
                            <td><b>Background color</b>:</td>
                            <td>
                                <select name="BkColor" style="width:430px;text-align:right">
                                    {{.BkColor}}
                                </select>
                            </td>
                        </tr>
                        <tr>
                            <td><b>Foreground color</b>:</td>
                            <td>
                                <select name="FgColor" style="width:430px;text-align:right">
                                    {{.FgColor}}
                                </select>
                            </td>
                        </tr>
                        <tr>
                            <td><input type="checkbox" name="Endless" value="1" {{.Endless}}>
                            <b>Endless animation</b></td>
                            <td><input type="submit" style="width:430px" value="Generate"></td>
                        </tr>
                    </table>
                </form>
            </td>
        </tr>
    </table>
    <div style="background-color:red">
        <center><b>{{.Error}}</b></center>
    </div>
    <div>
        <center>
            <img src="data:image/gif;base64,{{.GIFData}}" alt=":("/>
        </center>
    </div>
    <footer>
        <p><small>Googol is Copyright (C) 2019 by Rafael Santiago<br>
         Issues: <a href="https://github.com/rafael-santiago/googol/issues" target=_vblank>https://github.com/rafael-santiago/googol/issues</a><br>
         Contact: Yours nearest /dev/null</small>
    </footer>
</html>
`

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	var exitCode int = 0
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stdout, "usage: googol <command>\n")
	} else if command, ok := gAvailCommands[os.Args[1]]; ok {
		exitCode = command()
	} else {
		fmt.Fprintf(os.Stderr, "ERROR: What are you intending to do?\n")
		exitCode = 1
	}
	os.Exit(exitCode)
}

func help() int {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stdout, "usage: googol help <command>\n\n")
		fmt.Fprintf(os.Stdout, "Commands:\n\n")
		for command, _ := range gAvailCommandHelpers {
			fmt.Fprintf(os.Stdout, "\t* %s\n", command)
		}
		fmt.Fprintf(os.Stdout, "______\nGoogol is Copyright (C) 2019 by Rafael Santiago.\n"+
			"Bug reports, feedback, etc: <voidbrainvoid@tutanota.com> "+
			"or <https://github.com/rafael-santiago/googol/issues>\n")
		return 0
	}
	if helper, ok := gAvailCommandHelpers[os.Args[2]]; ok {
		return helper()
	}
	fmt.Fprintf(os.Stderr, "No help entry for %s.\n", os.Args[2])
	return 1
}

func helpGIF() int {
	fmt.Fprintf(os.Stdout, "usage: googol gif [--board-with=<n> --board-height=<n> --gif-with=<n>\n"+
		"                   --gif-height=<n> --delay=<n> --cell-size-in-px=<n>\n"+
		"                   --gen-total=<n> --bk-color=<color> --fg-color=<color>\n"+
		"                   --endless] --out=<file-path> [initial-board-state]\n\n"+
		"                  or\n\n"+
		"       googol gif [--board-with=<n> --board-height=<n> --gif-with=<n>\n"+
		"                   --gif-height=<n> --delay=<n> --cell-size-in-px=<n>\n"+
		"                   --gen-total=<n> --bk-color=<color> --fg-color=<color>\n"+
		"                   --endless] > <file-path> [initial-board-state]\n"+
		"Defaults:\n\n"+
		"\t* --board-width = %s\n"+
		"\t* --board-height = %s\n"+
		"\t* --gif-width = --board-width\n"+
		"\t* --gif-height = --board-height\n"+
		"\t* --delay = %sms\n"+
		"\t* --cell-size-in-px = --gif-width / 8\n"+
		"\t* --gen-total = %s\n"+
		"\t* --out = stdout\n"+
		"\t* --bk-color = %s\n"+
		"\t* --fg-color = %s\n"+
		"\t* --endless = false\n"+
		"Notes:\n\n"+
		"\t* The file path passed through --out is overwritten without\n"+
		"\t  any prompt.\n"+
		"\t* The available colors are: 'black', 'blue', 'cyan', 'gray'\n"+
		"\t  'green', 'magenta', 'red', 'white', 'yellow'. You can also\n"+
		"\t  use 'random' or 'any' instead.\n"+
		"\t* [initial-board-state] stands for a list of options in form\n"+
		"\t  '--<n>,<n>.', where <n>,<n> are the coordinates (x,y) of an\n"+
		"\t  alive cell.\n", gDefaultBoardWidth, gDefaultBoardHeight, gDefaultDelay, gDefaultGenTotal,
		gDefaultBkColor, gDefaultFgColor)
	return 0
}

func helpHttpd() int {
	fmt.Fprintf(os.Stdout, "usage: googol httpd [--port=<n> --addr=<address> --https\n"+
		"                     --server-crt=<file-path> --server-key=<file-path>\n"+
		"                     --form-template=<file-path>]\n"+
		"Defaults:\n\n"+
		"\t* --port = %s\n"+
		"\t* --addr = %s\n"+
		"\t* --https = false\n"+
		"\t* --server-crt = <empty>\n"+
		"\t* --server-key = <empty>\n"+
		"\t* --form-template = (some lousy default HTML)\n"+
		"\t* --max-board-width = %d\n"+
		"\t* --max-board-height = %d\n"+
		"Notes:\n\n"+
		"\t* When https is requested the default port is 443.\n"+
		"\t* In order to gracefully stop the server send to the process\n"+
		"\t  a SIGINT or SIGTERM. Note that a SIGINT is equivalent to a\n"+
		"\t  'CTRL + c'.\n"+
		"\t* The defaults for the game and gifs are the same of the 'gif'\n"+
		"\t  command.\n"+
		"\t* If you want to set new defaults for the game or gifs\n"+
		"\t  use the same options available in 'gif' command.\n", gDefaultPort, gDefaultAddr, gMaxBoardWidth,
		gMaxBoardHeight)
	return 0
}

func httpdGIFdumper() int {
	http.HandleFunc("/googol", httpdHandler)
	var err error
	gMaxBoardWidth, err = strconv.Atoi(getOption("max-board-width", fmt.Sprintf("%d", gMaxBoardWidth)))
	if err != nil || gMaxBoardWidth <= 0 {
		fmt.Fprintf(os.Stderr, "ERROR: option --max-board-width must be a valid positive integer.\n")
		os.Exit(1)
	}
	gMaxBoardHeight, err = strconv.Atoi(getOption("max-board-height", fmt.Sprintf("%d", gMaxBoardHeight)))
	if err != nil || gMaxBoardHeight <= 0 {
		fmt.Fprintf(os.Stderr, "ERROR: option --max-board-height must be a valid positive integer.\n")
		os.Exit(1)
	}
	var googol func()
	if !getBoolOption("https", false) {
		googol = func() {
			err = http.ListenAndServe(getOption("addr", gDefaultAddr)+":"+getOption("port", gDefaultPort), nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
				os.Exit(1)
			}
		}
	} else {
		serverCRT := getOption("server-crt", "")
		if len(serverCRT) == 0 {
			fmt.Fprintf(os.Stderr, "ERROR: option --server-crt must point to a valid certificate file.\n")
			return 1
		}
		serverKey := getOption("server-key", "")
		if len(serverKey) == 0 {
			fmt.Fprintf(os.Stderr, "ERROR: option --server-key must point to a valid private key file.\n")
			return 1
		}
		googol = func() {
			err = http.ListenAndServeTLS(getOption("addr", gDefaultAddr)+":"+getOption("port", "443"), serverCRT, serverKey, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
				os.Exit(1)
			}
		}
	}
	formTemplate := getOption("form-template", "")
	if len(formTemplate) > 0 {
		templateFile, err := os.Open(formTemplate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Unable to open form template: %v.\n", err)
			os.Exit(1)
		}
		defer templateFile.Close()
		buf, err := ioutil.ReadAll(templateFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Unable to read data from form template: %v.\n", err)
			os.Exit(1)
		}
		gFormTemplate = string(buf)
	}
	go googol() // OoOOoOoOOOL...
	sigintWatchdog := make(chan os.Signal, 1)
	signal.Notify(sigintWatchdog, os.Interrupt)
	signal.Notify(sigintWatchdog, syscall.SIGINT|syscall.SIGTERM)
	<-sigintWatchdog
	fmt.Fprintf(os.Stdout, "\nINFO: googol httpd finished.\n")
	return 0
}

func httpdHandler(w http.ResponseWriter, r *http.Request) {
	responseTemplate := template.Must(template.New("escape").Parse(gFormTemplate))
	userData := newGoogolRequest(r)
	var boardWidth, boardHeight, gifWidth, gifHeight, delay, cellSizeInPx, genNr int
	var err error
	boardWidth, err = strconv.Atoi(userData.BoardWidth)
	if err != nil || boardWidth <= 0 || boardWidth > gMaxBoardWidth {
		userData.Error = template.HTML(fmt.Sprintf("ERROR: Board width must be a valid positive "+
			"integer between 1 and %d.",
			gMaxBoardWidth))
		responseTemplate.Execute(w, userData)
		return
	}
	boardHeight, err = strconv.Atoi(userData.BoardHeight)
	if err != nil || boardHeight <= 0 || boardHeight > gMaxBoardHeight {
		userData.Error = template.HTML(fmt.Sprintf("ERROR: Board height must be a valid positive "+
			"integer between 1 and %d.",
			gMaxBoardHeight))
		responseTemplate.Execute(w, userData)
		return
	}
	gifWidth, err = strconv.Atoi(userData.GIFWidth)
	if err != nil || gifWidth <= 0 {
		userData.Error = "ERROR: GIF width must be a valid positive integer."
		responseTemplate.Execute(w, userData)
		return
	}
	gifHeight, err = strconv.Atoi(userData.GIFHeight)
	if err != nil || gifHeight <= 0 {
		userData.Error = "ERROR: GIF height must be a valid positive integer."
		responseTemplate.Execute(w, userData)
		return
	}
	delay, err = strconv.Atoi(userData.Delay)
	if err != nil || delay <= 0 {
		userData.Error = "ERROR: Delay must be a valid positive integer."
		responseTemplate.Execute(w, userData)
		return
	}
	cellSizeInPx, err = strconv.Atoi(userData.CellSizeInPx)
	if err != nil || cellSizeInPx <= 0 || cellSizeInPx > 200 {
		userData.Error = "ERROR: Cell size in pixels must be a valid positive integer less than 200."
		responseTemplate.Execute(w, userData)
		return
	}
	genNr, err = strconv.Atoi(userData.GenTotal)
	if err != nil || genNr <= 0 {
		userData.Error = "ERROR: Generation total must be a valid positive interger."
		responseTemplate.Execute(w, userData)
		return
	}
	cells := makeGameBoard(boardWidth, boardHeight)
	setBigBangGeneration(cells, userData.InitialState)
	gifBuf := bytes.NewBufferString("")
	makeGIFofLife(gifBuf, userData.SelectedBkColor, userData.SelectedFgColor, gifWidth, gifHeight, delay, userData.Endless == "checked",
		cellSizeInPx, cells, genNr)
	userData.GIFData = base64.StdEncoding.EncodeToString(gifBuf.Bytes())
	responseTemplate.Execute(w, userData)
}

func newGoogolRequest(r *http.Request) GoogolRequest {
	var usrData GoogolRequest
	if err := r.ParseForm(); err != nil {
		return GoogolRequest{}
	}
	for field, data := range r.Form {
		if gFieldsFiller[field] != nil {
			gFieldsFiller[field](&usrData, data)
		}
	}
	for field, setDefault := range gDefaultFields {
		if setDefault == nil {
			continue
		}
		if _, ok := r.Form[field]; !ok {
			setDefault(&usrData)
		}
	}
	return usrData
}

func dumpGIF() int {
	var err error
	xNr, err := strconv.Atoi(getOption("board-width", gDefaultBoardWidth))
	if err != nil || xNr < 0 {
		fmt.Fprintf(os.Stderr, "ERROR: option board-width must be a valid positive integer.\n")
		return 1
	}
	yNr, err := strconv.Atoi(getOption("board-height", gDefaultBoardHeight))
	if err != nil || yNr < 0 {
		fmt.Fprintf(os.Stderr, "ERROR: option board-heigh must be a valid positive integer.\n")
		return 1
	}
	gifWidth, err := strconv.Atoi(getOption("gif-width", fmt.Sprintf("%d", xNr)))
	if err != nil || gifWidth < 0 {
		fmt.Fprintf(os.Stderr, "ERROR: option gif-width must be a valid positive integer.\n")
		return 1
	}
	gifHeight, err := strconv.Atoi(getOption("gif-height", fmt.Sprintf("%d", yNr)))
	if err != nil || gifHeight < 0 {
		fmt.Fprintf(os.Stderr, "ERROR: option gif-height must be a valid positive integer.\n")
		return 1
	}
	delay, err := strconv.Atoi(getOption("delay", gDefaultDelay))
	if err != nil || delay <= 0 {
		fmt.Fprintf(os.Stderr, "ERROR: option delay must be a valid positive integer (frames in 'delay'ms).\n")
		return 1
	}
	cellSizeInPixels, err := strconv.Atoi(getOption("cell-size-in-px", fmt.Sprintf("%d", gifWidth>>3)))
	if err != nil || cellSizeInPixels <= 0 {
		fmt.Fprintf(os.Stderr, "ERROR: option cell-size-in-px must be a valid positive integer.\n")
		return 1
	}
	generationNr, err := strconv.Atoi(getOption("gen-total", gDefaultGenTotal))
	if err != nil || generationNr <= 0 {
		fmt.Fprintf(os.Stderr, "ERROR: option gen-total must be a valid positive integer.\n")
		return 1
	}
	cells := makeGameBoard(xNr, yNr)
	setBigBangGeneration(cells, os.Args[2:])
	makeGIFofLife(getOutput(),
		getColor(getOption("bk-color", gDefaultBkColor)),
		getColor(getOption("fg-color", gDefaultFgColor)),
		gifWidth, gifHeight,
		delay,
		getBoolOption("endless", gDefaultEndless),
		cellSizeInPixels, cells, generationNr)
	return 0
}

func drawAliveCell(frame *image.Paletted, x, y, cellSizeInPixels, fgColorIndex int) {
	for xtemp := 0; xtemp < cellSizeInPixels; xtemp++ {
		for ytemp := 0; ytemp < cellSizeInPixels; ytemp++ {
			frame.SetColorIndex(x+xtemp, y+ytemp, uint8(fgColorIndex))
		}
	}
}

func getOutput() io.Writer {
	out := getOption("out", "")
	if len(out) == 0 {
		return os.Stdout
	}
	f, err := os.Create(out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v.\n", err)
		os.Exit(1)
	}
	return f
}

func makeGIFofLife(out io.Writer,
	bkColor, fgColor color.Color,
	width, height,
	delay int,
	endless bool,
	cellSizeInPixels int,
	cells [][]byte, generationNr int) {
	var gifImage gif.GIF
	if endless {
		gifImage = gif.GIF{LoopCount: 0}
	} else {
		gifImage = gif.GIF{LoopCount: 1}
	}
	xNr := len(cells)
	yNr := len(cells[0])
	for g := 0; g < generationNr; g++ {
		frame := image.NewPaletted(image.Rect(0, 0, width, height), []color.Color{bkColor, fgColor})
		xFrame := 0
		for x := 0; x < xNr; x++ {
			yFrame := 0
			for y := 0; y < yNr; y++ {
				if (cells[x][y] & 0x1) == 1 {
					drawAliveCell(frame, xFrame, yFrame, cellSizeInPixels, 1)
				}
				yFrame += cellSizeInPixels
			}
			xFrame += cellSizeInPixels
		}
		gifImage.Delay = append(gifImage.Delay, delay)
		gifImage.Image = append(gifImage.Image, frame)
		getNextGeneration(cells)
	}
	gif.EncodeAll(out, &gifImage)
}

func makeGameBoard(xNr, yNr int) [][]byte {
	var cells [][]byte
	cells = make([][]byte, yNr)
	for y := 0; y < yNr; y++ {
		cells[y] = make([]byte, xNr)
	}
	return cells
}

func getCellCoords(str string) (int, int) {
	x := -1
	y := -1
	if strings.HasPrefix(str, "--") && strings.HasSuffix(str, ".") {
		coords := strings.Split(str[2:len(str)-1], ",")
		if len(coords) == 2 {
			var err error
			x, err = strconv.Atoi(coords[0])
			if err == nil {
				y, err = strconv.Atoi(coords[1])
			}
		}
	}
	return x, y
}

func setBigBangGeneration(cells [][]byte, args []string) {
	xNr := len(cells)
	yNr := len(cells[0])
	for _, a := range args {
		x, y := getCellCoords(a)
		if x > -1 && y > -1 && x < xNr && y < yNr {
			cells[x][y] = 1
		}
	}
}

func countAliveNeighboursIter(cells [][]byte, x, y, xNr, yNr int) int {
	if x < 0 || y < 0 || x >= xNr || y >= yNr {
		return 0
	}
	return int(cells[x][y] & 0x1)
}

func countAliveNeighbours(cells [][]byte, x, y int) int {
	xNr := len(cells)
	yNr := len(cells[0])
	return countAliveNeighboursIter(cells, x-1, y-1, xNr, yNr) +
		countAliveNeighboursIter(cells, x, y-1, xNr, yNr) +
		countAliveNeighboursIter(cells, x+1, y-1, xNr, yNr) +
		countAliveNeighboursIter(cells, x-1, y, xNr, yNr) +
		countAliveNeighboursIter(cells, x+1, y, xNr, yNr) +
		countAliveNeighboursIter(cells, x-1, y+1, xNr, yNr) +
		countAliveNeighboursIter(cells, x, y+1, xNr, yNr) +
		countAliveNeighboursIter(cells, x+1, y+1, xNr, yNr)
}

func getNextGeneration(cells [][]byte) {
	xNr := len(cells)
	yNr := len(cells[0])
	for x := 0; x < xNr; x++ {
		for y := 0; y < yNr; y++ {
			// INFO(Rafael): Just because I am a lazy person...
			aliveNeighboursNr := countAliveNeighbours(cells, x, y)
			itWillLiveOrReproduct := ((cells[x][y]&1) == 1 && (aliveNeighboursNr == 2 || aliveNeighboursNr == 3)) ||
				((cells[x][y]&1) == 0 && (aliveNeighboursNr == 3))
			if itWillLiveOrReproduct {
				cells[x][y] |= 0x2 // ...see?!
			}
		}
	}
	for x := 0; x < xNr; x++ {
		for y := 0; y < yNr; y++ {
			cells[x][y] >>= 1 // Zzz...
		}
	}
}

func getColor(colorName string) color.Color {
	if colorName == "any" || colorName == "random" {
		var r, g, b uint8
		for l := 0; l <= rand.Int()%((rand.Int()%10)+1); l++ {
			r = (uint8(rand.Int()) + g)
			g = (uint8(rand.Int()) + r + b)
			b = (uint8(rand.Int()) + r + g)
		}
		return color.RGBA{r, g, b, 0xFF}
	}
	if cl, ok := gAvailColors[colorName]; ok {
		return cl
	}
	return color.Black
}

func getOption(option, defaultValue string) string {
	optionLabel := "--" + option + "="
	for _, o := range os.Args[2:] {
		if strings.HasPrefix(o, optionLabel) {
			return o[len(option)+3:]
		}
	}
	return defaultValue
}

func getBoolOption(option string, defaultValue bool) bool {
	optionLabel := "--" + option
	for _, o := range os.Args[2:] {
		if o == optionLabel {
			return true
		}
	}
	return defaultValue
}
