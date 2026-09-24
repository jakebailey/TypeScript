package build

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/collections"
	"github.com/microsoft/TypeScript/tsc/internal/compiler"
	"github.com/microsoft/TypeScript/tsc/internal/contentmapper"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/diagnostics"
	"github.com/microsoft/TypeScript/tsc/internal/execute/incremental"
	"github.com/microsoft/TypeScript/tsc/internal/execute/tsc"
	"github.com/microsoft/TypeScript/tsc/internal/execute/watchmanager"
	"github.com/microsoft/TypeScript/tsc/internal/fswatch"
	"github.com/microsoft/TypeScript/tsc/internal/tsoptions"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/internal/vfs/cachedvfs"
)

type Options struct {
	Sys     tsc.System
	Command *tsoptions.ParsedBuildCommandLine
	Testing tsc.CommandLineTesting
}
type OrchestratorResult struct {
	Result        tsc.CommandLineResult
	Errors        []*ast.Diagnostic
	Statistics    tsc.Statistics
	FilesToDelete []string
}

func (b *OrchestratorResult) report(o *Orchestrator) {
	b.reportWithFilesToDelete(o, true)
}

func (b *OrchestratorResult) reportWithFilesToDelete(o *Orchestrator, reportFilesToDelete bool) {
	if o.opts.Command.CompilerOptions.Watch.IsTrue() {
		o.watchStatusReporter(ast.NewCompilerDiagnostic(core.IfElse(len(b.Errors) == 1, diagnostics.Found_1_error_Watching_for_file_changes, diagnostics.Found_0_errors_Watching_for_file_changes), len(b.Errors)))
	} else {
		o.errorSummaryReporter(b.Errors)
	}
	if reportFilesToDelete && b.FilesToDelete != nil {
		o.createBuilderStatusReporter(nil)(
			ast.NewCompilerDiagnostic(
				diagnostics.A_non_dry_build_would_delete_the_following_files_Colon_0,
				strings.Join(core.Map(b.FilesToDelete, func(f string) string {
					return "\r\n * " + f
				}), ""),
			),
		)
	}
	if !o.opts.Command.CompilerOptions.Diagnostics.IsTrue() && !o.opts.Command.CompilerOptions.ExtendedDiagnostics.IsTrue() {
		return
	}
	b.Statistics.SetTotalTime(o.opts.Sys.SinceStart())
	b.Statistics.Report(o.opts.Sys.Writer(), o.opts.Testing)
}

type Orchestrator struct {
	opts                Options
	comparePathsOptions tspath.ComparePathsOptions
	host                *host

	// contentMapperHost transforms content-mapped files; it is created once per build session (when
	// enabled) and shared across all projects so mapper processes are consolidated. It closes itself when
	// the session context is cancelled (see contentmapper.New).
	contentMapperHost contentmapper.Host

	// order generation result
	tasks          *collections.SyncMap[tspath.Path, *BuildTask]
	order          []string
	errors         []*ast.Diagnostic
	graphGenerated bool

	errorSummaryReporter tsc.DiagnosticsReporter
	watchStatusReporter  tsc.DiagnosticReporter

	// fswatch event-based watching
	wm *watchmanager.WatchManager
	// order sorted by dependency depth, to reduce how often builders block on upstream projects
	scheduleOrder []string
}

var _ tsc.Watcher = (*Orchestrator)(nil)

func (o *Orchestrator) relativeFileName(fileName string) string {
	return tspath.ConvertToRelativePath(fileName, o.comparePathsOptions)
}

func (o *Orchestrator) toPath(fileName string) tspath.Path {
	return tspath.ToPath(fileName, o.comparePathsOptions.CurrentDirectory, o.comparePathsOptions.UseCaseSensitiveFileNames)
}

func (o *Orchestrator) resolveBuildInfoFileName(fileName string, buildInfoDir string) string {
	if incremental.IsBuildInfoFileNameDefaultLibrary(fileName) {
		return tspath.CombinePaths(o.host.DefaultLibraryPath(), fileName)
	}
	return tspath.GetNormalizedAbsolutePath(fileName, buildInfoDir)
}

func (o *Orchestrator) Order() []string {
	return o.order
}

// ScheduleOrder is the order in which builders pick up projects: Order() stably sorted by dependency depth.
func (o *Orchestrator) ScheduleOrder() []string {
	return o.scheduleOrder
}

