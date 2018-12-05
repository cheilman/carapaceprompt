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

var DEFAULT *color.Color
var SPACER string
var LSQBRACKET string
var RSQBRACKET string
var LBRACE string
var RBRACE string
var LANBRACKET string
var RANBRACKET string

var HOME = os.ExpandEnv("$HOME")

var WIDTH int
var FIRST_LINE_WIDTH_AVAILABLE int
var SECOND_LINE_WIDTH_AVAILABLE int

var EXIT_CODE int
var WORKING_DIRECTORY string
var HAS_RUNNING_JOBS bool
var HAS_SUSPENDED_JOBS bool
var SHOW_BATTERY bool
var VCS_STATUS_CMD string
var WD_FORMAT_CMD string

type VCSInfo struct {
	Branch string
	Files  string
}

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

	if WORKING_DIRECTORY == "" {
		// Invalid working directory
		badDirStr := "<missing>"
		invalidDirColor := color.New(color.FgHiRed, color.Bold, color.BlinkSlow)
		return badDirStr, invalidDirColor.Sprint(badDirStr)
	}

	var homePath = WORKING_DIRECTORY

	// If a WD_FORMAT_CMD is specified, run our path through that
	if WD_FORMAT_CMD != "" {
		output, _, err := execAndGetOutput(WD_FORMAT_CMD, &WORKING_DIRECTORY, homePath)

		if err == nil {
			homePath = strings.TrimSpace(output)
		}
	}

	// Match the path to "HOME"
	var CANONHOME = normalizePath(HOME)

	if strings.HasPrefix(homePath, HOME) {
		relative, relErr := filepath.Rel(HOME, homePath)

		if relErr == nil {
			homePath = filepath.Join("~", relative)
		}
	} else if strings.HasPrefix(homePath, CANONHOME) {
		relative, relErr := filepath.Rel(CANONHOME, homePath)

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
	if SHOW_BATTERY {
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
				if battInfo.TimeLeft.Seconds() > 0 {
					// Display time left
					return "<" + fmt.Sprintf("%0d:%02d", int(battInfo.TimeLeft.Hours()), int(battInfo.TimeLeft.Minutes())) + ">",
						LANBRACKET + battInfo.ColorizedTimeLeft + RANBRACKET
				} else {
					// Display nothing (this is a weird error case sometimes)
					return "<>", DEFAULT.Sprint("<>")
				}
			}
		}
	} else {
		return "<>", DEFAULT.Sprint("<>")
	}
}

func getErrorCode() (string, string) {
	if EXIT_CODE != 0 {
		errStr := fmt.Sprintf(" :%d:", EXIT_CODE)
		return errStr, color.HiRedString(errStr)
	} else {
		return "", ""
	}
}

func getKerberos() (string, string) {
	// See if we even care (flag in host config)
	path := filepath.Join(HOME, ".host/config/check_kerberos")
	if fileExists(path) {
		// Do we have a ticket?
		_, exitCode, _ := execAndGetOutput("klist", nil, "-s")

		hasTicket := exitCode == 0

		if hasTicket {
			return "", ""
		} else {
			return " [K]", color.New(color.FgHiRed, color.Bold).Sprint(" [K]")
		}
	} else {
		return "", ""
	}
}

func getMidwayCert() (string, string) {
	// See if we even care (flag in host config)
	path := filepath.Join(HOME, ".host/config/check_midway")
	if fileExists(path) {
		// Do we have a cert?
		output, exitCode, _ := execAndGetOutput("mwinit", nil, "-l")

		hasCert := exitCode == 0

		if hasCert {
			hasCert = len(output) > 0
		}

		if hasCert {
			return "", ""
		} else {
			return " [M]", color.New(color.FgHiRed, color.Bold).Sprint(" [M]")
		}
	} else {
		return "", ""
	}
}

