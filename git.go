package main

/**
 * Git Repo Information
 */

import (
	"log"
	"strings"
	"github.com/fatih/color"
)

////////////////////////////////////////////
// Utility: Git Repo Info
////////////////////////////////////////////

type RepoStatusField struct {
	OutputCharacter rune
	OutputColor     *color.Color
}

// Key is the git status rune (what shows up in `git status -sb`)
var RepoStatusFieldDefinitionsOrderedKeys = []rune{'M', 'A', 'D', 'R', 'C', 'U', '?', '!'}
var RepoStatusFieldDefinitions = map[rune]RepoStatusField{
	// modified
	'M': {OutputCharacter: 'M', OutputColor: color.New(color.FgGreen)},
	// added
	'A': {OutputCharacter: '+', OutputColor: color.New(color.FgHiGreen)},
	// deleted
	'D': {OutputCharacter: '-', OutputColor: color.New(color.FgHiRed)},
	// renamed
	'R': {OutputCharacter: 'R', OutputColor: color.New(color.FgHiYellow)},
	// copied
	'C': {OutputCharacter: 'C', OutputColor: color.New(color.FgHiBlue)},
	// updated
	'U': {OutputCharacter: 'U', OutputColor: color.New(color.FgHiMagenta)},
	// untracked
	'?': {OutputCharacter: '?', OutputColor: color.New(color.FgRed)},
	// ignored
	'!': {OutputCharacter: '!', OutputColor: color.New(color.FgCyan)},
}

type RepoInfo struct {
	IsRepo bool
	BranchNameColored string
	BranchName string
	OtherBranches []string
	Status       string
	StatusColored       string
}

func NewRepoInfo() *RepoInfo {
	// TODO: Make this not run a command to get this data
	// Go do a git status in that folder
	output, exitCode, err := execAndGetOutput("git", nil,
		"-c", "color.status=never", "-c", "color.ui=never", "status")

	if err != nil {
		// Some kind of command execution error!
		log.Printf("Failed to get git output for repo: %v", err)
		return nil
	} else if exitCode == 128 {
		// Not a git repo
		return &RepoInfo{}  // IsRepo defaults to false
	} else if exitCode != 0 {
		// Some kind of git error!
		log.Printf("Bad exit code getting git output for repo: %v", exitCode)
		return nil
	}

	info := &RepoInfo{IsRepo:true}

	// Figure out branch status TODO: This could be optimized I bet
	branchColor := color.New(color.FgGreen)

	if strings.Contains(output, "still merging") || strings.Contains(output, "Unmerged paths") {
		branchColor = color.New(color.FgHiMagenta)
	} else if strings.Contains(output, "Untracked files") {
		branchColor = color.New(color.FgHiRed)
	} else if strings.Contains(output, "Changes not staged for commit") {
		branchColor = color.New(color.FgHiYellow)
	} else if strings.Contains(output, "Changes to be committed") {
		branchColor = color.New(color.FgYellow)
	} else if strings.Contains(output, "Your branch is ahead of") {
		branchColor = color.New(color.FgMagenta)
	}

	// Figure out branches
	output, _, err = execAndGetOutput("git", nil,
		"-c", "color.status=never", "-c", "color.ui=never", "branch")

	if err == nil {
		lines := strings.Split(output, "\n")

		info.OtherBranches = []string{}

		for _, line := range lines {
			if strings.HasPrefix(line, "* ") {
				info.BranchName = strings.TrimPrefix(line, "* ")
			} else {
				info.OtherBranches = append(info.OtherBranches, strings.TrimSpace(line))
			}
		}

		info.BranchNameColored = branchColor.Sprint(info.BranchName)
	} else {
		info.OtherBranches = []string{}
		info.BranchName = "!branch!"
		info.BranchNameColored = branchColor.Sprint(info.BranchName)
	}

	if err == nil {
		// Get per-file status

		status := make(map[rune]int, len(RepoStatusFieldDefinitions))
		for field := range RepoStatusFieldDefinitions {
			status[field] = 0
		}

		output, _, err = execAndGetOutput("git", nil,
			"-c", "color.status=never", "-c", "color.ui=never", "status", "-s")

		lines := strings.Split(output, "\n")

		for _, line := range lines {
			line = strings.TrimSpace(line)

			if len(line) < 2 {
				continue
			}

			// Grab first two characters
			statchars := line[:2]

			for key := range status {
				if strings.ContainsRune(statchars, key) {
					status[key]++
				}
			}
		}

		info.StatusColored = buildColoredStatusStringFromMap(status)
		info.Status = stripANSI(info.StatusColored)
	} else {
		info.Status = "!status!"
		info.StatusColored = color.HiRedString(info.Status)
	}

	return info
}

func buildColoredStatusStringFromMap(status map[rune]int) string {
	retval := ""

	for _, key := range RepoStatusFieldDefinitionsOrderedKeys {
		count := status[key]

		if count > 0 {
			if retval != "" {
				retval += " "
			}

			retval += RepoStatusFieldDefinitions[key].OutputColor.Sprintf("%c:%d",
				RepoStatusFieldDefinitions[key].OutputCharacter, count)
		}
	}

	return retval
}