// computeScheduleOrder sorts the build order by dependency depth (projects with no
// upstream first, then their dependents, and so on). Builders take projects from this
// order and block until upstream projects are done, so with the plain depth-first order
// a builder that picks the root of a long chain sits idle while another builder works
// through the chain, even when unrelated projects are ready to build. Depth order reduces
// that avoidable blocking but does not eliminate it: a shallower project that has been
// picked up may not be done yet, so a builder can take a dependent of a slow project and
// wait on that project while a later project's upstream has already finished. The stable
// sort preserves the original order within a depth, and reporting still follows Order().
func (o *Orchestrator) computeScheduleOrder() []string {
	type scheduleEntry struct {
		config string
		depth  int
	}
	entries := make([]scheduleEntry, len(o.order))
	depths := make(map[*BuildTask]int, len(o.order))
	for i, config := range o.order {
		task := o.getTask(o.toPath(config))
		depth := 0
		for _, upstream := range task.upStream {
			depth = max(depth, depths[upstream.task]+1)
		}
		depths[task] = depth
		entries[i] = scheduleEntry{config: config, depth: depth}
	}
	slices.SortStableFunc(entries, func(a, b scheduleEntry) int {
		return a.depth - b.depth
	})
	return core.Map(entries, func(entry scheduleEntry) string {
		return entry.config
	})
}

func (o *Orchestrator) Upstream(configName string) []string {
	path := o.toPath(configName)
	task := o.getTask(path)
	return core.Map(task.upStream, func(t *upstreamTask) string {
		return t.task.config
	})
}

func (o *Orchestrator) Downstream(configName string) []string {
	path := o.toPath(configName)
	task := o.getTask(path)
	return core.Map(task.downStream, func(t *BuildTask) string {
		return t.config
	})
}

func (o *Orchestrator) getTask(path tspath.Path) *BuildTask {
	task, ok := o.tasks.Load(path)
	if !ok {
		panic("No build task found for " + path)
	}
	return task
}

func (o *Orchestrator) createBuildTasks(oldTasks *collections.SyncMap[tspath.Path, *BuildTask], configs []string, wg core.WorkGroup) {
	for _, config := range configs {
		wg.Queue(func() {
			path := o.toPath(config)
			var task *BuildTask
			var buildInfo *buildInfoEntry
			if oldTasks != nil {
				if existing, ok := oldTasks.Load(path); ok {
					if !existing.dirty {
						// Reuse existing task if config is same
						task = existing
					} else {
						if existing.contentMapperProject != nil {
							_ = existing.contentMapperProject.Close()
						}
						buildInfo = existing.buildInfoEntry
					}
				}
			}
			if task == nil {
				task = &BuildTask{config: config, isInitialCycle: oldTasks == nil}
				task.pending.Store(true)
				task.buildInfoEntry = buildInfo
			}
			if _, loaded := o.tasks.LoadOrStore(path, task); loaded {
				return
			}
			task.resolved = o.host.GetResolvedProjectReference(config, path)
			task.upStream = nil
			if task.resolved != nil {
				o.createBuildTasks(oldTasks, task.resolved.ResolvedProjectReferencePaths(), wg)
			}
		})
	}
}

func (o *Orchestrator) setupBuildTask(
	configName string,
	downStream *BuildTask,
	inCircularContext bool,
	completed *collections.Set[tspath.Path],
	analyzing *collections.Set[tspath.Path],
	circularityStack []string,
) *BuildTask {
	path := o.toPath(configName)
	task := o.getTask(path)
	if !completed.Has(path) {
		if analyzing.Has(path) {
			if !inCircularContext {
				o.errors = append(o.errors, ast.NewCompilerDiagnostic(
					diagnostics.Project_references_may_not_form_a_circular_graph_Cycle_detected_Colon_0,
					strings.Join(circularityStack, "\n"),
				))
			}
			return nil
		}
		analyzing.Add(path)
		circularityStack = append(circularityStack, configName)
		if task.resolved != nil {
			for index, subReference := range task.resolved.ResolvedProjectReferencePaths() {
				upstream := o.setupBuildTask(subReference, task, inCircularContext || task.resolved.ProjectReferences()[index].Circular, completed, analyzing, circularityStack)
				if upstream != nil {
					task.upStream = append(task.upStream, &upstreamTask{task: upstream, refIndex: index})
				}
			}
		}
		circularityStack = circularityStack[:len(circularityStack)-1]
		completed.Add(path)
		task.built = make(chan struct{})
		task.done = make(chan struct{})
		o.order = append(o.order, configName)
	}
	if o.opts.Command.CompilerOptions.Watch.IsTrue() && downStream != nil {
		task.downStream = append(task.downStream, downStream)
	}
	return task
}

func (o *Orchestrator) GenerateGraphReusingOldTasks() {
	tasks := o.tasks
	o.tasks = &collections.SyncMap[tspath.Path, *BuildTask]{}
	o.order = nil
	o.errors = nil
	o.GenerateGraph(tasks)
}

func (o *Orchestrator) GenerateGraph(oldTasks *collections.SyncMap[tspath.Path, *BuildTask]) {
	projects := o.opts.Command.ResolvedProjectPaths()
	// Parse all config files in parallel
	wg := core.NewWorkGroup(o.opts.Command.CompilerOptions.SingleThreaded.IsTrue())
	o.createBuildTasks(oldTasks, projects, wg)
	wg.RunAndWait()

	// Generate the graph
	completed := collections.Set[tspath.Path]{}
	analyzing := collections.Set[tspath.Path]{}
	circularityStack := []string{}
	for _, project := range projects {
		o.setupBuildTask(project, nil, false, &completed, &analyzing, circularityStack)
	}
	o.scheduleOrder = o.computeScheduleOrder()
	if oldTasks != nil {
		oldTasks.Range(func(path tspath.Path, oldTask *BuildTask) bool {
			if task, ok := o.tasks.Load(path); ok && task == oldTask {
				return true
			}
			if oldTask.contentMapperProject != nil {
				_ = oldTask.contentMapperProject.Close()
			}
			return true
		})
	}
	o.graphGenerated = true
}

