package main

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/wayneashleyberry/terminal-dimensions"
	"strings"
	"os"
	"path/filepath"
	"time"
	"os/user"
)

/*
-[heilmanc@laptop]----------------------------------------------------------------------------------------{~/.cahhome}-
-- git:<master>                                                                                                                                                                                                   --
-12:50<####.>~16~%
*/

var DEFAULT = color.New(color.FgGreen)
var RESET = color.New(color.Reset).Sprint()
var SPACER = DEFAULT.Sprint("-")
var LSQBRACKET = DEFAULT.Sprint("[")
var RSQBRACKET = DEFAULT.Sprint("]")
var LBRACE = DEFAULT.Sprint("{")
var RBRACE = DEFAULT.Sprint("}")
var LANBRACKET = DEFAULT.Sprint("<")
var RANBRACKET = DEFAULT.Sprint(">")

var WIDTH int
var FIRST_LINE_WIDTH_AVAILABLE int
var SECOND_LINE_WIDTH_AVAILABLE int

func username() (string, string) {
	curUser, userErr := user.Current()
	if userErr != nil {
		return "!user!", color.HiRedString("!user!")
	} else {
		userName := curUser.Username

		if userName == "root" {
			return userName, color.HiYellowString(userName)
		} else {
			return userName, color.CyanString(userName)
		}
	}
}

func atjobs() (string, string) {
	// TODO
	return "@", color.CyanString("@") + RESET
}

func hostload() (string, string) {

	// Get hostname

	hostName, hostErr := os.Hostname()
	if hostErr != nil {
		hostName = "!host!"
	}

	prettyName, _, prettyNameErr := execAndGetOutput("pretty-hostname", nil)

	if prettyNameErr == nil {
		hostName = prettyName
	}

	// Figure out load color
	loadColor := color.New(color.FgCyan)

	return hostName, loadColor.Sprint(hostName) + RESET
}

func cwd(dirWidthAvailable int) (string, string) {
	fullPath, err := os.Getwd()
	if err != nil {
		return "!dir!", color.HiRedString("!dir!")
	} else {
		var homePath = fullPath

		// Match the path to "HOME"
		var HOME = os.ExpandEnv("$HOME")
		var CANONHOME = normalizePath(HOME)

		if strings.HasPrefix(fullPath, HOME) {
			relative, relErr := filepath.Rel(HOME, fullPath)

			if relErr == nil {
				homePath = filepath.Join("~", relative)
			}
		} else if strings.HasPrefix(fullPath, CANONHOME) {
			relative, relErr := filepath.Rel(CANONHOME, fullPath)

			if relErr == nil {
				homePath = filepath.Join("~", relative)
			}
		}

		// Truncate to the space available
		homePath = truncateAndEllipsisAtStart(homePath, dirWidthAvailable)

		// Look up space left on that path, writability to get color
		dirColor := color.New(color.FgHiGreen, color.Bold)

		// Return
		return homePath, dirColor.Sprint(homePath) + RESET
	}
}

func curtime() (string, string) {
	t := time.Now().Local().Format("15:04")
	return t, color.YellowString(t) + RESET
}

func battery() (string, string) {
	battInfo, err := NewBatteryInfo()

	if err != nil {
		return "<!bat!>", color.HiRedString("<!bat!>")
	} else {
		if battInfo.Percent > 99 {
			// Display nothing
			return "<>", DEFAULT.Sprint("<>")
		} else if battInfo.Percent > 20 {
			// Display bars
			return "<" + battInfo.Gauge + ">",
				LANBRACKET + battInfo.ColorizedGauge + RANBRACKET
		} else {
			// Display time left
			return "<" + fmt.Sprintf("%0d:%02d", battInfo.TimeLeft.Hours(), battInfo.TimeLeft.Minutes()) + ">",
				LANBRACKET + battInfo.ColorizedTimeLeft + RANBRACKET
		}
	}
}

func gitBranch() (string, string) {
	// TODO
	git := "git:<master>"
	return git, color.HiGreenString(git) + RESET
}

func gitFiles() (string, string) {
	// TODO
	files := "M:2 +:1"
	return files, DEFAULT.Sprint(files) + RESET
}

func getWidth() int {
	w, err := terminaldimensions.Width()

	if err != nil {
		// Guess
		return 100
	} else {
		return int(w)
	}
}

func main() {
	WIDTH = getWidth()

	//////////////////
	// FIRST LINE
	//////////////////

	FIRST_LINE_WIDTH_AVAILABLE = WIDTH

	fmt.Print(SPACER + LSQBRACKET)
	FIRST_LINE_WIDTH_AVAILABLE -= 2

	user, userColor := username()
	fmt.Print(userColor)
	FIRST_LINE_WIDTH_AVAILABLE -= len(user)

	jobs, jobsColor := atjobs()
	fmt.Print(jobsColor)
	FIRST_LINE_WIDTH_AVAILABLE -= len(jobs)

	host, hostColor := hostload()
	fmt.Print(hostColor)
	FIRST_LINE_WIDTH_AVAILABLE -= len(host)

	fmt.Print(RSQBRACKET + SPACER)
	FIRST_LINE_WIDTH_AVAILABLE -= 2

	dir, dirColor := cwd(FIRST_LINE_WIDTH_AVAILABLE - (1 + 2 + 2))

	spacersRequired := WIDTH - (2 + len(user) + len(jobs) + len(host) + 2 + 2 + len(dir) + 2)
	if spacersRequired < 1 {
		spacersRequired = 1
	}
	firstLineDynamicSpace := strings.Repeat(SPACER, spacersRequired)

	fmt.Print(firstLineDynamicSpace)
	FIRST_LINE_WIDTH_AVAILABLE -= spacersRequired

	fmt.Print(SPACER + LBRACE)
	FIRST_LINE_WIDTH_AVAILABLE -= 2

	fmt.Print(dirColor)
	FIRST_LINE_WIDTH_AVAILABLE -= len(dir)

	fmt.Print(RBRACE + SPACER)
	FIRST_LINE_WIDTH_AVAILABLE -= 2

	fmt.Println()
	fmt.Printf("left: %d\n", FIRST_LINE_WIDTH_AVAILABLE)

	//////////////////
	// SECOND LINE
	//////////////////

	SECOND_LINE_WIDTH_AVAILABLE = WIDTH

	tme, tmeColor := curtime()
	batt, battColor := battery()
	branch, branchColor := gitBranch()
	files, filesColor := gitFiles()

	fmt.Printf("%s%s%s%s %s%s%s %s\n",
		SPACER+SPACER,
		tmeColor,
		battColor,
		SPACER+SPACER,
		branchColor,
		"    ",
		filesColor,
		SPACER+SPACER)

	fmt.Println(tme, batt, branch, files)

}