func getVCSInfo(workingdir *string) *VCSInfo {
	if workingdir == nil || len(*workingdir) <= 0 {
		return nil
	}

	if len(VCS_STATUS_CMD) <= 0 {
		return nil
	}

	// Run the command
	output, exitCode, err := execAndGetOutput(VCS_STATUS_CMD, workingdir,
		"--output=prompt", "--color", "--vcs=git")

	if err != nil || exitCode != 0 {
		return nil
	}

	// Output is the two lines we want
	lines := strings.Split(output, "\n")

	if len(lines) < 2 {
		// Invalid output format
		return nil
	}

	return &VCSInfo{
		Branch: "   " + strings.TrimSpace(lines[0]),
		Files:  strings.TrimSpace(lines[1]),
	}
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

	exitcode := getopt.IntLong("exitcode", 'e', EXIT_CODE,
		"The exit code of the previously run command.")

	fullPath, err := os.Getwd()
	if err != nil {
		// Working directory doesn't exist anymore
		WORKING_DIRECTORY = ""
	} else {
		workingdir := getopt.StringLong("dir", 'd', fullPath,
			"The working directory to pretend we're in.\nNOTE: Tilde (~) expansion is best-effort and should not be relied on.")
		WORKING_DIRECTORY = *workingdir
	}

	wdFormatCmd := getopt.StringLong("wdformat", 'p', "",
		"If specified, the current working directory will be passed through this command for additional formatting/truncation.")

	vcscmd := getopt.StringLong("vcs", 'g', "vcsstatus",
		"Command to run that outputs VCS information.")

	width := getopt.IntLong("width", 'w', 0,
		"Override detected terminal width.")

	hasrunningjobs := getopt.BoolLong("runningjobs", 'r',
		"Flag that indicates if the shell has background jobs running.")
	hassuspendedjobs := getopt.BoolLong("suspendedjobs", 's',
		"Flag that indicates if the shell has background jobs that are suspended.")

	showBattery := getopt.BoolLong("showBattery", 'b',
		"Should we attempt to show battery data on the prompt.")

	forcecolor := getopt.BoolLong("color", 'c',
		"Force colored output.")

	//
	// Parse
	//

	getopt.Parse()

	EXIT_CODE = *exitcode
	WIDTH = *width
	HAS_RUNNING_JOBS = *hasrunningjobs
	HAS_SUSPENDED_JOBS = *hassuspendedjobs
	SHOW_BATTERY = *showBattery
	VCS_STATUS_CMD = *vcscmd
	WD_FORMAT_CMD = *wdFormatCmd

	if *forcecolor {
		color.NoColor = false
	}

	//
	// Validate results
	//

	if WIDTH <= 0 {
		WIDTH = getWidth()
	}

	if len(WORKING_DIRECTORY) < 0 {
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

func setupDefaults() {
	// Colors need to happen after command line options to force color
	DEFAULT = color.New(color.FgGreen)
	SPACER = DEFAULT.Sprint("-")
	LSQBRACKET = DEFAULT.Sprint("[")
	RSQBRACKET = DEFAULT.Sprint("]")
	LBRACE = DEFAULT.Sprint("{")
	RBRACE = DEFAULT.Sprint("}")
	LANBRACKET = DEFAULT.Sprint("<")
	RANBRACKET = DEFAULT.Sprint(">")

	HOME = os.ExpandEnv("$HOME")
}

func main() {

	//////////////////
	// Options/Setup
	//////////////////

	parseOptions()

	setupDefaults()

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

	// Kerberos ticket status
	kerberos, kerberosColor := getKerberos()
	if len(kerberos) > 0 {
		fmt.Print(kerberosColor)
		SECOND_LINE_WIDTH_AVAILABLE -= len(kerberos)
	}

	// Midway ticket status
	midway, midwayColor := getMidwayCert()
	if len(midway) > 0 {
		fmt.Print(midwayColor)
		SECOND_LINE_WIDTH_AVAILABLE -= len(midway)
	}

	// Error code from last command
	errCode, errCodeColor := getErrorCode()
	if len(errCode) > 0 {
		fmt.Print(errCodeColor)
		SECOND_LINE_WIDTH_AVAILABLE -= len(errCode)
	}

	// Load vcs info
	if WORKING_DIRECTORY != "" {
		vcsInfo := getVCSInfo(&WORKING_DIRECTORY)

		if vcsInfo != nil {
			branchColor := vcsInfo.Branch
			branch := stripANSI(branchColor)

			filesColor := vcsInfo.Files
			files := stripANSI(filesColor)

			// Branch line
			fmt.Print(branchColor)
			SECOND_LINE_WIDTH_AVAILABLE -= len(branch)

			// Spacers with the file status on the right side
			spacersRequired := SECOND_LINE_WIDTH_AVAILABLE - (len(files) + 3)
			if spacersRequired < 1 {
				spacersRequired = 1
			}
			secondLineDynamicSpace := strings.Repeat(" ", spacersRequired)

			fmt.Print(secondLineDynamicSpace)
			SECOND_LINE_WIDTH_AVAILABLE -= spacersRequired

			// File status
			fmt.Print(filesColor)
			SECOND_LINE_WIDTH_AVAILABLE -= len(files)
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
	}

	// Right spacers
	fmt.Print(" " + SPACER + SPACER)
	SECOND_LINE_WIDTH_AVAILABLE -= 3

	fmt.Println()
}