// tsc -b entrypoint
func (o *Orchestrator) Start(ctx context.Context) tsc.CommandLineResult {
	return o.start(ctx, "", false /*onlyReferences*/).Result
}

// orchestrator.Build() entrypoint for api
func (o *Orchestrator) Build(ctx context.Context, project string) *OrchestratorResult {
	o.recheckAllProjects(project)
	return o.start(ctx, project, false /*onlyReferences*/)
}

// orchestrator.BuildReferences() entrypoint for api
func (o *Orchestrator) BuildReferences(ctx context.Context, project string) *OrchestratorResult {
	o.recheckAllProjects(project)
	return o.start(ctx, project, true /*onlyReferences*/)
}

func (o *Orchestrator) start(ctx context.Context, project string, onlyReferences bool) *OrchestratorResult {
	o.contentMapperHost = tsc.NewContentMapperHost(ctx, o.opts.Sys, o.opts.Command.CompilerOptions)
	if o.contentMapperHost != nil && (!o.opts.Command.CompilerOptions.Watch.IsTrue() || o.opts.Testing == nil) {
		defer o.contentMapperHost.Close()
	}
	if o.opts.Command.CompilerOptions.Watch.IsTrue() {
		o.watchStatusReporter(ast.NewCompilerDiagnostic(diagnostics.Starting_compilation_in_watch_mode))
	}
	if o.graphGenerated {
		o.GenerateGraphReusingOldTasks()
	} else {
		o.GenerateGraph(nil)
	}
	order, ok := o.getBuildOrderFor(project)
	if !ok {
		return &OrchestratorResult{Result: tsc.CommandLineResult{Status: tsc.ExitStatusInvalidProject_OutputsSkipped}}
	}
	if onlyReferences && len(o.errors) == 0 {
		if project == "" {
			return &OrchestratorResult{Result: tsc.CommandLineResult{Status: tsc.ExitStatusInvalidProject_OutputsSkipped}}
		}
		order = order[:len(order)-1]
	}
	result := o.buildOrCleanOrder(order, false)
	if o.opts.Command.CompilerOptions.Watch.IsTrue() {
		o.Watch(ctx)
		result.Result.Watcher = o
	}
	return result
}

func (o *Orchestrator) recheckAllProjects(project string) {
	if !o.graphGenerated {
		return
	}
	order, ok := o.getBuildOrderFor(project)
	if !ok {
		return
	}
	o.rangeTasks(order, func(path tspath.Path, task *BuildTask) {
		task.resetStatus()
		task.resetConfig(o, o.toPath(task.config))
	})
	o.host.mTimes = &collections.SyncMap[tspath.Path, time.Time]{}
	o.resetCaches()
}

// orchestrator.Clean() entrypoint for api
func (o *Orchestrator) Clean(project string) *OrchestratorResult {
	return o.clean(project, false)
}

// orchestrator.CleanReferences() entrypoint for api
func (o *Orchestrator) CleanReferences(project string) *OrchestratorResult {
	return o.clean(project, true)
}

func (o *Orchestrator) clean(project string, onlyReferences bool) *OrchestratorResult {
	if !o.graphGenerated {
		o.GenerateGraph(nil)
	}
	if len(o.errors) != 0 {
		result := &OrchestratorResult{
			Result: tsc.CommandLineResult{Status: tsc.ExitStatusProjectReferenceCycle_OutputsSkipped},
			Errors: o.errors,
		}
		result.reportWithFilesToDelete(o, true)
		return result
	}

	order, ok := o.getBuildOrderFor(project)
	if !ok {
		return &OrchestratorResult{Result: tsc.CommandLineResult{Status: tsc.ExitStatusInvalidProject_OutputsSkipped}}
	}
	if onlyReferences {
		order = order[:len(order)-1]
	}

	result := &OrchestratorResult{}
	result.Statistics.Projects = len(order)
	dry := o.opts.Command.BuildOptions.Dry.IsTrue()
	reportDiagnostic := o.createDiagnosticReporter(nil)
	for _, config := range order {
		task := o.getTask(o.toPath(config))
		if task.resolved == nil {
			diagnostic := ast.NewCompilerDiagnostic(diagnostics.File_0_not_found, task.config)
			reportDiagnostic(diagnostic)
			result.Errors = append(result.Errors, diagnostic)
			continue
		}

		inputs := collections.NewSetFromItems(core.Map(task.resolved.FileNames(), o.toPath)...)
		projectOutputs := task.resolved.GetOutputFileNames()
		deleted := false
		for outputFile := range projectOutputs {
			deleted = o.cleanProjectOutput(outputFile, inputs, dry, &result.FilesToDelete, reportDiagnostic) || deleted
		}
		deleted = o.cleanProjectOutput(task.resolved.GetBuildInfoFileName(), inputs, dry, &result.FilesToDelete, reportDiagnostic) || deleted
		if deleted {
			task.resetStatus()
			task.buildInfoEntryMu.Lock()
			task.buildInfoEntry = nil
			task.buildInfoEntryMu.Unlock()
		}
	}

	result.reportWithFilesToDelete(o, dry)
	return result
}

