package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/evergreen-ci/evergreen"
	"github.com/mitchellh/go-homedir"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/level"
	"github.com/mongodb/grip/send"
	"github.com/urfave/cli"
)

const (
	versionsFlagName    = "versions"
	limitFlagName       = "limit"
	testFlagName        = "test"
	showSummaryFlagName = "show-summary"
	projectFlagName     = "project"
	confFlagName        = "conf"
)

func main() {
	// this is where the main action of the program starts. The
	// command line interface is managed by the cli package and
	// its objects/structures. This, plus the basic configuration
	// in buildApp(), is all that's necessary for bootstrapping the
	// environment.
	app := buildApp()
	grip.EmergencyFatal(app.Run(os.Args))
}

func buildApp() *cli.App {
	app := cli.NewApp()
	app.Name = "evergreen"
	app.Usage = "MongoDB Continuous Integration Platform"
	app.Version = evergreen.ClientVersion

	// Register sub-commands here.
	app.Commands = []cli.Command{
		topfail(),
		failstats(),
	}

	userHome, err := homedir.Dir()
	if err != nil {
		// workaround for cygwin if we're on windows but couldn't get a homedir
		if runtime.GOOS == "windows" && len(os.Getenv("HOME")) > 0 {
			userHome = os.Getenv("HOME")
		}
	}
	confPath := filepath.Join(userHome, evergreen.DefaultEvergreenConfig)

	// These are global options. Use this to configure logging or
	// other options independent from specific sub commands.
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "level",
			Value: "info",
			Usage: "Specify lowest visible log level as string: 'emergency|alert|critical|error|warning|notice|info|debug|trace'",
		},
		cli.StringFlag{
			Name:  "conf, config, c",
			Usage: "specify the path for the evergreen CLI config",
			Value: confPath,
		},
	}

	app.Before = func(c *cli.Context) error {
		return loggingSetup(app.Name, c.String("level"))
	}

	return app
}

func loggingSetup(name, l string) error {
	if err := grip.SetSender(send.MakeErrorLogger()); err != nil {
		return err
	}
	grip.SetName(name)

	sender := grip.GetSender()
	info := sender.Level()
	info.Threshold = level.FromString(l)

	return sender.SetLevel(info)
}

const (
	mainlineFailuresQuery = `
  query MainlineCommits(
	$mainlineCommitsOptions: MainlineCommitsOptions!
	$buildVariantOptions: BuildVariantOptions!
	$buildVariantOptionsForGraph: BuildVariantOptions!
	$buildVariantOptionsForTaskIcons: BuildVariantOptions!
	$buildVariantOptionsForGroupedTasks: BuildVariantOptions!
  ) {
	mainlineCommits(
	  options: $mainlineCommitsOptions
	  buildVariantOptions: $buildVariantOptions
	) {
	  nextPageOrderNumber
	  prevPageOrderNumber
	  versions {
		rolledUpVersions {
		  author
		  createTime
		  id
		  ignored
		  message
		  order
		  revision
		  __typename
		}
		version {
		  author
		  buildVariants(options: $buildVariantOptionsForTaskIcons) {
			displayName
			tasks {
			  displayName
			  execution
			  id
			  status
			  timeTaken
			  __typename
			}
			variant
			__typename
		  }
		  buildVariantStats(options: $buildVariantOptionsForGroupedTasks) {
			displayName
			statusCounts {
			  count
			  status
			  __typename
			}
			variant
			__typename
		  }
		  createTime
		  gitTags {
			pusher
			tag
			__typename
		  }
		  id
		  message
		  order
		  projectIdentifier
		  revision
		  taskStatusStats(options: $buildVariantOptionsForGraph) {
			counts {
			  count
			  status
			  __typename
			}
			eta
			__typename
		  }
		  ...UpstreamProject
		  __typename
		}
		__typename
	  }
	  __typename
	}
  }
  
  fragment UpstreamProject on Version {
	upstreamProject {
	  owner
	  project
	  repo
	  revision
	  task {
		execution
		id
		__typename
	  }
	  triggerID
	  triggerType
	  version {
		id
		__typename
	  }
	  __typename
	}
	__typename
  }`

	taskTestSampleQuery = `
  query ($versionId: String!, $taskIds: [String!]!, $filters: [TestFilter!]!) {
	taskTestSample(versionId: $versionId, taskIds: $taskIds, filters: $filters) {
	  execution
	  matchingFailedTestNames
	  taskId
	  totalTestCount
	}
  }`
)

