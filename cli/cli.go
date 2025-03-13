package cli

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/ProjectLighthouseCAU/beacon/config"
	"github.com/ProjectLighthouseCAU/beacon/directory"
	"github.com/ProjectLighthouseCAU/beacon/resource"
	"github.com/ProjectLighthouseCAU/beacon/resource/brokerless"
	"github.com/ProjectLighthouseCAU/beacon/snapshot"
	"github.com/vmihailenco/msgpack/v5"
)

func RunCLI(stop chan struct{}, directory directory.Directory[resource.Resource[resource.Content]]) {
	reader := bufio.NewReader(os.Stdin)
Loop:
	for {
		fmt.Print("> ")
		s, err := reader.ReadString('\n')
		s = strings.TrimSuffix(s, "\n")
		if err != nil {
			fmt.Println(err)
			break
		}
		words := strings.Split(s, " ")
		switch words[0] {
		case "help":
			fmt.Println("Available commands:")
			fmt.Println("create <path/to/resource> - creates a new resource")
			fmt.Println("mkdir <path/to/directory> - creates a new directory")
			fmt.Println("delete <path/to/resource/or/directory> - deletes a resource or directory")
			fmt.Println("get <path/to/resource> - prints the content of a resource")
			fmt.Println("list <path/to/directory> - lists the contents of a directory")
			fmt.Println("link <path/to/dst> <path/to/src> - links two resources and forwards all data from src to dst")
			fmt.Println("unlink <path/to/dst> <path/to/src> - unlinks two resources")
			fmt.Println("snapshot <optional/filepath/to/snapshot-file> - creates a snapshot of the directory and all resources")
			fmt.Println("restore <optional/filepath/to/snapshot-file> - restores the directory and resources from a snapshot")
			fmt.Println("stop - stops the server gracefully (alt: Ctrl+C)")
		case "create":
			path := []string{}
			if len(words) > 1 {
				path = strings.Split(words[1], "/")
			}
			err := directory.CreateLeaf(path, brokerless.Create(path, resource.Nil))
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Println("Created resource", path)
		case "mkdir":
			path := []string{}
			if len(words) > 1 {
				path = strings.Split(words[1], "/")
			}
			err := directory.CreateDirectory(path)
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Println("Created directory", path)
		case "delete":
			path := []string{}
			if len(words) > 1 {
				path = strings.Split(words[1], "/")
			}
			err := directory.Delete(path)
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Println("Deleted", path)
		case "get":
			path := []string{}
			if len(words) > 1 {
				path = strings.Split(words[1], "/")
			}
			r, err := directory.GetLeaf(path)
			if err != nil {
				fmt.Println(err)
				continue
			}
			v := r.Get()
			fmt.Println("Raw data:", hex.EncodeToString(v))
			var x any
			err = msgpack.Unmarshal(([]byte)(v), &x)
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Println(x)
		case "list":
			path := []string{}
			if len(words) > 1 {
				path = strings.Split(words[1], "/")
			}
			s, err := directory.String(path)
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			fmt.Print(s)
		case "link":
			dstPath := []string{}
			srcPath := []string{}
			if len(words) > 1 {
				dstPath = strings.Split(words[1], "/")
			}
			if len(words) > 2 {
				srcPath = strings.Split(words[2], "/")
			}
			dst, err := directory.GetLeaf(dstPath)
			if err != nil {
				fmt.Println(err)
				continue
			}
			src, err := directory.GetLeaf(srcPath)
			if err != nil {
				fmt.Println(err)
				continue
			}
			err = dst.Link(src)
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Println("Link created", srcPath, "->", dstPath)
		case "unlink":
			dstPath := []string{}
			srcPath := []string{}
			if len(words) > 1 {
				dstPath = strings.Split(words[1], "/")
			}
			if len(words) > 2 {
				srcPath = strings.Split(words[2], "/")
			}
			dst, err := directory.GetLeaf(dstPath)
			if err != nil {
				fmt.Println(err)
				continue
			}
			src, err := directory.GetLeaf(srcPath)
			if err != nil {
				fmt.Println(err)
				continue
			}
			err = dst.UnLink(src)
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Println("Link deleted", srcPath, "->", dstPath)
		case "snapshot":
			snapshotPath := config.SnapshotPath
			if len(words) > 1 {
				snapshotPath = words[1]
			}
			err := snapshot.Snapshot(snapshotPath, directory)
			if err != nil {
				fmt.Println(err)
				continue
			}

		case "restore":
			snapshotPath := config.SnapshotPath
			if len(words) > 1 {
				snapshotPath = words[1]
			}
			err := snapshot.Restore(snapshotPath, directory)
			if err != nil {
				fmt.Println(err)
				continue
			}

		case "stop":
			close(stop)
			break Loop
		}
	}
}
