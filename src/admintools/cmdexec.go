package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"
	"io"
	"os"
)

const fileCacheTime = 30   // seconds
const default_timeout = 10 // seconds

type Command struct {
	Command  string              `json:"command" yaml:"command"`
	Required []map[string]string `json:"required" yaml:"required"`
	Validate []map[string]string `json:"validate" yaml:"validate"`
	Timeout  int                 `json:"timeout" yaml:"timeout"`
}

type Commands []Command

type jsonResponse struct {
	Result map[string]string
}

type appError struct {
	Message string
	Code    int
}

type flushWriter struct {
	f http.Flusher
	w io.Writer
}

func (fw *flushWriter) Write(p []byte) (n int, err error) {
	_, _ = os.Stdout.Write(p)
	n, err = fw.w.Write(p)
	if fw.f != nil {
		fw.f.Flush()
	}
	return
}

func New() *Commands {
	cmds := new(Commands)
	return cmds
}

func (c *Commands) AddCommand(cmd Command) {
	*c = append(*c, cmd)
}

func (c *Commands) RunCommands() string {
	response := ""
	//for _, cmd := range *c {
	//	response += cmd.Run()
	//}
	return response
}

func (c *Command) Run(stdout, stderr *flushWriter) {
	log.Printf("Running: %v\n", c)
	var args []string
	command := strings.Split(c.Command, " ")
	if len(command) > 1 {
		args = command[1:]
	}
	if c.Timeout == 0 {
		c.Timeout = default_timeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Timeout)*time.Second)
	defer cancel()
	cmdcontext := exec.CommandContext(ctx, command[0], args...)
	cmdcontext.Stdout = stdout
	cmdcontext.Stderr = stderr
	err := cmdcontext.Run()
	if err != nil {
		log.Printf("Error %v: %v. stdout: %s stderr: %s \n", command[0], err, stdout, stderr)
	}
	//log.Printf("Command %v: Timeout: %d Output:\n%s\n", c.Command, c.Timeout, out.String())
	//return out.String()
}

func InteractiveExec(w http.ResponseWriter, file string, parameters map[string][]string) {
	filedata, err, jsonerr := validateFile(file, parameters)
	if err != nil {
		w.Write([]byte(jsonerr))
		return
	}

	w.Header().Set("Connection", "Transfer-Encoding")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Transfer-Encoding", "chunked")
	fw := flushWriter{w: w}
	if f, ok := w.(http.Flusher); ok {
		fw.f = f
	}

	for _, cmd := range filedata.Commands {
		cmd.Run(&fw, &fw)
	}

}

func RenderFile(file string, parameters map[string][]string, w http.ResponseWriter) {
	filedata, err := readFile(file)
	if err != nil {
		w.Write([]byte(returnError(err, 100)))
	}

	if err = checkParameters(&filedata, parameters, true); err != nil {
		w.Write([]byte(returnError(err, 110)))
	} else {
		for _, cmd := range filedata.Commands {
			log.Printf("Running: %v\n", cmd)
			var args []string
			command := strings.Split(cmd.Command, " ")
			if len(command) > 1 {
				args = command[1:]
			}
			if cmd.Timeout == 0 {
				cmd.Timeout = default_timeout
			}
			w.Write([]byte(RunCommand(command[0], cmd.Timeout, args)))
		}
	}

}

func RunCommand(cmd string, timeout int, args []string) string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	result, err := exec.CommandContext(ctx, cmd, args...).Output()
	if err != nil {
		log.Printf("Error %v: %v. Res: %s \n", cmd, err, result)
	} else {
		//log.Printf("Command %v: Timeout: %d = %s\n", cmd, timeout, result)
	}
	log.Printf("Command %v: Timeout: %d = %s\n", cmd, timeout, result)
	return string(result[:])
}

func checkParameters(filedata *fileFormat, parameters map[string][]string, required bool) (err error) {
	// check params
	for index, cmd := range filedata.Commands {
		loop_through := cmd.Validate
		if required {
			loop_through = cmd.Required
		}
		for _, req := range loop_through {
			for name, expr := range req {
				if len(parameters[name]) == 0 && required {
					return errors.New(fmt.Sprintf("Parameter '%s' is missing", name))
				} else {
					for _, value := range parameters[name] {
						re := regexp.MustCompile(expr)
						rexp := re.MatchString(value)
						if err != nil {
							return errors.New(fmt.Sprintf("Can not parse regexp '%s' for '%s'", expr, name))
						}
						if rexp != true {
							return errors.New(fmt.Sprintf("Value '%s' is not valid.", name))
						}
						filedata.Commands[index].Command = strings.Replace(filedata.Commands[index].Command, "{{"+name+"}}", value, -1)
					}
				}
			}
		}

	}
	if required {
		return checkParameters(filedata, parameters, false)
	}
	return nil
}

func validateFile(file string, parameters map[string][]string) (filedata fileFormat, err error, jsonerror string) {
	filedata, err = readFile(file)
	if err != nil {
		jsonerror = returnError(err, 100)
	}

	if err = checkParameters(&filedata, parameters, true); err != nil {
		jsonerror = returnError(err, 110)
	}
	return
}

func ResponseToText(response jsonResponse) string {
	text := ""
	for _, result := range response.Result {
		text += result
	}
	return text
}

func ResponseToJSON(response interface{}) string {
	encode, _ := json.Marshal(response)
	return string(encode)
}

func returnError(err error, code int) string {
	var errorData appError
	errorData.Message = err.Error()
	errorData.Code = code
	log.Print(errorData)
	return ResponseToJSON(errorData)
}

func replacePlaceholders(fd *fileFormat) {
	for index, cmd := range fd.Commands {
		fd.Commands[index].Command = strings.Replace(cmd.Command, "{{", "", -1)
	}
}

func errorNew(vars ...string) error {
	var result string
	for _, v := range vars {
		result += fmt.Sprintf(v)
	}
	return errors.New(result)
}
