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

	hostName = strings.TrimSpace(hostName)

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
		dirColor := color.New(color.FgHiGreen)

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

func getErrorCode() (string, string) {
	// TODO: This doesn't work
	err := os.ExpandEnv("$?")
	return err, err
}

func getKerberos() (string, string) {
	// See if we even care (flag in host config)
	if !fileExists(os.ExpandEnv("$HOME/.host/config/ignore_kerberos")) {
		// Do we have a ticket?
		_, exitCode, _ := execAndGetOutput("klist", nil, "-s")

		hasTicket := exitCode == 0

		if hasTicket {
			return "", ""
		} else {
			return "[K]", color.New(color.FgHiRed, color.Bold).Sprint("[K]")
		}
	} else {
		return "", ""
	}
}

func gitBranch(info *RepoInfo) (string, string) {
	gitColor := color.New(color.FgHiCyan)

	if info.BranchName == "master" || info.BranchName == "mainline" {
		gitColor = color.New(color.FgHiGreen)
	}

	branchLine := gitColor.Sprint("   git:<") + info.BranchNameColored + gitColor.Sprint(">")

	if len(info.OtherBranches) > 0 {
		branchLine += " " + color.WhiteString("{" + strings.Join(info.OtherBranches, ", ") + "}")
	}

	return stripANSI(branchLine), branchLine
}

func gitFiles(info *RepoInfo) (string, string) {
	return info.Status, info.StatusColored
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

	//////////////////
	// SECOND LINE
	//////////////////

	SECOND_LINE_WIDTH_AVAILABLE = WIDTH

	fmt.Print(SPACER+SPACER)
	SECOND_LINE_WIDTH_AVAILABLE -= 2

	tme, tmeColor := curtime()
	fmt.Print(tmeColor)
	SECOND_LINE_WIDTH_AVAILABLE -= len(tme)

	batt, battColor := battery()
	if len(batt) > 0 {
		fmt.Print(battColor)
		SECOND_LINE_WIDTH_AVAILABLE -= len(batt)
	}

	errCode, errCodeColor := getErrorCode()
	if len(errCode) > 0 {
		fmt.Print(errCodeColor)
		SECOND_LINE_WIDTH_AVAILABLE -= len(errCode)
	}

	kerberos, kerberosColor := getKerberos()

	if len(kerberos) > 0 {
		fmt.Print(kerberosColor)
		SECOND_LINE_WIDTH_AVAILABLE -= len(kerberos)
	}

	gitInfo := NewRepoInfo()

	if gitInfo.IsRepo {
		branch, branchColor := gitBranch(gitInfo)
		files, filesColor := gitFiles(gitInfo)

		fmt.Print(branchColor)
		SECOND_LINE_WIDTH_AVAILABLE -= len(branch)

		spacersRequired := SECOND_LINE_WIDTH_AVAILABLE - (len(files) + 3)
		if spacersRequired < 1 {
			spacersRequired = 1
		}
		secondLineDynamicSpace := strings.Repeat(" ", spacersRequired)

		fmt.Print(secondLineDynamicSpace)
		SECOND_LINE_WIDTH_AVAILABLE -= spacersRequired

		fmt.Print(filesColor)
	}

	fmt.Print(" " + SPACER + SPACER)
	SECOND_LINE_WIDTH_AVAILABLE -= 3
}