type revisionInfo struct {
	VersionID      string
	Created        time.Time
	Revision       string
	Message        string
	FailedVariants []variantInfo
}

type variantInfo struct {
	DisplayName string
	FailedTasks []taskInfo
}

type taskInfo struct {
	Task        string
	FailedTests []string
}

func getInfos(
	ctx context.Context,
	user, apiKey, projectID string,
	versions int,
) ([]revisionInfo, error) {
	// Define the types required to unmarshal the mainlineCommits GraphQL
	// response.
	type taskRes struct {
		DisplayName string `json:"displayName"`
		Execution   int    `json:"execution"`
		ID          string `json:"id"`
		Status      string `json:"status"`
	}
	type buildVariant struct {
		DisplayName string    `json:"displayName"`
		Tasks       []taskRes `json:"tasks"`
	}
	type versionRes struct {
		ID            string         `json:"id"`
		Revision      string         `json:"revision"`
		Message       string         `json:"message"`
		BuildVariants []buildVariant `json:"buildVariants"`
		CreateTime    time.Time      `json:"createTime"`
	}
	type mainlineCommitVersion struct {
		Version versionRes `json:"version"`
	}
	type mainlineCommits struct {
		Versions []mainlineCommitVersion `json:"versions"`
	}

	// Run the mainlineCommits query that returns a summary of the failures in
	// the last N mainline versions (i.e. waterfall builds) for the given
	// project ID.
	mainlineFailuresVars := map[string]any{
		"mainlineCommitsOptions": map[string]any{
			"projectIdentifier": projectID,
			"limit":             versions,
			"shouldCollapse":    false,
			"requesters":        []string{},
		},
		"buildVariantOptions": map[string]any{
			"tasks":            []string{},
			"variants":         []string{},
			"statuses":         []string{},
			"includeBaseTasks": false,
		},
		"buildVariantOptionsForGraph": map[string]any{
			"statuses": []string{},
			"tasks":    []string{},
			"variants": []string{},
		},
		"buildVariantOptionsForGroupedTasks": map[string]any{
			"tasks":    []string{"^\b$"},
			"variants": []string{},
			"statuses": []string{},
		},
		"buildVariantOptionsForTaskIcons": map[string]any{
			"tasks":    []string{},
			"variants": []string{},
			"statuses": []string{
				"failed",
				"task-timed-out",
				"test-timed-out",
				"known-issue",
				"setup-failed",
				"system-failed",
				"system-timed-out",
				"system-unresponsive",
				"aborted",
			},
			"includeBaseTasks": false,
		},
	}
	resJSON, err := graphql(
		ctx,
		user,
		apiKey,
		mainlineFailuresQuery,
		mainlineFailuresVars)
	if err != nil {
		return nil, fmt.Errorf("error querying mainlineCommits: %w", err)
	}
	var res struct {
		Data struct {
			MainlineCommits mainlineCommits `json:"mainlineCommits"`
		} `json:"data"`
	}
	err = json.Unmarshal(resJSON, &res)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling mainlineCommits: %w", err)
	}

	// Define the type required to unmarshal the taskTestSample GraphQL
	// responses.
	type taskTestSample struct {
		Execution               int      `json:"execution"`
		MatchingFailedTestNames []string `json:"matchingFailedTestNames"`
		TaskID                  string   `json:"taskId"`
		TotalTestCount          int      `json:"totalTestCount"`
	}

	// For each version, run the taskTestSample query to get the failed test
	// names for all tasks in that version.
	infos := make([]revisionInfo, 0)
	for _, ver := range res.Data.MainlineCommits.Versions {
		failedVariants := make([]variantInfo, 0)
		for _, variant := range ver.Version.BuildVariants {
			taskIDs := make(map[string]string, len(variant.Tasks)) // map[taskId]displayName
			for _, t := range variant.Tasks {
				taskIDs[t.ID] = t.DisplayName
			}

			resJSON, err := graphql(
				ctx,
				user,
				apiKey,
				taskTestSampleQuery,
				map[string]any{
					"versionId": ver.Version.ID,
					"taskIds":   slices.Collect(maps.Keys(taskIDs)),
					"filters":   []string{},
				})
			if err != nil {
				return nil, fmt.Errorf("error querying taskTestSample: %w", err)
			}

			var res struct {
				Data struct {
					TaskTestSample []taskTestSample `json:"taskTestSample"`
				} `json:"data"`
			}
			err = json.Unmarshal(resJSON, &res)
			if err != nil {
				return nil, fmt.Errorf("error unmarshaling taskTestSample: %w", err)
			}

			grip.Debugln("Version ID:", ver.Version.ID, "Task IDs:", taskIDs, "Failing Tests:")

			failedTasks := make([]taskInfo, 0)
			for _, sample := range res.Data.TaskTestSample {
				grip.Debugf("Version:", ver.Version.ID, "Task:", taskIDs[sample.TaskID])
				for _, test := range sample.MatchingFailedTestNames {
					grip.Debugln(test)
				}
				failedTasks = append(failedTasks, taskInfo{
					Task:        taskIDs[sample.TaskID],
					FailedTests: sample.MatchingFailedTestNames,
				})
			}
			failedVariants = append(failedVariants, variantInfo{
				DisplayName: variant.DisplayName,
				FailedTasks: failedTasks,
			})
		}

		infos = append(infos, revisionInfo{
			VersionID:      ver.Version.ID,
			Created:        ver.Version.CreateTime,
			Revision:       ver.Version.Revision,
			Message:        ver.Version.Message,
			FailedVariants: failedVariants,
		})
	}

	return infos, nil
}