func (o *Orchestrator) getBuildOrderFor(project string) ([]string, bool) {
	if project == "" {
		return o.order, true
	}

	config := core.ResolveConfigFileNameOfProjectReference(
		tspath.ResolvePath(o.opts.Sys.GetCurrentDirectory(), project),
	)
	target, ok := o.tasks.Load(o.toPath(config))
	if !ok {
		return nil, false
	}

	projects := collections.Set[tspath.Path]{}
	var addProjectAndReferences func(*BuildTask)
	addProjectAndReferences = func(task *BuildTask) {
		path := o.toPath(task.config)
		if projects.Has(path) {
			return
		}
		projects.Add(path)
		for _, upstream := range task.upStream {
			addProjectAndReferences(upstream.task)
		}
	}
	addProjectAndReferences(target)

	order := make([]string, 0, len(projects.M))
	for _, config := range o.order {
		if projects.Has(o.toPath(config)) {
			order = append(order, config)
		}
	}
	return order, true
}

func (o *Orchestrator) cleanProjectOutput(
	outputFile string,
	inputs *collections.Set[tspath.Path],
	dry bool,
	filesToDelete *[]string,
	reportDiagnostic tsc.DiagnosticReporter,
) bool {
	if outputFile == "" || inputs.Has(o.toPath(outputFile)) || !o.host.FS().FileExists(outputFile) {
		return false
	}
	*filesToDelete = append(*filesToDelete, outputFile)
	if dry {
		return false
	}
	if err := o.host.FS().Remove(outputFile); err != nil {
		reportDiagnostic(ast.NewCompilerDiagnostic(diagnostics.Failed_to_delete_file_0, outputFile))
		return false
	}
	return true
}

func (o *Orchestrator) Watch(ctx context.Context) {
	o.wm.Lock()

	if o.opts.Testing == nil {
		if value, _ := o.opts.Sys.GetEnvironmentVariable("TS_WATCH_DEBUG"); value != "" {
			o.wm.DebugLog = o.opts.Sys.Writer()
		}
		o.wm.EnsureDefaultBackend()
	}

	o.updateWatch()
	watchFiles, logicalDirs, desiredDirs := o.computeDesiredWatches()
	if err := o.wm.ReconcileWatches(watchFiles, desiredDirs, o.host.FS(), logicalDirs...); err != nil {
		fmt.Fprintf(o.opts.Sys.Writer(), "%v\n", err)
		o.wm.ForceOverflow()
	}
	o.resetCaches()

	o.wm.Unlock()

	if o.opts.Testing == nil {
		o.wm.RunLoop(ctx, o.DoCycle)
	}
}

func (o *Orchestrator) updateWatch() {
	oldCache := o.host.mTimes
	o.host.mTimes = &collections.SyncMap[tspath.Path, time.Time]{}
	o.rangeTask(func(path tspath.Path, task *BuildTask) {
		task.updateWatch(o, oldCache)
	})
}

func (o *Orchestrator) resetCaches() {
	// Clean out all the caches
	cachesVfs := o.host.host.FS().(*cachedvfs.FS)
	cachesVfs.ClearCache()
	o.host.extendedConfigCache = tsc.ExtendedConfigCache{}
	o.host.sourceFiles.reset()
	o.host.configTimes = collections.SyncMap[tspath.Path, time.Duration]{}
}

