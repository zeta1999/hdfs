package main

import (
	"fmt"
	"github.com/colinmarc/hdfs/v2"
	"github.com/colinmarc/hdfs/v2/hadoopconf"
	"io"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"text/tabwriter"
)

func ls(paths []string, long, all, humanReadable bool) {

	if len(paths) == 0 {
		fatal("path is empty")
	}

	for _, perPath := range paths {
		files := make([]string, 0, len(paths))
		fileInfos := make([]os.FileInfo, 0, len(paths))
		dirs := make([]string, 0, len(paths))

		if strings.Compare(perPath, "/") == 0 {
			conf, exception := hadoopconf.LoadFromEnvironment()
			if exception != nil {
				fatal("NormalizePaths occur problem loading configuration: ", exception)
			}

			for k, v := range conf {
				if strings.Contains(k, "fs.viewfs.mounttable") {
					u, err := url.Parse(v)
					if err == nil && strings.Count(u.Path, "/") == 1 {
						dirs = append(dirs, u.Path)
					}
				}
			}
			for _, dir := range dirs {
				fmt.Println(dir)
			}
			continue
		}

		abPath, client, err := getClientAndExpandedPaths([]string{perPath})
		if err != nil {
			fatal(err)
		}

		for _, p := range abPath {
			fi, err := client.Stat(p)
			if err != nil {
				fatal(err)
			}

			if fi.IsDir() {
				dirs = append(dirs, p)
			} else {
				files = append(files, p)
				fileInfos = append(fileInfos, fi)
			}
		}

		if long {
			tw := lsTabWriter()
			for i, p := range files {
				printLong(tw, p, fileInfos[i], humanReadable)
			}

			tw.Flush()
		} else {
			for _, p := range files {
				fmt.Println(p)
			}
		}

		for _, dir := range dirs {
			printDir(client, dir, long, all, humanReadable)
		}
	}
}

func printDir(client *hdfs.Client, dir string, long, all, humanReadable bool) {
	dirReader, err := client.Open(dir)

	if err != nil {
		fatal(err)
	}

	var tw *tabwriter.Writer
	if long {
		tw = lsTabWriter()
		defer tw.Flush()
	}

	if all {
		if long {
			dirInfo, err := client.Stat(dir)
			if err != nil {
				fatal(err)
			}

			parentPath := path.Join(dir, "..")
			parentInfo, err := client.Stat(parentPath)
			if err != nil {
				fatal(err)
			}

			printLong(tw, ".", dirInfo, humanReadable)
			printLong(tw, "..", parentInfo, humanReadable)
		} else {
			fmt.Println(".")
			fmt.Println("..")
		}
	}

	var partial []os.FileInfo
	for ; err != io.EOF; partial, err = dirReader.Readdir(100) {
		if err != nil {
			fatal(err)
		}

		printFiles(tw, partial, long, all, humanReadable, dir)
	}

	if long {
		tw.Flush()
	}
}

func printFiles(tw *tabwriter.Writer, files []os.FileInfo, long, all, humanReadable bool, dir string) {

	for _, file := range files {
		if !all && strings.HasPrefix(file.Name(), ".") {
			continue
		}

		if long {
			printLong(tw, dir+"/"+file.Name(), file, humanReadable)
		} else {
			fmt.Println(dir + "/" + file.Name())
		}
	}
}

func printLong(tw *tabwriter.Writer, name string, info os.FileInfo, humanReadable bool) {
	fi := info.(*hdfs.FileInfo)
	// mode owner group size date(\w tab) time/year name
	mode := fi.Mode().String()
	owner := fi.Owner()
	group := fi.OwnerGroup()
	size := strconv.FormatInt(fi.Size(), 10)
	if humanReadable {
		size = formatBytes(uint64(fi.Size()))
	}

	modtime := fi.ModTime()
	date := modtime.Format("2006-01-02 15:04")
	/*var timeOrYear string
	if modtime.Year() == time.Now().Year() {
		timeOrYear = modtime.Format("2006 15:04")
	} else {
		timeOrYear = modtime.Format("2006")
	}*/

	fmt.Fprintf(tw, "%s \t%s \t %s \t %s \t%s \t%s\n",
		mode, owner, group, size, date, name)
}

func lsTabWriter() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 3, 8, 0, ' ', tabwriter.AlignRight|tabwriter.TabIndent)
}
