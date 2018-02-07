package main

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/wayneashleyberry/terminal-dimensions"
	"github.com/pborman/getopt/v2"
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

var EXIT_CODE int
var WORKING_DIRECTORY string

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
	var homePath = WORKING_DIRECTORY

	// Match the path to "HOME"
	var HOME = os.ExpandEnv("$HOME")
	var CANONHOME = normalizePath(HOME)

	if strings.HasPrefix(WORKING_DIRECTORY, HOME) {
		relative, relErr := filepath.Rel(HOME, WORKING_DIRECTORY)

		if relErr == nil {
			homePath = filepath.Join("~", relative)
		}
	} else if strings.HasPrefix(WORKING_DIRECTORY, CANONHOME) {
		relative, relErr := filepath.Rel(CANONHOME, WORKING_DIRECTORY)

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
	if EXIT_CODE != 0 {
		errStr := fmt.Sprintf("~%d~", EXIT_CODE)
		return errStr, color.HiRedString(errStr)
	} else {
		return "", ""
	}
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

func parseOptions() {
	//
	// Set up options
	//

	exitcode := getopt.IntLong("exitcode", 'e', EXIT_CODE, "The exit code of the previously run command.")

	fullPath, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	workingdir := getopt.StringLong("dir", 'd', fullPath, "The working directory to pretend we're in.\nNOTE: Tilde (~) expansion is best-effort and should not be relied on.")

	width := getopt.IntLong("width", 'w', 0, "Override detected terminal width.")

	//
	// Parse
	//

	getopt.Parse()

	EXIT_CODE = *exitcode
	WORKING_DIRECTORY = *workingdir
	WIDTH = *width

	//
	// Validate results
	//

	if WIDTH <= 0 {
		WIDTH = getWidth()
	}

	if len(WORKING_DIRECTORY) <= 0 {
		WORKING_DIRECTORY = fullPath
	}

	if WORKING_DIRECTORY[:2] == "~/" {
		WORKING_DIRECTORY = filepath.Join(os.ExpandEnv("$HOME"), WORKING_DIRECTORY[2:])
	}
}

func main() {

	//////////////////
	// Options/Setup
	//////////////////

	parseOptions()

	//////////////////
	// FIRST LINE
	//////////////////

	FIRST_LINE_WIDTH_AVAILABLE = WIDTH

	// Leading space/bracket
	fmt.Print(SPACER + LSQBRACKET)
	FIRST_LINE_WIDTH_AVAILABLE -= 2

	// Username
	usr, usrColor := username()
	fmt.Print(usrColor)
	FIRST_LINE_WIDTH_AVAILABLE -= len(usr)

	// Active jobs
	jobs, jobsColor := atjobs()
	fmt.Print(jobsColor)
	FIRST_LINE_WIDTH_AVAILABLE -= len(jobs)

	// Hostname and CPU load
	host, hostColor := hostload()
	fmt.Print(hostColor)
	FIRST_LINE_WIDTH_AVAILABLE -= len(host)

	// Trailing bracket/space
	fmt.Print(RSQBRACKET + SPACER)
	FIRST_LINE_WIDTH_AVAILABLE -= 2

	// Load working directory information
	dir, dirColor := cwd(FIRST_LINE_WIDTH_AVAILABLE - (1 + 2 + 2))

	// Spaces needed for directory line
	spacersRequired := FIRST_LINE_WIDTH_AVAILABLE - (2 + len(dir) + 2)
	if spacersRequired < 1 {
		spacersRequired = 1
	}
	firstLineDynamicSpace := strings.Repeat(SPACER, spacersRequired)

	fmt.Print(firstLineDynamicSpace)
	FIRST_LINE_WIDTH_AVAILABLE -= spacersRequired

	// Leading space/brace
	fmt.Print(SPACER + LBRACE)
	FIRST_LINE_WIDTH_AVAILABLE -= 2

	// Current directory
	fmt.Print(dirColor)
	FIRST_LINE_WIDTH_AVAILABLE -= len(dir)

	// Trailing brace/space
	fmt.Print(RBRACE + SPACER)
	FIRST_LINE_WIDTH_AVAILABLE -= 2

	fmt.Println()

	//////////////////
	// SECOND LINE
	//////////////////

	SECOND_LINE_WIDTH_AVAILABLE = WIDTH

	// Initial spacers
	fmt.Print(SPACER+SPACER)
	SECOND_LINE_WIDTH_AVAILABLE -= 2

	// Current time
	tme, tmeColor := curtime()
	fmt.Print(tmeColor)
	SECOND_LINE_WIDTH_AVAILABLE -= len(tme)

	// Battery status
	batt, battColor := battery()
	if len(batt) > 0 {
		fmt.Print(battColor)
		SECOND_LINE_WIDTH_AVAILABLE -= len(batt)
	}

	// Error code from last command
	errCode, errCodeColor := getErrorCode()
	if len(errCode) > 0 {
		fmt.Print(errCodeColor)
		SECOND_LINE_WIDTH_AVAILABLE -= len(errCode)
	}

	// Kerberos ticket status
	kerberos, kerberosColor := getKerberos()
	if len(kerberos) > 0 {
		fmt.Print(kerberosColor)
		SECOND_LINE_WIDTH_AVAILABLE -= len(kerberos)
	}

	// Load git info
	gitInfo := NewRepoInfo(&WORKING_DIRECTORY)

	if gitInfo != nil && gitInfo.IsRepo {
		branch, branchColor := gitBranch(gitInfo)
		files, filesColor := gitFiles(gitInfo)

		// Git branch line
		fmt.Print(branchColor)
		SECOND_LINE_WIDTH_AVAILABLE -= len(branch)

		// Spacers with the git file status on the right side
		spacersRequired := SECOND_LINE_WIDTH_AVAILABLE - (len(files) + 3)
		if spacersRequired < 1 {
			spacersRequired = 1
		}
		secondLineDynamicSpace := strings.Repeat(" ", spacersRequired)

		fmt.Print(secondLineDynamicSpace)
		SECOND_LINE_WIDTH_AVAILABLE -= spacersRequired

		// Git file status
		fmt.Print(filesColor)
	} else {
		// Spacers without anything on the right side
		spacersRequired := SECOND_LINE_WIDTH_AVAILABLE - (3)
		if spacersRequired < 1 {
			spacersRequired = 1
		}
		secondLineDynamicSpace := strings.Repeat(" ", spacersRequired)

		fmt.Print(secondLineDynamicSpace)
		SECOND_LINE_WIDTH_AVAILABLE -= spacersRequired
	}

	// Right spacers
	fmt.Print(" " + SPACER + SPACER)
	SECOND_LINE_WIDTH_AVAILABLE -= 3

	fmt.Println()
}