func (o *Orchestrator) checkTasksForEventChanges(changedPaths map[string]fswatch.EventKind, needsConfigUpdate, needsUpdate *atomic.Bool) {
	normalizedPaths := make(map[tspath.Path]fswatch.EventKind, len(changedPaths))
	for eventPath, kind := range changedPaths {
		normalizedPaths[o.toPath(eventPath)] = kind
	}

	for i := range o.order {
		config := o.order[i]
		path := o.toPath(config)
		task := o.getTask(path)

		if o.watchFileChanged(task.config, normalizedPaths) {
			task.resetConfig(o, path)
			needsConfigUpdate.Store(true)
			needsUpdate.Store(true)
			continue
		}

		if task.resolved == nil {
			continue
		}

		configChanged := false
		for _, file := range task.resolved.ExtendedSourceFiles() {
			if o.watchFileChanged(file, normalizedPaths) {
				task.resetConfig(o, path)
				needsConfigUpdate.Store(true)
				needsUpdate.Store(true)
				configChanged = true
				break
			}
		}
		if configChanged {
			continue
		}
		for _, mapper := range task.resolved.ContentMappers() {
			if mapper.PackageDirectory == "" || mapper.ContributionID != "" {
				continue
			}
			if o.watchFileChanged(tspath.CombinePaths(mapper.PackageDirectory, "package.json"), normalizedPaths) {
				task.resetConfig(o, path)
				needsConfigUpdate.Store(true)
				needsUpdate.Store(true)
				configChanged = true
				break
			}
		}
		if configChanged {
			continue
		}

		rootChanged := false
		if task.contentMapperProject != nil {
			watchedFiles, err := task.contentMapperProject.WatchedFiles()
			if err != nil {
				task.contentMapperProjectErr = err
				task.resetStatus()
				needsUpdate.Store(true)
				rootChanged = true
			}
			for _, fileName := range watchedFiles {
				if o.watchFileChanged(fileName, normalizedPaths) {
					task.refreshContentMapperProject(o)
					task.resetStatus()
					needsUpdate.Store(true)
					rootChanged = true
					break
				}
			}
		}
		fileNames := task.resolved.FileNames()
		roots := collections.NewSetWithSizeHint[tspath.Path](len(fileNames))
		for _, file := range fileNames {
			fp := o.toPath(file)
			roots.Add(fp)
			if !rootChanged {
				if o.watchFileChanged(file, normalizedPaths) {
					task.resetStatus()
					needsUpdate.Store(true)
					rootChanged = true
				}
			}
		}

		if !rootChanged {
			task.buildInfoEntryMu.Lock()
			bi := task.buildInfoEntry
			task.buildInfoEntryMu.Unlock()
			if bi != nil && bi.buildInfo != nil {
				buildInfoDir := bi.directory()
				for _, fileName := range bi.buildInfo.FileNames {
					fp := o.toPath(o.resolveBuildInfoFileName(fileName, buildInfoDir))
					if roots.Has(fp) {
						continue
					}
					if o.watchFileChanged(o.resolveBuildInfoFileName(fileName, buildInfoDir), normalizedPaths) {
						task.resetStatus()
						needsUpdate.Store(true)
						break
					}
				}
				for packageJson := range bi.buildInfo.GetPackageJsons(buildInfoDir) {
					if o.watchFileChanged(packageJson, normalizedPaths) {
						task.resetStatus()
						needsUpdate.Store(true)
						break
					}
				}
				for packageJson := range bi.buildInfo.GetMissingPackageJsons(buildInfoDir) {
					if o.watchFileChanged(packageJson, normalizedPaths) {
						task.resetStatus()
						needsUpdate.Store(true)
						break
					}
				}
			}
			for _, packageJson := range task.packageJsons {
				if o.watchFileChanged(packageJson, normalizedPaths) {
					task.resetStatus()
					needsUpdate.Store(true)
					break
				}
			}
		}

		task.built = make(chan struct{})
		task.done = make(chan struct{})

		newConfig := task.resolved.ReloadFileNamesOfParsedCommandLine(o.host.FS())
		if !slices.Equal(task.resolved.FileNames(), newConfig.FileNames()) {
			o.host.resolvedReferences.store(path, newConfig)
			task.resolved = newConfig
			task.resetStatus()
			needsUpdate.Store(true)
		}
	}

	if !needsUpdate.Load() {
		opts := o.comparePathsOptions
		for eventPath := range changedPaths {
			if o.host.FS().DirectoryExists(eventPath) {
				if o.wm.IsPathUnderWatch(eventPath, opts) {
					o.rangeTask(func(path tspath.Path, task *BuildTask) {
						task.resetStatus()
						task.built = make(chan struct{})
						task.done = make(chan struct{})
					})
					needsUpdate.Store(true)
					break
				}
			}
		}
	}
}

func (o *Orchestrator) watchFileChanged(fileName string, changedPaths map[tspath.Path]fswatch.EventKind) bool {
	_, changed := changedPaths[o.toPath(fileName)]
	return changed
}