// TODO: Re-add caching.
// func cached() {
// 	// Caching.
// 	// TODO: Remove?
// 	{
// 		const cacheFilePrefix = ".evergreen_cache_tests_"
// 		h := sha1.New()
// 		h.Write(body)
// 		sfx := hex.EncodeToString(h.Sum(nil))
// 		if b, err := os.ReadFile(cacheFilePrefix + sfx); err == nil {
// 			return b, nil
// 		}
// 		defer func() {
// 			if err != nil {
// 				return
// 			}
// 			os.WriteFile(cacheFilePrefix+sfx, data, 0666)
// 		}()
// 	}
// }

// graphql queries the Evergreen GraphQL API using the provided user creds,
// query, and variables. It returns the response body as a byte slice.
func graphql(
	ctx context.Context,
	user string,
	apiKey string,
	query string,
	variables map[string]any,
) ([]byte, error) {
	body, err := json.Marshal(map[string]any{
		"query":     query,
		"variables": variables,
	})
	if err != nil {
		return nil, fmt.Errorf("error marshaling variables: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://evergreen.mongodb.com/graphql/query",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("error building GraphQL query: %w", err)
	}
	req.Header.Add(evergreen.APIUserHeader, user)
	req.Header.Add(evergreen.APIKeyHeader, apiKey)
	req.Header.Add("content-type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error querying GraphQL API: %w", err)
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading GraphQL response: %w", err)
	}

	var errRes struct {
		Errors []map[string]any `json:"errors"`
	}
	err = json.Unmarshal(data, &errRes)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling GraphQL response to check for errors: %w", err)
	}

	if len(errRes.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL API returned errors: %v", errRes.Errors)
	}

	return data, nil
}

// func (fti *failedTestInfo) String() string {
// 	if fti == nil {
// 		return fmt.Sprint(nil)
// 	}

// 	return fmt.Sprintf("%s: %+v", fti.Test, struct {
// 		// FailedTasksPerRevision map[string][]string
// 		FailuresPerRevision map[string]int
// 		TotalFailures       int
// 	}{
// 		// FailedTasksPerRevision: fti.FailedTasksPerRevision,
// 		FailuresPerRevision: fti.FailuresPerRevision,
// 		TotalFailures:       fti.TotalFailures,
// 	})
// }

func filterTests(tests []string) []string {
	sort.Strings(tests)

	res := make([]string, 0, len(tests))
	for i := range tests {
		if i >= len(tests)-1 || strings.HasPrefix(tests[i+1], tests[i]+"/") {
			continue
		}
		res = append(res, tests[i])
	}
	return res
}

// Command line stuff

func mergeFlagSlices(in ...[]cli.Flag) []cli.Flag {
	out := []cli.Flag{}

	for idx := range in {
		out = append(out, in[idx]...)
	}

	return out
}

func addProjectFlag(flags ...cli.Flag) []cli.Flag {
	return append(flags, cli.StringFlag{
		Name:  joinFlagNames(projectFlagName, "p"),
		Usage: "specify the name of an existing Evergreen project",
	})
}

func joinFlagNames(ids ...string) string { return strings.Join(ids, ", ") }
