package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/evergreen-ci/evergreen/operations"
	"github.com/mongodb/grip"
	"github.com/urfave/cli"
)

func failstats() cli.Command {
	return cli.Command{
		Name:  "failstats",
		Usage: "show how many times a specific test fails per version, variant, and task",
		Flags: mergeFlagSlices(
			addProjectFlag(),
			[]cli.Flag{
				cli.IntFlag{
					Name:  joinFlagNames(versionsFlagName, "l"),
					Usage: "number of patches to show (0 for all patches)",
					Value: 6,
				},
				cli.StringFlag{
					Name:     joinFlagNames(testFlagName, "n"),
					Usage:    "the test name to filter for",
					Required: true,
				},
			}),
		Action: func(c *cli.Context) error {
			confPath := c.Parent().String(confFlagName)
			limit := c.Int(versionsFlagName)
			projectID := c.String(projectFlagName)
			testName := c.String(testFlagName)

			conf, err := operations.NewClientSettings(confPath)
			if err != nil {
				return fmt.Errorf("error loading configuration: %w", err)
			}

			if projectID == "" {
				grip.Debug("No project ID specified, trying to find default project for cwd")

				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("error getting cwd: %w", err)
				}
				cwd, err = filepath.EvalSymlinks(cwd)
				if err != nil {
					return fmt.Errorf("error evaluating symlinks for cwd: %w", err)
				}

				grip.Debugf("Trying to find default project for dir %q", cwd)

				projectID = conf.FindDefaultProject(cwd, false)
			}
			if projectID == "" {
				return errors.New("need to specify a project")
			}

			infos, err := getInfos(
				context.Background(),
				conf.User,
				conf.APIKey,
				projectID,
				limit,
			)
			if err != nil {
				return fmt.Errorf("error getting revision info: %w", err)
			}
			log.Print("infos", infos)

			versions := make(map[string]int)
			variants := make(map[string]int)
			tasks := make(map[string]int)
			for _, info := range infos {
				// versionInfo := fmt.Sprintf("https://spruce.mongodb.com/version/%s Created:%v", info.VersionID, info.Created)
				for _, variant := range info.FailedVariants {
					// variantInfo := fmt.Sprintf("Variant:%v", variant.DisplayName)
					for _, task := range variant.FailedTasks {
						// taskInfo := fmt.Sprintf("Task:%v", task.Task)
						for _, test := range task.FailedTests {
							if !strings.Contains(test, testName) {
								continue
							}
							// if versionInfo != "" {
							// 	fmt.Println(versionInfo)
							// 	versionInfo = ""
							// }
							// if variantInfo != "" {
							// 	fmt.Println(variantInfo)
							// 	variantInfo = ""
							// }
							// if taskInfo != "" {
							// 	fmt.Println(taskInfo)
							// 	taskInfo = ""
							// }
							versions[info.VersionID]++
							variants[variant.DisplayName]++
							tasks[task.Task]++
						}
					}
				}
			}

			printColumns := func(header string, rows map[string]int) {
				w := new(tabwriter.Writer)
				// Format in tab-separated columns with a tab stop of 8.
				w.Init(os.Stdout, 0, 8, 0, '\t', 0)
				fmt.Fprintln(w, header)

				type tuple struct {
					k string
					v int
				}

				tup := make([]tuple, 0, len(rows))

				for k, v := range rows {
					tup = append(tup, tuple{k: k, v: v})
				}
				sort.Slice(tup, func(i, j int) bool { return tup[i].v > tup[j].v })

				for _, t := range tup {
					line := fmt.Sprintf("\t%v\t%v", t.v, t.k)
					fmt.Fprintln(w, line)
				}
				w.Flush()
			}

			printColumns("\tCount\tVersion", versions)
			fmt.Println()
			printColumns("\tCount\tVariant", variants)
			fmt.Println()
			printColumns("\tCount\tTask", tasks)

			return nil
		},
	}
}