func (o *Orchestrator) computeDesiredWatches() ([]string, []string, map[string]bool) {
	realpath := func(name string) string { return o.wm.Realpath(name, o.host.FS()) }
	desiredDirs := watchmanager.NewDirWatchSet(o.comparePathsOptions)
	var watchFiles []string
	var logicalDirs []string

	for i := range o.order {
		config := o.order[i]
		path := o.toPath(config)
		task := o.getTask(path)
		watchFiles = append(watchFiles, task.config)
		watchFiles = append(watchFiles, task.seenFiles...)
		// Buildinfo stores compiler keys. Prefer the original read spellings
		// captured in this session, retaining every identity for alias fanout.
		originals := make(map[tspath.Path][]string, len(task.seenFiles))
		for _, name := range task.seenFiles {
			key := o.toPath(name)
			originals[key] = append(originals[key], name)
		}
		originalNames := func(name string) []string {
			if names := originals[o.toPath(name)]; len(names) != 0 {
				return names
			}
			return []string{name}
		}

		// Watch config file directory
		configDir := tspath.GetDirectoryPath(task.config)
		logicalDirs = append(logicalDirs, configDir)
		realConfigDir := realpath(configDir)
		desiredDirs.Set(realConfigDir, false)
		desiredDirs.Set(tspath.GetDirectoryPath(realpath(task.config)), false)

		if task.resolved == nil {
			continue
		}

		// Extended config file directories
		for _, cfgPath := range task.resolved.ExtendedSourceFiles() {
			watchFiles = append(watchFiles, cfgPath)
			realPath := realpath(cfgPath)
			dir := tspath.GetDirectoryPath(realPath)
			desiredDirs.Set(dir, false)
		}

		// Wildcard directories from tsconfig
		for dir, recursive := range task.resolved.WildcardDirectories() {
			watchFiles = append(watchFiles, dir)
			logicalDirs = append(logicalDirs, dir)
			realDir := realpath(dir)
			desiredDirs.Set(realDir, recursive)
		}

		// Input file directories not already covered
		for _, fileName := range task.resolved.FileNames() {
			absPath := tspath.GetNormalizedAbsolutePath(fileName, o.opts.Sys.GetCurrentDirectory())
			watchFiles = append(watchFiles, absPath)
			o.addProgramFileWatchDir(desiredDirs, tspath.GetDirectoryPath(absPath))
			o.addProgramFileWatchDir(desiredDirs, tspath.GetDirectoryPath(realpath(absPath)))
			for _, mapper := range task.resolved.ContentMappers() {
				if mapper.PackageDirectory == "" || mapper.ContributionID != "" {
					continue
				}
				manifestPath := tspath.CombinePaths(mapper.PackageDirectory, "package.json")
				watchFiles = append(watchFiles, manifestPath)
				dir := tspath.GetDirectoryPath(manifestPath)
				if !desiredDirs.Covered(dir) && watchmanager.CanWatchDirectory(dir) {
					desiredDirs.Set(dir, false)
				}
			}
		}
		if task.contentMapperProject != nil {
			watchedFiles, err := task.contentMapperProject.WatchedFiles()
			if err != nil {
				task.contentMapperProjectErr = err
			}
			for _, fileName := range watchedFiles {
				watchFiles = append(watchFiles, fileName)
				absPath := realpath(fileName)
				dir := tspath.GetDirectoryPath(absPath)
				if !desiredDirs.Covered(dir) && watchmanager.CanWatchDirectory(dir) {
					desiredDirs.Set(dir, false)
				}
			}
		}

		// Non-root dependency directories from buildinfo (e.g. node_modules .d.ts files).
		task.buildInfoEntryMu.Lock()
		bi := task.buildInfoEntry
		task.buildInfoEntryMu.Unlock()
		if bi != nil && bi.buildInfo != nil {
			buildInfoDir := bi.directory()
			roots := collections.NewSetFromItems(core.Map(task.resolved.FileNames(), o.toPath)...)
			for _, fileName := range bi.buildInfo.FileNames {
				for _, original := range originalNames(o.resolveBuildInfoFileName(fileName, buildInfoDir)) {
					watchFiles = append(watchFiles, original)
					absPath := realpath(original)
					fp := o.toPath(absPath)
					if roots.Has(fp) {
						continue
					}
					o.addProgramFileWatchDir(desiredDirs, tspath.GetDirectoryPath(original))
					o.addProgramFileWatchDir(desiredDirs, tspath.GetDirectoryPath(absPath))
				}
			}
			for packageJson := range bi.buildInfo.GetPackageJsons(buildInfoDir) {
				for _, original := range originalNames(packageJson) {
					watchFiles = append(watchFiles, original)
					o.addPackageJsonWatchDirs(desiredDirs, original)
				}
			}
			for packageJson := range bi.buildInfo.GetMissingPackageJsons(buildInfoDir) {
				for _, original := range originalNames(packageJson) {
					watchFiles = append(watchFiles, original)
					o.addPackageJsonWatchDirs(desiredDirs, original)
				}
			}
		}
		for _, packageJson := range task.packageJsons {
			for _, original := range originalNames(packageJson) {
				watchFiles = append(watchFiles, original)
				o.addPackageJsonWatchDirs(desiredDirs, original)
			}
		}
	}

	// Watch link replacements and target edits independently of buildinfo.
	for _, name := range watchFiles {
		if resolved := realpath(name); resolved != name {
			o.addWatchDir(desiredDirs, tspath.GetDirectoryPath(name))
			o.addWatchDir(desiredDirs, tspath.GetDirectoryPath(resolved))
		}
	}
	return watchFiles, logicalDirs, o.wm.ResolveDesiredDirs(desiredDirs.Dirs())
}

func (o *Orchestrator) addWatchDir(desiredDirs *watchmanager.DirWatchSet, dir string) {
	if !desiredDirs.Covered(dir) && watchmanager.CanWatchDirectory(dir) {
		desiredDirs.Set(dir, false)
	}
}

