package main

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"text/tabwriter"

	"github.com/evergreen-ci/evergreen/operations"
	"github.com/mongodb/grip"
	"github.com/urfave/cli"
)

func topfail() cli.Command {
	return cli.Command{
		Name:  "topfail",
		Usage: "find the most frequent test failures in recent waterfall builds",
		Flags: mergeFlagSlices(
			addProjectFlag(),
			[]cli.Flag{
				cli.IntFlag{
					Name:  versionsFlagName,
					Usage: "number of patches to show (0 for all patches)",
					Value: 6,
				},
				cli.IntFlag{
					Name:  limitFlagName,
					Usage: "number of most frequent test failures to show",
					Value: 20,
				},
			}),
		Action: func(c *cli.Context) error {
			confPath := c.Parent().String(confFlagName)
			projectID := c.String(projectFlagName)
			versions := c.Int(versionsFlagName)
			limit := c.Int(limitFlagName)

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
				versions)
			if err != nil {
				return fmt.Errorf("error getting revision info: %w", err)
			}

			type failedTestInfo struct {
				Test                   string
				FailedTasksPerRevision map[string][]string
				FailuresPerRevision    map[string]int
				TotalFailures          int
			}

			tests := make(map[string]*failedTestInfo) // map[test]failureStats

			for _, info := range infos {
				for _, variant := range info.FailedVariants {
					for _, task := range variant.FailedTasks {
						for _, test := range filterTests(task.FailedTests) {
							if tests[test] == nil {
								tests[test] = &failedTestInfo{
									FailuresPerRevision:    make(map[string]int),
									FailedTasksPerRevision: make(map[string][]string),
								}
							}
							tests[test].Test = test

							tasks := tests[test].FailedTasksPerRevision[info.Revision]
							tasks = append(tasks, task.Task)
							tests[test].FailedTasksPerRevision[info.Revision] = tasks

							tests[test].FailuresPerRevision[info.Revision]++
							tests[test].TotalFailures++
						}
					}
				}
			}

			testInfos := slices.Collect(maps.Values(tests))
			sort.Slice(testInfos, func(i, j int) bool { return testInfos[i].TotalFailures > testInfos[j].TotalFailures })

			if limit >= 0 && len(testInfos) > limit {
				n := limit
				if n >= len(testInfos) {
					n = len(testInfos) - 1
				}
				testInfos = testInfos[:n]
			}

			fmt.Println()

			w := new(tabwriter.Writer)
			// Format in tab-separated columns with a tab stop of 8.
			w.Init(os.Stdout, 0, 8, 0, '\t', 0)
			fmt.Fprintln(w, "\tCount\tTest Name")
			for _, info := range testInfos {
				line := fmt.Sprintf("\t%v\t%v", info.TotalFailures, info.Test)
				fmt.Fprintln(w, line)
			}

			return w.Flush()
		},
	}
}
