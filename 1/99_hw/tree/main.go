package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
)

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}

func dirTree(out io.Writer, path string, printFiles bool) error {
	err := recursiveDirTree(out, path, printFiles, *new([]int))
	if err != nil {
		return err
	}
	return nil
}

func recursiveDirTree(out io.Writer, path string, printFiles bool, depth []int) error {
	outWriter := bufio.NewWriter(out)

	arr, _ := os.ReadDir(path)
	sort.Slice(arr, func(i, j int) bool {
		return arr[i].Name() < arr[j].Name()
	})
	var arr1 []os.DirEntry
	if !printFiles {
		for _, file := range arr {
			if file.IsDir() {
				arr1 = append(arr1, file)
			}
		}
		arr = arr1
	}
	for n, file := range arr {
		for _, element := range depth {
			if element == 1 {
				fmt.Fprintf(outWriter, "│\t")
			} else {
				fmt.Fprintf(outWriter, "\t")
			}
		}
		if n == len(arr)-1 {
			if file.IsDir() {
				fmt.Fprintf(outWriter, "└───%s\n", file.Name())
				err := outWriter.Flush()
				if err != nil {
					return err
				}
				err = recursiveDirTree(out, path+"/"+file.Name(), printFiles, append(depth, 0))
				if err != nil {
					return err
				}
			} else {
				fileInfo, _ := file.Info()
				size := fileInfo.Size()
				if size == 0 {
					fmt.Fprintf(outWriter, "└───%s (empty)\n", file.Name())
				} else {
					fmt.Fprintf(outWriter, "└───%s (%db)\n", file.Name(), fileInfo.Size())
				}
				err := outWriter.Flush()
				if err != nil {
					return err
				}
			}
		} else {
			if file.IsDir() {
				fmt.Fprintf(outWriter, "├───%s\n", file.Name())
				outWriter.Flush()
				err := recursiveDirTree(out, path+"/"+file.Name(), printFiles, append(depth, 1))
				if err != nil {
					return err
				}
			} else {
				fileInfo, _ := file.Info()
				size := fileInfo.Size()
				if size == 0 {
					fmt.Fprintf(outWriter, "├───%s (empty)\n", file.Name())
				} else {
					fmt.Fprintf(outWriter, "├───%s (%db)\n", file.Name(), size)
				}
				err := outWriter.Flush()
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
