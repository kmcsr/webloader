
package main

import (
	os "os"
	exec "os/exec"
	fmt "fmt"
	time "time"
	strconv "strconv"

	filex "github.com/kmcsr/go-util/file"
	. "github.com/kmcsr/webloader/src"
)

const (
	MAJOR_VERSION = 0
	MINOR_VERSION = 1
	PATCH_VERSION = 0
)

func help(){
	println(`usage: webloader [-h|--help]
usage: webloader [--assets <assets src> <assets dst>] [--html <html src> <html dst> <assets url>]\
 [--path|-p <run path>] [--restart <time>] <cmd> [args...]

  --assets: Set assets file source dir and destination dir
  --html: Set html file (*.html/*.htm) source dir and destination dir
  --path, -p: Set runtime path, default current dir
  --restart: enable restart app when it crash and set restart timeout
  cmd & [args...]: Running after load assets and html file
`)
}

var currentPath = filex.RunPath()
var linker *AssetsLinker
var html_linker *HtmlLinker
var restart time.Duration = -1
var run_path string = currentPath

func main(){
	var (
		err error
		cmdargs []string
	)
	{ // init
		args := os.Args[1:]
		if len(args) == 0 {
			help()
			os.Exit(2)
			return
		}
		i := 0
		for ; i < len(args) ; i++ {
			a := args[i]
			if a[0] != '-' {
				break
			}
			switch a {
			case "-h", "--help":
				help()
				return
			case "--assets":
				i += 2
				if i >= len(args) {
					fmt.Printf("ERROR: Not enough arguments for '%s'\n", a)
					os.Exit(2)
					return
				}
				src, dst := args[i - 1], args[i]
				linker = NewAssetsLinker(filex.AbsPath(src), filex.AbsPath(dst))
				if html_linker != nil {
					html_linker.SetAssetsLinker(linker)
				}
			case "--html":
				i += 3
				if i >= len(args) {
					fmt.Printf("ERROR: Not enough arguments for '%s'\n", a)
					os.Exit(2)
					return
				}
				src, dst, url := args[i - 2], args[i - 1], args[i]
				html_linker = NewHtmlLinker(filex.AbsPath(src), filex.AbsPath(dst))
				html_linker.SetAssetsPrefix(url)
				if linker != nil {
					html_linker.SetAssetsLinker(linker)
				}
			case "-p", "--path":
				i += 1
				if i >= len(args) {
					fmt.Printf("ERROR: Not enough arguments for '%s'\n", a)
					os.Exit(2)
					return
				}
				run_path = filex.FixPath(filex.AbsPath(args[i]))
			case "--restart":
				i += 1
				if i >= len(args) {
					fmt.Printf("ERROR: Not enough arguments for '%s'\n", a)
					os.Exit(2)
					return
				}
				var sec int
				sec, err = strconv.Atoi(args[i])
				if err != nil {
					fmt.Printf("ERROR: Error number '%s': %v\n", args[i], err)
				}
				if sec < 0 {
					sec = 0
				}
				restart = time.Duration(sec) * time.Second
			}
		}
		cmdargs = args[i:]
		if len(cmdargs) == 0 {
			println("ERROR: You must give a command after the arguments\n")
			help()
			os.Exit(2)
			return
		}
	}
	fmt.Printf(`
   ##########     #######
   #        #    #     #       ____        ____
   # \ /\ / #   #     #        \   \      /   /
   #  V  V  #  #     #          \   \    /   /
   #        # #     #            \   \  /   /
   #  ___   ##     #              \   \/   /
   # |      #     #                \      /
   # |--         #                  \____/
   # |____      #
   #          l  #
   #  ___   #  o  #                  %2d
   # | _ \  ##  a  #                  .
   # |___/  # #  d  #                %2d
   # | _ \  #  #  e  #                .
   # |___/  #   #  r  #              %2d
   #        #    #     #
   ##########     #######

`, MAJOR_VERSION, MINOR_VERSION, PATCH_VERSION)
	if html_linker != nil {
		fmt.Println("Loading html linker...")
		err = html_linker.Load()
		if err != nil {
			panic(err)
		}
		fmt.Println("html linker loaded")
	}else if linker != nil {
		fmt.Println("Loading assets linker...")
		err = linker.Load()
		if err != nil {
			panic(err)
		}
		fmt.Println("Assets linker loaded")
	}
	if run_path != currentPath {
		os.Chdir(run_path)
	}
	finish := make(chan struct{}, 1)
	var command *exec.Cmd
	go func(){
		for {
			command = buildCommand(cmdargs)
			err = command.Start()
			fmt.Printf("Starting: `%s`\n", command.String())
			if err != nil {
				fmt.Printf("Start command `%s` failed: %v\n", command.String(), err)
				panic(err)
			}
			err = command.Wait()
			if err != nil {
				if exiterr, ok := err.(*exec.ExitError); ok {
					fmt.Printf("Run command `%s` failed: exit code %d\n", command.String(), exiterr.ExitCode())
					if restart >= 0 {
						time.Sleep(restart)
						continue
					}
				}else{
					fmt.Printf("Run command `%s` failed: %v\n", command.String(), err)
				}
				panic(err)
			}
			fmt.Printf("Finished: `%s`\n", command.String())
			break
		}
		close(finish)
	}()
	select{
	case <-finish:
	}
}

func buildCommand(args []string)(cmd *exec.Cmd){
	cmd = exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	return
}

