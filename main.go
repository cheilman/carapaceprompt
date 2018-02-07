package main

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/pborman/getopt/v2"
	"github.com/wayneashleyberry/terminal-dimensions"
	"golang.org/x/sys/unix"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var DEFAULT = color.New(color.FgGreen)
var SPACER = DEFAULT.Sprint("-")
var LSQBRACKET = DEFAULT.Sprint("[")
var RSQBRACKET = DEFAULT.Sprint("]")
var LBRACE = DEFAULT.Sprint("{")
var RBRACE = DEFAULT.Sprint("}")
var LANBRACKET = DEFAULT.Sprint("<")
var RANBRACKET = DEFAULT.Sprint(">")

var HOME = os.ExpandEnv("$HOME")

var WIDTH int
var FIRST_LINE_WIDTH_AVAILABLE int
var SECOND_LINE_WIDTH_AVAILABLE int

var EXIT_CODE int
var WORKING_DIRECTORY string
var HAS_RUNNING_JOBS bool
var HAS_SUSPENDED_JOBS bool

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
	c := color.New(color.FgCyan)

	if HAS_SUSPENDED_JOBS {
		c = color.New(color.FgHiRed, color.Bold)
	} else if HAS_RUNNING_JOBS {
		c = color.New(color.FgHiGreen, color.Bold)
	}

	return "@", c.Sprint("@")
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

	// Get load
	loadColor := color.New(color.FgCyan)
	info := NewCPUInfo()

	if info.Load1MinPercentage > 1.00 {
		loadColor = color.New(color.BgRed, color.FgHiWhite, color.Bold)
		hostName = fmt.Sprintf("%s(%0.2f)", hostName, info.Load1Min)
	} else if info.Load1MinPercentage > 0.75 {
		loadColor = color.New(color.FgHiRed, color.Bold)
		hostName = fmt.Sprintf("%s(%0.2f)", hostName, info.Load1Min)
	} else if info.Load1MinPercentage > 0.50 {
		loadColor = color.New(color.FgHiMagenta, color.Bold)
		hostName = fmt.Sprintf("%s(%0.2f)", hostName, info.Load1Min)
	} else if info.Load1MinPercentage > 0.25 {
		loadColor = color.New(color.FgHiYellow, color.Bold)
	}

	return hostName, loadColor.Sprint(hostName)
}

func cwd(dirWidthAvailable int) (string, string) {
	var homePath = WORKING_DIRECTORY

	// Match the path to "HOME"
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

	// Figure out directory color
	dirColor := color.New(color.FgHiGreen)

	// Check writable
	if unix.Access(WORKING_DIRECTORY, unix.W_OK) == nil {
		// Writable, check space left
		output, _, err := execAndGetOutput("df", &WORKING_DIRECTORY, "-P", WORKING_DIRECTORY)
		if err != nil {
			// Error!
			homePath = "!" + homePath + "!"
			dirColor = color.New(color.FgHiMagenta, color.Bold)
		} else {
			// Try to parse output
			lines := strings.Split(strings.TrimSpace(output), "\n")

			// We care about the 2nd line
			if len(lines) > 1 {
				// Now we care about the 5th column.  This POSIX output, we could streamline by using --output (GNU)
				// https://stackoverflow.com/a/46798310
				splitFn := func(c rune) bool {
					return c == ' '
				}
				fields := strings.FieldsFunc(strings.TrimSpace(lines[1]), splitFn)

				if len(fields) >= 4 {
					content := strings.TrimSuffix(fields[4], "%")
					perc, err := strconv.Atoi(content)

					if err != nil {
						// Everything is terrible
						homePath = "=" + homePath + "="
						dirColor = color.New(color.FgHiBlack)
					} else {
						// Finally!  Color according to space left
						if perc > 90 {
							dirColor = color.New(color.BgRed, color.FgHiWhite, color.Bold)
						} else if perc > 80 {
							dirColor = color.New(color.FgHiRed, color.Bold)
						} else if perc > 70 {
							dirColor = color.New(color.FgHiYellow, color.Bold)
						}
					}
				} else {
					// Failed yet again
					homePath = "+" + homePath + "+"
					dirColor = color.New(color.FgYellow)
				}
			} else {
				// Couldn't figure it out
				homePath = "~" + homePath + "~"
				dirColor = color.New(color.FgMagenta, color.Bold)
			}
		}
	} else {
		// Not writable
		dirColor = color.New(color.FgRed)
	}

	// Return
	return homePath, dirColor.Sprint(homePath)
}

func curtime() (string, string) {
	t := time.Now().Local().Format("15:04")
	return t, color.YellowString(t)
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
	if !fileExists(filepath.Join(HOME, "host/config/ignore_kerberos")) {
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
		branchLine += " " + color.WhiteString("{"+strings.Join(info.OtherBranches, ", ")+"}")
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

	hasrunningjobs := getopt.BoolLong("runningjobs", 'r', "Flag that indicates if the shell has background jobs running.")
	hassuspendedjobs := getopt.BoolLong("suspendedjobs", 's', "Flag that indicates if the shell has background jobs that are suspended.")

	//
	// Parse
	//

	getopt.Parse()

	EXIT_CODE = *exitcode
	WORKING_DIRECTORY = *workingdir
	WIDTH = *width
	HAS_RUNNING_JOBS = *hasrunningjobs
	HAS_SUSPENDED_JOBS = *hassuspendedjobs

	//
	// Validate results
	//

	if WIDTH <= 0 {
		WIDTH = getWidth()
	}

	if len(WORKING_DIRECTORY) <= 0 {
		WORKING_DIRECTORY = fullPath
	}

	if len(WORKING_DIRECTORY) > 1 && WORKING_DIRECTORY[:1] == "~" {
		if len(WORKING_DIRECTORY) > 2 && WORKING_DIRECTORY[:2] == "~/" {
			WORKING_DIRECTORY = filepath.Join(HOME, WORKING_DIRECTORY[2:])
		} else {
			WORKING_DIRECTORY = HOME
		}
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
	fmt.Print(SPACER + SPACER)
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