// addProgramFileWatchDir watches the directory of a program file at any depth, unlike addWatchDir, which guards lookup
// locations against watching something as generic as / or /home.
func (o *Orchestrator) addProgramFileWatchDir(desiredDirs *watchmanager.DirWatchSet, dir string) {
	if !desiredDirs.Covered(dir) {
		desiredDirs.Set(dir, false)
	}
}

func (o *Orchestrator) addPackageJsonWatchDirs(desiredDirs *watchmanager.DirWatchSet, packageJson string) {
	dir := tspath.GetDirectoryPath(packageJson)
	dirs := []string{dir}
	foundNodeModules := false
	for current := dir; ; {
		parent := tspath.GetDirectoryPath(current)
		if parent == "" || parent == current {
			break
		}
		dirs = append(dirs, parent)
		if tspath.GetBaseFileName(parent) == "node_modules" {
			foundNodeModules = true
			if grandparent := tspath.GetDirectoryPath(parent); grandparent != "" && grandparent != parent {
				dirs = append(dirs, grandparent)
			}
			break
		}
		current = parent
	}

	if !foundNodeModules {
		o.addWatchDir(desiredDirs, dir)
		return
	}
	for _, dir := range dirs {
		o.addWatchDir(desiredDirs, dir)
	}
}

func (o *Orchestrator) DoCycle() {
	o.wm.Lock()
	defer o.wm.Unlock()

	changes := o.wm.DrainEvents()
	realpathsChanged, err := o.wm.RefreshResolutions(changes)
	changedPaths, overflow := changes.Changes, changes.Overflow
	if err != nil {
		fmt.Fprintf(o.opts.Sys.Writer(), "%v\n", err)
		overflow = true
	}
	hasEvents := len(changedPaths) > 0 || overflow

	if !hasEvents {
		if o.wm.DebugLog != nil {
			fmt.Fprintf(o.wm.DebugLog, "[watch] DoCycle: no events, skipping\n")
		}
		return
	}

	var needsConfigUpdate atomic.Bool
	var needsUpdate atomic.Bool

	if realpathsChanged || overflow {
		o.resetCaches()
	}
	if overflow || realpathsChanged {
		// A new namespace can replace inputs without changing their timestamps.
		o.rangeTask(func(path tspath.Path, task *BuildTask) {
			task.resetConfig(o, path)
			task.built = make(chan struct{})
			task.done = make(chan struct{})
		})
		needsConfigUpdate.Store(true)
		needsUpdate.Store(true)
	} else {
		// Event-driven: check only tasks affected by changed paths
		o.checkTasksForEventChanges(changedPaths, &needsConfigUpdate, &needsUpdate)
	}

	if !needsUpdate.Load() {
		o.resetCaches()
		return
	}

	o.watchStatusReporter(ast.NewCompilerDiagnostic(diagnostics.File_change_detected_Starting_incremental_compilation))
	if needsConfigUpdate.Load() {
		// Generate new tasks
		o.GenerateGraphReusingOldTasks()
	}

	o.buildOrClean(realpathsChanged || overflow)
	o.updateWatch()
	watchFiles, logicalDirs, desiredDirs := o.computeDesiredWatches()
	if err := o.wm.ReconcileWatches(watchFiles, desiredDirs, o.host.FS(), logicalDirs...); err != nil {
		fmt.Fprintf(o.opts.Sys.Writer(), "%v\n", err)
		// Mark overflow so the next event triggers a full rebuild
		o.wm.ForceOverflow()
	}
	o.resetCaches()
}

func (o *Orchestrator) buildOrClean(force bool) tsc.CommandLineResult {
	return o.buildOrCleanOrder(o.order, force).Result
}

func (o *Orchestrator) buildOrCleanOrder(order []string, force bool) *OrchestratorResult {
	if !o.opts.Command.BuildOptions.Clean.IsTrue() && o.opts.Command.BuildOptions.Verbose.IsTrue() {
		o.createBuilderStatusReporter(nil)(ast.NewCompilerDiagnostic(
			diagnostics.Projects_in_this_build_Colon_0,
			strings.Join(core.Map(order, func(p string) string {
				return "\r\n    * " + o.relativeFileName(p)
			}), ""),
		))
	}
	var buildResult *OrchestratorResult = &OrchestratorResult{}
	if len(o.errors) == 0 {
		// var prevReporter *BuildTask
		// for _, config := range order {
		// 	task := o.getTask(o.toPath(config))
		// 	task.prevReporter = prevReporter
		// 	prevReporter = task
		// }
		buildResult.Statistics.Projects = len(order)
		// Builders pick up projects in scheduleOrder; results are reported in Order(), waiting for each project to finish
		reported := make(chan struct{})
		go func() {
			defer close(reported)
			for _, config := range order {
				path := o.toPath(config)
				task := o.getTask(path)
				<-task.built
				task.report(o, path, buildResult)
			}
		}()
		o.rangeTasks(order, func(path tspath.Path, task *BuildTask) {
			o.buildOrCleanProject(task, path, force)
		})
		<-reported
	} else {
		// Circularity errors prevent any project from being built
		buildResult.Result.Status = tsc.ExitStatusProjectReferenceCycle_OutputsSkipped
		reportDiagnostic := o.createDiagnosticReporter(nil)
		for _, err := range o.errors {
			reportDiagnostic(err)
		}
		buildResult.Errors = o.errors
	}
	buildResult.report(o)
	return buildResult
}

