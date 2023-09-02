package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

type tokenType uint8

const (
	globals tokenType = iota
	files
	cmds
)

const (
	yes = "y"
	no  = "n"
)

type (
	Config struct {
		Global map[string]any `json:"global"`
		Files  []File         `json:"files,omitempty"`
		Cmds   []Command      `json:"commands,omitempty"`
	}
	File struct {
		Name     string         `json:"name"`
		Path     string         `json:"path"`
		Template string         `json:"template"`
		Local    map[string]any `json:"local"`
	}
	Command struct {
		Name string   `json:"name"`
		Args []string `json:"args"`
	}
)

func main() {
	var (
		path   string
		output Config
		err    error
	)

	flag.StringVar(&path, "output", "", "output destination path (shortened)")
	flag.StringVar(&path, "o", "", "output destination path (shortened)")
	flag.Parse()

	defer func() {
		f := os.Stdout
		if path != "" {
			if f, err = os.Create(path); err != nil {
				log.Fatalln("failed to create output file:", err)
			}
		}
		if err = json.NewEncoder(f).Encode(output); err != nil {
			log.Printf("failed to produce output: %s\n", err)
		}
	}()

	for i := range []tokenType{
		globals,
		files,
		cmds,
	} {
		switch tokenType(i) {
		case globals:
			output.Global, err = readGlobals()
		case files:
			output.Files, err = readFiles()
		default:
			output.Cmds, err = readCommands()
		}
		if err != nil {
			log.Printf("failed to process config: %s", err)
			os.Exit(1)
		}
	}
}

const globalPrompt = `		-- Global parameters preparation --
Here you can add global variables that will be used throughout all templates.
Provide values as space separated tokens.

Example: SomeValue 123

Whould you like to add Global config values: y/n? `

func readGlobals() (map[string]any, error) {
	result, err := processVariables(globalPrompt)
	if err != nil {
		return nil, fmt.Errorf("global variables: %w", err)
	}
	return result, nil
}

const (
	filesPrompt = `		-- Files configuration part preparation --
This part is dedicated to specifying everything that has to do with file templates.
Each file consists of four parts:

	1. File name	   - the name of the file to be generated out of the template;
	2. File path	   - the path to where the file will be placed;
	3. Template name   - the name of the tmeplate to use;
	4. Local variables - the local variables specific to the specified template.

NOTE: there has to be at least one file to add.`
	localVarsPrompt = `		--- Local variables ---
Provide values as space separated tokens.
Example: SomeValue 123

Whould you like to add local config values: y/n? `
)

func readFiles() ([]File, error) {
	fmt.Printf("\n%s\n", filesPrompt)

	var (
		result []File
		err    error
	)
Cycle:
	for {
		var f File
		for i, v := range []string{
			"File name: ",
			"File path: ",
			"Template name: ",
		} {
			switch i {
			case 0:
				f.Name, err = scan(v)
			case 1:
				f.Path, err = scan(v)
			default:
				f.Template, err = scan(v)
			}
			if err != nil {
				return nil, fmt.Errorf("file parameters: %w", err)
			}
		}

		fmt.Println()
		f.Local, err = processVariables(localVarsPrompt)
		if err != nil {
			return nil, fmt.Errorf("file parameters: %w", err)
		}

		result = append(result, f)

		answer, err := scan("Add next file: y/n? ")
		if err != nil {
			return nil, fmt.Errorf("file parameters: %w", err)
		}
		switch answer {
		case yes:
			continue
		case no:
			break Cycle
		default:
		}
	}
	return result, nil
}

const commandsPrompt = `		-- Command post-hooks configuration preparation --
This part is dedicated to specifying everything that has to do with post-generation hooks.
Each entry consists of two parts:

	1. Command name	     - the name of the command to be called;
	2. Command arguments - the path to where the file will be placed.

Example: ls -a -l

Whould you like to add post-processing commands: y/n? `

func readCommands() ([]Command, error) {
	fmt.Printf("\n%s", commandsPrompt)

	var (
		result []Command
		s      = bufio.NewScanner(os.Stdin)
	)
Cycle:
	for s.Scan() {
		switch s.Text() {
		case yes:
			continue
		case no:
			break Cycle
		default:
		}
		parts := strings.Fields(s.Text())
		if len(parts) == 0 {
			return nil, fmt.Errorf("incorrect command declaration length")
		}
		result = append(result, Command{
			Name: parts[0],
			Args: parts[1:],
		})
		fmt.Print(`Add next value: y/n? `)
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("read commands: %w", err)
	}
	return result, nil
}

func format(v string) any {
	if val, err := strconv.ParseBool(v); err == nil {
		return val
	}
	if val, err := strconv.ParseInt(v, 10, 64); err == nil {
		return val
	}
	if val, err := strconv.ParseFloat(v, 64); err == nil {
		return val
	}
	return v
}

func processVariables(prompt string) (map[string]any, error) {
	fmt.Print(prompt)
	var (
		result map[string]any
		s      = bufio.NewScanner(os.Stdin)
	)
Cycle:
	for s.Scan() {
		switch s.Text() {
		case yes:
			continue
		case no:
			break Cycle
		default:
		}
		parts := strings.Split(s.Text(), " ")
		if len(parts) != 2 {
			return nil, fmt.Errorf("incorrect number of tokens")
		}
		if result == nil {
			result = make(map[string]any)
		}
		result[parts[0]] = format(parts[1])
		fmt.Print(`Add next value: y/n? `)
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("process variables: %w", err)
	}
	return result, nil
}

func scan(prompt string) (string, error) {
	fmt.Print(prompt)
	var temp string

	n, err := fmt.Scanf("%s", &temp)
	if err != nil {
		return "", err
	}
	if n != 1 {
		return "", fmt.Errorf("wrong number of tokens: %d", n)
	}
	return temp, nil
}