func (o *Orchestrator) rangeTask(f func(path tspath.Path, task *BuildTask)) {
	o.rangeTasks(o.order, f)
}

func (o *Orchestrator) rangeTasks(order []string, f func(path tspath.Path, task *BuildTask)) {
	numRoutines := 4
	if o.opts.Command.CompilerOptions.SingleThreaded.IsTrue() {
		numRoutines = 1
	} else if builders := o.opts.Command.BuildOptions.Builders; builders != nil {
		numRoutines = *builders
	}

	var currentTaskIndex atomic.Int64
	getNextTask := func() (tspath.Path, *BuildTask, bool) {
		index := int(currentTaskIndex.Add(1) - 1)
		if index >= len(order) {
			return "", nil, false
		}
		config := order[index]
		path := o.toPath(config)
		task := o.getTask(path)
		return path, task, true
	}
	runTask := func() {
		for path, task, ok := getNextTask(); ok; path, task, ok = getNextTask() {
			f(path, task)
		}
	}

	if numRoutines == 1 {
		runTask()
	} else {
		wg := core.NewWorkGroup(false)
		for range numRoutines {
			wg.Queue(runTask)
		}
		wg.RunAndWait()
	}
}

func (o *Orchestrator) buildOrCleanProject(task *BuildTask, path tspath.Path, force bool) {
	task.result = &taskResult{}
	task.result.reportStatus = o.createBuilderStatusReporter(task)
	task.result.diagnosticReporter = o.createDiagnosticReporter(task)
	if !o.opts.Command.BuildOptions.Clean.IsTrue() {
		task.buildProject(o, path, force)
	} else {
		task.cleanProject(o, path)
	}
	if o.opts.Testing == nil {
		// The program is only needed by Testing.OnProgram at report time; drop it now so a task
		// that has finished but is not yet reported does not keep its program alive.
		task.result.program = nil
	}
	close(task.built)
}

func (o *Orchestrator) getWriter(task *BuildTask) io.Writer {
	if task == nil {
		return o.opts.Sys.Writer()
	}
	return &task.result.builder
}

func (o *Orchestrator) createBuilderStatusReporter(task *BuildTask) tsc.DiagnosticReporter {
	return tsc.CreateBuilderStatusReporter(o.opts.Sys, o.getWriter(task), o.opts.Command.Locale(), o.opts.Command.CompilerOptions, o.opts.Testing)
}

func (o *Orchestrator) createDiagnosticReporter(task *BuildTask) tsc.DiagnosticReporter {
	return tsc.CreateDiagnosticReporter(o.opts.Sys, o.getWriter(task), o.opts.Command.Locale(), o.opts.Command.CompilerOptions)
}

func NewOrchestrator(opts Options) *Orchestrator {
	wm := watchmanager.NewWatchManager(opts.Sys.Writer(), opts.Sys.FS().DirectoryExists, opts.Sys.FS())
	orchestrator := &Orchestrator{
		opts: opts,
		comparePathsOptions: tspath.ComparePathsOptions{
			CurrentDirectory:          opts.Sys.GetCurrentDirectory(),
			UseCaseSensitiveFileNames: opts.Sys.FS().UseCaseSensitiveFileNames(),
		},
		tasks: &collections.SyncMap[tspath.Path, *BuildTask]{},
		wm:    wm,
	}
	orchestrator.host = &host{
		orchestrator: orchestrator,
		host: compiler.NewCachedFSCompilerHost(
			orchestrator.opts.Sys.GetCurrentDirectory(),
			orchestrator.opts.Sys.FS(),
			orchestrator.opts.Sys.DefaultLibraryPath(),
			nil,
			nil,
			nil,
		),
		mTimes: &collections.SyncMap[tspath.Path, time.Time]{},
	}
	if opts.Command.CompilerOptions.Watch.IsTrue() {
		orchestrator.watchStatusReporter = tsc.CreateWatchStatusReporter(opts.Sys, opts.Command.Locale(), opts.Command.CompilerOptions, opts.Testing)
		if t, ok := opts.Testing.(watchmanager.CommandLineTestingWithWatchBackend); ok {
			wm.SetBackend(t.WatchBackend())
		}
	} else {
		orchestrator.errorSummaryReporter = tsc.CreateReportErrorSummary(opts.Sys, opts.Command.Locale(), opts.Command.CompilerOptions)
	}
	return orchestrator
}
